package migrations

import (
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3" //nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"      //nolint
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
)

// DoMigrateDb performs DB migrations.
func DoMigrateDb(dbURL string) error {
	resources := bindata.Resource(AssetNames(),
		func(name string) ([]byte, error) {
			return Asset(name)
		})

	migrationData, err := bindata.WithInstance(resources)
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("go-bindata", migrationData, dbURL)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
