package main

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ardanlabs/conf/v3"
	"github.com/gin-gonic/gin"
	"github.com/hamidoujand/jumble/internal/auth"
	"github.com/hamidoujand/jumble/internal/debug"
	healthHandlers "github.com/hamidoujand/jumble/internal/domains/health/handler"
	"github.com/hamidoujand/jumble/internal/domains/user/bus"
	userHandlers "github.com/hamidoujand/jumble/internal/domains/user/handler"
	"github.com/hamidoujand/jumble/internal/domains/user/store/userdb"
	"github.com/hamidoujand/jumble/internal/metrics"
	"github.com/hamidoujand/jumble/internal/mid"
	"github.com/hamidoujand/jumble/internal/sqldb"
	"github.com/hamidoujand/jumble/pkg/keystore"
	"github.com/hamidoujand/jumble/pkg/logger"
	"github.com/hamidoujand/jumble/pkg/telemetry"
	"go.opentelemetry.io/otel"
)

//TODOS: add TLS support.

var build = "development"

func main() {
	traceIDFn := func(ctx context.Context) string {
		return telemetry.GetTraceID(ctx)
	}
	//os.Interrupt is going to be platform independent for example on UNIX it mapped to "syscall.SIGINT" on Windows to
	//something else, so we need a flexibility in here so we use os.Interrupt as well.
	//NOTE: you can skip "syscall.SIGINT" and only use os.Interrupt.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log := logger.New(os.Stdout, logger.LevelDebug, "jumble", traceIDFn)

	if err := run(ctx, log); err != nil {
		log.Error(ctx, "main failed to execute run", "err", err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, log *logger.Logger) error {
	log.Info(ctx, "run", "build", build, "GOMAXPROCS", runtime.GOMAXPROCS(0))

	//configuration
	cfg := struct {
		Web struct {
			ReadTimeout    time.Duration `conf:"default:10s"`
			WriteTimeout   time.Duration `conf:"default:30s"`
			IdleTimeout    time.Duration `conf:"default:120s"`
			ShutdownTimout time.Duration `conf:"default:120s"`
			DebugHost      string        `conf:"default:0.0.0.0:3000"`
			APIHost        string        `conf:"default:0.0.0.0:8000"`
			HealthCheck    string        `conf:"default:0.0.0.0:9000"`
		}

		DB struct {
			User     string `conf:"default:postgres"`
			Password string `conf:"default:postgres,mask"`
			//the app and db running in the same namespace, no need for cross namespace service discovery.
			Host        string `conf:"default:database:5432"`
			Name        string `conf:"default:postgres"`
			MaxIdleConn int    `conf:"default:0"` //needs load testing
			MaxOpenConn int    `conf:"default:0"`
			DisableTLS  bool   `conf:"default:true"`
		}

		Auth struct {
			Keys        string        `conf:"default:/etc/rsa-keys"`
			ActiveKey   string        `conf:"default:f7b7936a-1ca3-4015-811b-ec31b61e3071"`
			Issuer      string        `conf:"default:jumple project"`
			TokenMaxAge time.Duration `conf:"default:1h"`
		}

		Tempo struct {
			Host string `conf:"default:tempo:4318"`
			// Host        string  `conf:"default:dev"`
			ServiceName string  `conf:"default:jumble-service"`
			Probability float64 `conf:"default:1"`
		}
	}{}

	const prefix = "JUMBLE"
	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			//print help
			fmt.Println(help)
			return nil
		}
		return fmt.Errorf("parsing conf: %w", err)
	}

	out, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("conf to string: %w", err)
	}

	log.Info(ctx, "app configuration", "cfg", out)

	//==========================================================================
	//Debug Server
	go func() {
		log.Info(ctx, "debug server starting", "host", cfg.Web.DebugHost)
		if err := http.ListenAndServe(cfg.Web.DebugHost, debug.Register()); err != nil {
			log.Error(ctx, "failed to start debug server", "host", cfg.Web.DebugHost, "err", err.Error())
			return
		}
	}()

	expvar.NewString("build").Set(build)

	//==========================================================================
	// Database init
	db, err := sqldb.Open(sqldb.Config{
		User:         cfg.DB.User,
		Password:     cfg.DB.Password,
		Host:         cfg.DB.Host,
		Name:         cfg.DB.Name,
		MaxIdleConns: cfg.DB.MaxIdleConn,
		MaxOpenConns: cfg.DB.MaxOpenConn,
		DisableTLS:   cfg.DB.DisableTLS,
	})

	if err != nil {
		return fmt.Errorf("failed to open connection to database: %w", err)
	}

	defer db.Close()

	log.Info(ctx, "database initialized", "host", cfg.DB.Host)

	//==========================================================================
	// Trace init
	cleanup, err := telemetry.SetupOTelSDK(telemetry.Config{
		ServiceName: cfg.Tempo.ServiceName,
		Host:        cfg.Tempo.Host,
		Probability: cfg.Tempo.Probability,
		Build:       build,
	})

	if err != nil {
		return fmt.Errorf("setupOTelSDK: %w", err)
	}

	defer func() {
		cleanup(ctx)
	}()

	tracer := otel.Tracer(cfg.Tempo.ServiceName)

	log.Info(ctx, "tracer successfully initialized", "host", cfg.Tempo.Host, "probability", cfg.Tempo.Probability)

	//==========================================================================
	// Auth init

	ks := keystore.New()

	count, err := ks.LoadFromFileSystem(os.DirFS(cfg.Auth.Keys))
	if err != nil {
		return fmt.Errorf("loadFromFileSystem: %w", err)
	}

	log.Info(ctx, "loaded rsa keys into in-memory keystore", "count", count)

	//set the active kid
	if err := ks.SetActiveKey(cfg.Auth.ActiveKey); err != nil {
		return fmt.Errorf("setActiveKey: %w", err)
	}

	validActiveKid := ks.GetActiveKid()
	log.Info(ctx, "setting active KID was successfull", "activeKID", validActiveKid)

	store := userdb.NewStore(db, tracer)
	usrBus := bus.New(store)

	a := auth.New(ks, usrBus, cfg.Auth.Issuer)

	log.Info(ctx, "auth initialized", "key-count", count)

	//==========================================================================
	// Metrics init
	m := metrics.New()

	//==========================================================================
	// Router init
	r := gin.New()

	//middleare stack
	r.Use(mid.Telemetry(tracer))
	r.Use(mid.Logger(log))
	r.Use(mid.Metrics(m))
	r.Use(mid.Panic(log))

	userHandlers.RegisterRoutes(userHandlers.Conf{
		UserBus:     usrBus,
		Auth:        a,
		Kid:         validActiveKid,
		Issuer:      cfg.Auth.Issuer,
		TokenMaxAge: cfg.Auth.TokenMaxAge,
		Tracer:      tracer,
		Logger:      log,
		Router:      r,
	})

	healthCheckMux := healthHandlers.RegisterRoutes(healthHandlers.Conf{
		DB:    db,
		Log:   log,
		Build: build,
	})

	//health check server
	go func() {
		log.Info(ctx, "health check server is running", "host", cfg.Web.HealthCheck)
		if err := http.ListenAndServe(cfg.Web.HealthCheck, healthCheckMux); err != nil {
			log.Error(ctx, "health check server failed", "err", err)
			return
		}
	}()

	//==========================================================================
	// API Server
	server := http.Server{
		Addr:         cfg.Web.APIHost,
		Handler:      r,
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		IdleTimeout:  cfg.Web.IdleTimeout,
		ErrorLog:     log.StdLogger(logger.LevelError),
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
	}

	serverErrs := make(chan error, 1)

	go func() {
		log.Info(ctx, "API server starting", "host", cfg.Web.APIHost)
		if err := server.ListenAndServe(); err != nil {
			serverErrs <- fmt.Errorf("listenAndServe: %w", err)
		}
	}()

	select {
	case err := <-serverErrs:
		//something went wrong when starting the server
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		log.Info(ctx, "server received a shutdown signal")
		defer log.Info(ctx, "server completed the shutdown process")

		shutdownCtx, cancel := context.WithTimeout(ctx, cfg.Web.ShutdownTimout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			server.Close()
			return fmt.Errorf("failed to gracefully shutdown the server: %w", err)
		}
	}
	return nil
}
