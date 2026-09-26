package migrations

import "github.com/sllt/pi/pkg/pi/migration"

const createUserProfilesTable = `CREATE TABLE IF NOT EXISTS user_profiles (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id TEXT UNIQUE NOT NULL,
	nickname TEXT NOT NULL DEFAULT '',
	created_at DATETIME,
	updated_at DATETIME,
	FOREIGN KEY(user_id) REFERENCES users(user_id)
);`

func createUserProfilesTableMigration() migration.Migrate {
	return migration.Migrate{
		UP: func(d migration.Datasource) error {
			_, err := d.SQL.Exec(createUserProfilesTable)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
