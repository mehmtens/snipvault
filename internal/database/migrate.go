package database

import (
	"context"
	_ "embed"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/001_create_pastes.sql
var createPastesSQL string

//go:embed migrations/002_create_users.sql
var createUsersSQL string

//go:embed migrations/003_add_paste_owner.sql
var addPasteOwnerSQL string

//go:embed migrations/004_add_paste_expiration.sql
var addPasteExpirationSQL string

//go:embed migrations/005_add_paste_updated_at.sql
var addPasteUpdatedAtSQL string

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	for _, migration := range []string{createPastesSQL, createUsersSQL, addPasteOwnerSQL, addPasteExpirationSQL, addPasteUpdatedAtSQL} {
		if _, err := pool.Exec(ctx, migration); err != nil {
			return err
		}
	}
	return nil
}
