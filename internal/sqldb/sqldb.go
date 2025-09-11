package sqldb

import (
	"context"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"net/url"
	"time"
)

type Config struct {
	User         string
	Password     string
	Host         string
	Name         string
	Schema       string
	MaxIdleConns int
	MaxOpenConns int
	DisableTLS   bool
}

func Open(cfg Config) (*sqlx.DB, error) {
	sslmode := "require"
	if cfg.DisableTLS {
		sslmode = "disable"
	}

	q := make(url.Values)
	q.Set("sslmode", sslmode)
	q.Set("timezone", "utc")

	if cfg.Schema != "" {
		q.Set("search_path", cfg.Schema)
	}

	uri := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     cfg.Host,
		Path:     cfg.Name,
		RawQuery: q.Encode(),
	}

	db, err := sqlx.Open("pgx", uri.String())
	if err != nil {
		return nil, fmt.Errorf("open connection: %w", err)
	}
	return db, nil
}

func ConnCheck(ctx context.Context, db *sqlx.DB) error {
	//make sure the ctx is with deadline
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		//default is 10s, counting slow machines
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}

	for attempt := 1; ; attempt++ {
		pingErr := db.PingContext(ctx)
		if pingErr == nil {
			break
		}

		//otherwise we wait
		d := time.Duration(attempt) * 100 * time.Millisecond
		time.Sleep(d)

		//check the ctx when wake up
		if ctx.Err() != nil {
			return fmt.Errorf("deadline exceeded: %s: %w", ctx.Err(), pingErr)
		}
	}

	//we got here we have a connection, we just need to check the engine
	var res bool
	if err := db.QueryRowContext(ctx, "SELECT TRUE").Scan(&res); err != nil {
		return fmt.Errorf("check sql engine: %w", err)
	}

	return nil
}
