// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package migration

import (
	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID: "20260422130000",
		Description: "Convert tasks.checklist_items from json to jsonb on postgres so SELECT DISTINCT on tasks works",
		Migrate: func(tx *xorm.Engine) error {
			if tx.Dialect().URI().DBType != schemas.POSTGRES {
				return nil
			}
			_, err := tx.Exec(`ALTER TABLE tasks ALTER COLUMN checklist_items TYPE jsonb USING COALESCE(checklist_items::jsonb, '[]'::jsonb)`)
			return err
		},
		Rollback: func(tx *xorm.Engine) error {
			if tx.Dialect().URI().DBType != schemas.POSTGRES {
				return nil
			}
			_, err := tx.Exec(`ALTER TABLE tasks ALTER COLUMN checklist_items TYPE json USING checklist_items::json`)
			return err
		},
	})
}
