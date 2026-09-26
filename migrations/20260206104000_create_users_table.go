package migrations

import (
	"context"
	"github.com/sllt/pi/pkg/pi/migration"
)

const createUsersTable = `CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id TEXT UNIQUE NOT NULL,
	password TEXT NOT NULL,
	email TEXT UNIQUE NOT NULL,
	created_at DATETIME,
	updated_at DATETIME
);`

func createUsersTableMigration() migration.Migrate {
	return migration.Migrate{
		Name: "create_users_table",
		UpContext: func(ctx context.Context, d migration.Datasource) error {
			_, err := d.SQL.ExecContext(ctx, createUsersTable)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
