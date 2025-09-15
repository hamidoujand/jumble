package migrate

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
)

//go:embed sql/*.sql
var migrationFiles embed.FS

func Migrate(db *sqlx.DB, dbname string) error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("creating dirver: %w", err)
	}

	source, err := iofs.New(migrationFiles, "sql") // "sql" is the prefix from the path "sql/init.sql"
	if err != nil {
		return fmt.Errorf("creating source from fs: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, dbname, driver)
	if err != nil {
		return fmt.Errorf("creating migration instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration up: %w", err)
	}

	return nil
}
