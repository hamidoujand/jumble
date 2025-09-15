package cli

import (
	"fmt"

	"github.com/hamidoujand/jumble/internal/migrate"
	"github.com/hamidoujand/jumble/internal/sqldb"
	"github.com/spf13/cobra"
)

// Database required configs
var (
	dbuser string
	dbpass string
	host   string
	name   string
)

func init() {
	rootCommand.AddCommand(migrateCommand)

	//database connection flags
	migrateCommand.Flags().StringVarP(&dbuser, "user", "u", "postgres", "Database username required.")
	migrateCommand.Flags().StringVarP(&dbpass, "pass", "p", "postgres", "Database password required.")
	migrateCommand.Flags().StringVar(&host, "host", "localhost:5432", "Database host:port required.")
	migrateCommand.Flags().StringVarP(&name, "name", "n", "postgres", "Database name to run migration againt required.")

	// since we have defaults, comment these.
	// migrateCommand.MarkFlagRequired("user")
	// migrateCommand.MarkFlagRequired("pass")
	// migrateCommand.MarkFlagRequired("host")
	// migrateCommand.MarkFlagRequired("name")
}

var migrateCommand = &cobra.Command{
	Use:   "migrate",
	Short: "performs migration",
	Long: `Execute database migrations.

Examples:
  dmin migrate --user=myuser --pass=mypass --host=localhost:5432 --dbname=mydbe`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if dbuser == "" {
			return fmt.Errorf("database user is required (--user)")
		}

		if dbpass == "" {
			return fmt.Errorf("database password is required (--pass)")
		}

		if host == "" {
			return fmt.Errorf("database host is required (--host)")
		}

		if name == "" {
			return fmt.Errorf("database name is required (--name)")
		}

		return nil
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("applying migrations...")

		db, err := sqldb.Open(sqldb.Config{
			User:       dbuser,
			Password:   dbpass,
			Host:       host,
			Name:       name,
			DisableTLS: true,
		})

		if err != nil {
			return fmt.Errorf("open connection: %w", err)
		}

		defer db.Close()

		if err := migrate.Migrate(db, name); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}

		fmt.Println("migration completed!")
		return nil
	},
}
