package dbtest

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/hamidoujand/jumble/internal/migrate"
	"github.com/hamidoujand/jumble/internal/sqldb"
	"github.com/hamidoujand/jumble/pkg/docker"
	"github.com/jmoiron/sqlx"
)

func CreateDBContainer() (docker.Container, error) {
	image := "postgres:17"
	name := "bustest"
	port := "5432"
	dockerArgs := []string{"-e", "POSTGRES_PASSWORD=postgres"}
	appArgs := []string{"-c", "log_statement=all"}

	c, err := docker.StartContainer(image, name, port, dockerArgs, appArgs)
	if err != nil {
		return docker.Container{}, fmt.Errorf("startContainer: %w", err)
	}
	return c, nil
}

func New(t *testing.T, c docker.Container, testName string) *sqlx.DB {

	t.Logf("Name:\t%s\n", c.Name)
	t.Logf("HostPort:\t%s\n", c.HostPort)

	master, err := sqldb.Open(sqldb.Config{
		User:       "postgres",
		Password:   "postgres",
		Host:       c.HostPort,
		Name:       "postgres",
		DisableTLS: true,
	})

	if err != nil {
		t.Fatalf("open conn: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()

	if err := sqldb.ConnCheck(ctx, master); err != nil {
		t.Fatalf("connCheck: %s", err)
	}

	//because of restrictions on DB name from posgres, we can not use any random string
	const letters = "abcdefghijklmnopqrstuvwxyz"
	bs := make([]byte, 4)
	for i := range bs {
		bs[i] = letters[rand.IntN(len(letters))]
	}

	dbName := string(bs)

	t.Logf("creating database: %s\n", dbName)
	if _, err := master.ExecContext(ctx, "CREATE DATABASE "+dbName); err != nil {
		t.Fatalf("execContext: %s", err)
	}

	//create a new client
	db, err := sqldb.Open(sqldb.Config{
		User:       "postgres",
		Password:   "postgres",
		Host:       c.HostPort,
		Name:       dbName,
		DisableTLS: true,
	})
	if err != nil {
		t.Fatalf("open:%s:%s", dbName, err)
	}

	t.Logf("running migrations against: %s\n", dbName)
	if err := migrate.Migrate(db, dbName); err != nil {
		t.Logf("logs for: %s\n%s\n", c.Name, docker.DumpContainerLogs(c.Name))
		t.Fatalf("migration failed: %s", err)
	}

	t.Cleanup(func() {
		t.Helper()

		//terminate conn to the client db so we can delete it .
		const q = `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`
		if _, err := master.ExecContext(context.Background(), q, dbName); err != nil {
			t.Fatalf("terminating conn for %s: %s", dbName, err)
		}

		t.Logf("Drop Database %s\n", dbName)
		_ = db.Close()

		if _, err := master.ExecContext(context.Background(), "DROP DATABASE "+dbName); err != nil {
			t.Fatalf("dropping database %s: %s", dbName, err)
		}

		_ = master.Close()
	})

	return db
}
