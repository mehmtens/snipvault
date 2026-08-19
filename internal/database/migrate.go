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

//go:embed migrations/006_add_account_security.sql
var addAccountSecuritySQL string

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	connection, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer connection.Release()

	if _, err := connection.Exec(ctx, `SELECT pg_advisory_lock(739_628_155)`); err != nil {
		return err
	}
	defer func() { _, _ = connection.Exec(context.Background(), `SELECT pg_advisory_unlock(739_628_155)`) }()

	for _, migration := range []string{createPastesSQL, createUsersSQL, addPasteOwnerSQL, addPasteExpirationSQL, addPasteUpdatedAtSQL, addAccountSecuritySQL} {
		if _, err := connection.Exec(ctx, migration); err != nil {
			return err
		}
	}
	return nil
}
