package main

import (
	"flag"

	"github.com/go-faster/errors"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var storagePath, migrationPath string
	flag.StringVar(&storagePath, "storage", "", "path to storage")
	flag.StringVar(&migrationPath, "migrations", "", "path to migrations")
	flag.Parse()

	if storagePath == "" {
		panic("storage path is required")
	}
	if migrationPath == "" {
		panic("migration path is required")
	}

	m, err := migrate.New("file://"+migrationPath, storagePath)
	if err != nil {
		panic(err)
	}
	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return
		}
		panic(err)
	}
}
