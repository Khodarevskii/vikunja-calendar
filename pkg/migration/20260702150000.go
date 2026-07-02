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
	"time"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

// Two feedback-related columns on tasks and a join table listing users
// from whom feedback is expected.

type tasks20260702150000 struct {
	FeedbackRequested bool  `xorm:"NOT NULL DEFAULT false" json:"feedback_requested"`
	FeedbackManagerID int64 `xorm:"BIGINT INDEX null default null" json:"feedback_manager_id"`
}

func (tasks20260702150000) TableName() string {
	return "tasks"
}

type taskFeedbackReviewer20260702150000 struct {
	ID      int64     `xorm:"bigint autoincr not null unique pk" json:"id"`
	TaskID  int64     `xorm:"bigint INDEX not null" json:"task_id"`
	UserID  int64     `xorm:"bigint INDEX not null" json:"user_id"`
	Created time.Time `xorm:"created not null" json:"created"`
}

func (taskFeedbackReviewer20260702150000) TableName() string {
	return "task_feedback_reviewers"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260702150000",
		Description: "Add feedback columns to tasks and task_feedback_reviewers join table",
		Migrate: func(tx *xorm.Engine) error {
			return tx.Sync2(tasks20260702150000{}, taskFeedbackReviewer20260702150000{})
		},
		Rollback: func(tx *xorm.Engine) error {
			return nil
		},
	})
}
