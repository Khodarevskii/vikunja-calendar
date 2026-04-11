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

	"xorm.io/xorm"
)

// recalculateTaskPercentDoneFromSubtasks recomputes the percent_done column of
// the task with the given id from its real subtasks (task_relations with kind
// "subtask"). Each subtask contributes an equal share; percent_done is the
// fraction of subtasks currently marked as done.
//
// If the task has no subtasks, its percent_done is left untouched so users can
// still set it manually.
func recalculateTaskPercentDoneFromSubtasks(s *xorm.Session, parentID int64) error {
	relations := []*TaskRelation{}
	err := s.
		Where("task_id = ? AND relation_kind = ?", parentID, RelationKindSubtask).
		Find(&relations)
	if err != nil {
		return err
	}
	if len(relations) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(relations))
	for _, r := range relations {
		ids = append(ids, r.OtherTaskID)
	}

	subtasks := []*Task{}
	if err := s.In("id", ids).Find(&subtasks); err != nil {
		return err
	}
	if len(subtasks) == 0 {
		return nil
	}

	var doneCount int
	for _, st := range subtasks {
		if st.Done {
			doneCount++
		}
	}

	newPercent := math.Round((float64(doneCount)/float64(len(subtasks)))*100) / 100

	parent, err := GetTaskByIDSimple(s, parentID)
	if err != nil {
		return err
	}
	if math.Abs(newPercent-parent.PercentDone) < 0.001 {
		return nil
	}

	// Use a map so xorm writes the value even when it is the zero value (0).
	_, err = s.ID(parentID).
		Cols("percent_done").
		Update(map[string]interface{}{"percent_done": newPercent})
	return err
}

// recalculateParentTasksPercentDone finds every real parent of the given task
// (task relations of kind "subtask" pointing at this task) and recomputes
// each parent's percent_done based on the done state of its subtasks.
func recalculateParentTasksPercentDone(s *xorm.Session, subtaskID int64) error {
	parents := []*TaskRelation{}
	err := s.
		Where("other_task_id = ? AND relation_kind = ?", subtaskID, RelationKindSubtask).
		Find(&parents)
	if err != nil {
		return err
	}

	seen := make(map[int64]bool, len(parents))
	for _, p := range parents {
		if seen[p.TaskID] {
			continue
		}
		seen[p.TaskID] = true
		if err := recalculateTaskPercentDoneFromSubtasks(s, p.TaskID); err != nil {
			return err
		}
	}
	return nil
}
