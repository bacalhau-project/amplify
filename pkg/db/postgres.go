package db

import (
	"bytes"
	"database/sql"
	"embed"
	"io/fs"

	"github.com/rs/zerolog/log"
	migrate "github.com/rubenv/sql-migrate"
)

// TODO: Need to wrap/parse database errors

// migrations holds the database migrations
//
//go:embed migrations/*
var embedFS embed.FS

func NewPostgresDB(connStr string) (Persistence, error) {
	pdb, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	list, err := fs.Glob(embedFS, "migrations/*.sql")
	if err != nil {
		return nil, err
	}
	migrations := &migrate.MemoryMigrationSource{}
	for _, f := range list {
		b, err := fs.ReadFile(embedFS, f)
		if err != nil {
			return nil, err
		}
		m, err := migrate.ParseMigration(f, bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		migrations.Migrations = append(migrations.Migrations, m)
	}
	migrationCount, err := migrate.Exec(pdb, "postgres", migrations, migrate.Up)
	if err != nil {
		return nil, err
	}
	log.Debug().Int("migrations", migrationCount).Msg("migrations applied")
	return New(pdb), nil
}
