// This is auto-generated style file for migration registry.
package migrations

import "github.com/sllt/pi/pkg/pi/migration"

func All() map[int64]migration.Migrate {
	return map[int64]migration.Migrate{
		20260206104000: createUsersTableMigration(),
		20260501120000: createUserProfilesTableMigration(),
	}
}
