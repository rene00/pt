package cli

import (
	"database/sql"
	"fmt"
	"pt/db/migrations"

	_ "github.com/mattn/go-sqlite3" //nolint
	"github.com/spf13/cobra"
)

func initCmd(cli *cli) *cobra.Command {
	var cmd = &cobra.Command{
		Use: "init",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := migrations.DoMigrateDb(fmt.Sprintf("sqlite3://%s", cli.config.DBFile)); err != nil {
				return err
			}

			db, err := sql.Open("sqlite3", cli.config.DBFile)
			if err != nil {
				return err
			}
			defer db.Close()

			return nil
		},
	}

	return cmd
}
