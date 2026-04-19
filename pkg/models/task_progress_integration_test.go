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

package models

import (
	"math"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

// assertTaskState reloads the task from the session and checks done and
// percent_done values.
func assertTaskState(t *testing.T, s *xorm.Session, id int64, wantDone bool, wantPercent float64) {
	t.Helper()
	got := &Task{}
	exists, err := s.ID(id).Get(got)
	require.NoError(t, err)
	require.True(t, exists, "task %d should exist", id)
	assert.Equal(t, wantDone, got.Done, "task %d: done mismatch", id)
	assert.LessOrEqual(t, math.Abs(got.PercentDone-wantPercent), 0.001,
		"task %d: percent_done mismatch, want %v got %v", id, wantPercent, got.PercentDone)
}

// TestRecalculateTaskPercentDone_NestedCascade verifies that marking a deeply
// nested subtask as done propagates through every ancestor level, not just
// the immediate parent. Covers the bug where Task 1 → 2 → 3 did not update
// Task 1's progress when Task 3 was completed.
func TestRecalculateTaskPercentDone_NestedCascade(t *testing.T) {
	usr := &user.User{ID: 1}

	t.Run("linear chain of four tasks", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// Create four tasks chained as t1 → t2 → t3 → t4.
		tasks := make([]*Task, 4)
		for i := range tasks {
			tasks[i] = &Task{Title: "chain", ProjectID: 1}
			require.NoError(t, tasks[i].Create(s, usr))
		}
		for i := 0; i < len(tasks)-1; i++ {
			rel := TaskRelation{
				TaskID:       tasks[i].ID,
				OtherTaskID:  tasks[i+1].ID,
				RelationKind: RelationKindSubtask,
			}
			require.NoError(t, rel.Create(s, usr))
		}

		tasks[3].Done = true
		require.NoError(t, tasks[3].Update(s, usr))

		for _, task := range tasks {
			assertTaskState(t, s, task.ID, true, 1.0)
		}

		tasks[3].Done = false
		require.NoError(t, tasks[3].Update(s, usr))
		for _, task := range tasks {
			assertTaskState(t, s, task.ID, false, 0)
		}

		require.NoError(t, s.Commit())
	})

	t.Run("nested subtask lifts grandparent partial progress", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		//   t1 → {t2, t2b}
		//   t2 → t3
		t1 := &Task{Title: "t1", ProjectID: 1}
		t2 := &Task{Title: "t2", ProjectID: 1}
		t2b := &Task{Title: "t2b", ProjectID: 1}
		t3 := &Task{Title: "t3", ProjectID: 1}
		require.NoError(t, t1.Create(s, usr))
		require.NoError(t, t2.Create(s, usr))
		require.NoError(t, t2b.Create(s, usr))
		require.NoError(t, t3.Create(s, usr))

		rels := []TaskRelation{
			{TaskID: t1.ID, OtherTaskID: t2.ID, RelationKind: RelationKindSubtask},
			{TaskID: t1.ID, OtherTaskID: t2b.ID, RelationKind: RelationKindSubtask},
			{TaskID: t2.ID, OtherTaskID: t3.ID, RelationKind: RelationKindSubtask},
		}
		for i := range rels {
			require.NoError(t, rels[i].Create(s, usr))
		}

		t3.Done = true
		require.NoError(t, t3.Update(s, usr))

		assertTaskState(t, s, t3.ID, true, 0)
		assertTaskState(t, s, t2.ID, true, 1.0)
		assertTaskState(t, s, t2b.ID, false, 0)
		assertTaskState(t, s, t1.ID, false, 0.5)

		require.NoError(t, s.Commit())
	})

	t.Run("nested subtask contributes exactly its weight", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		//   t1 → {t2 (weight=80), t2b (auto)}
		//   t2 → t3
		t1 := &Task{Title: "t1", ProjectID: 1}
		t2 := &Task{Title: "t2", ProjectID: 1, SubtaskWeight: 80}
		t2b := &Task{Title: "t2b", ProjectID: 1}
		t3 := &Task{Title: "t3", ProjectID: 1}
		require.NoError(t, t1.Create(s, usr))
		require.NoError(t, t2.Create(s, usr))
		require.NoError(t, t2b.Create(s, usr))
		require.NoError(t, t3.Create(s, usr))

		rels := []TaskRelation{
			{TaskID: t1.ID, OtherTaskID: t2.ID, RelationKind: RelationKindSubtask},
			{TaskID: t1.ID, OtherTaskID: t2b.ID, RelationKind: RelationKindSubtask},
			{TaskID: t2.ID, OtherTaskID: t3.ID, RelationKind: RelationKindSubtask},
		}
		for i := range rels {
			require.NoError(t, rels[i].Create(s, usr))
		}

		t3.Done = true
		require.NoError(t, t3.Update(s, usr))

		assertTaskState(t, s, t2.ID, true, 1.0)
		assertTaskState(t, s, t1.ID, false, 0.8)

		require.NoError(t, s.Commit())
	})
}
