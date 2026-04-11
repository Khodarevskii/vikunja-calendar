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
)

type taskChecklistItem20260411120000 struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Weight     float64 `json:"weight"`
	Done       bool    `json:"done"`
	AssigneeID int64   `json:"assignee_id"`
	Position   float64 `json:"position"`
}

type tasks20260411120000 struct {
	ChecklistItems []*taskChecklistItem20260411120000 `xorm:"json null" json:"checklist_items"`
}

func (tasks20260411120000) TableName() string {
	return "tasks"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260411120000",
		Description: "Add checklist_items column to tasks for decomposition checklists",
		Migrate: func(tx *xorm.Engine) error {
			return tx.Sync2(tasks20260411120000{})
		},
		Rollback: func(tx *xorm.Engine) error {
			return nil
		},
	})
}
