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

// progressItem abstracts a single contributor (real subtask or checklist item)
// to the percent_done calculation of a parent task.
type progressItem struct {
	Weight float64
	Done   bool
}

// calculateWeightedProgress computes a percent_done value in the range [0, 1]
// from a list of progress items, applying the weight rules:
//
//  1. Items with Weight > 0 contribute exactly that percentage.
//  2. Unweighted items split the remaining (100 - sumOfSetWeights) equally.
//  3. If sumOfSetWeights > 100, all items split 100 equally regardless of the
//     individual weights.
//  4. If no items are present, the result is 0.
//
// The special "split equally regardless of weight" override used when both
// real subtasks AND checklist items are present is handled by the caller by
// passing items with Weight = 0.
func calculateWeightedProgress(items []progressItem) float64 {
	if len(items) == 0 {
		return 0
	}

	var weightSum float64
	var weightedCount int
	for _, it := range items {
		if it.Weight > 0 {
			weightSum += it.Weight
			weightedCount++
		}
	}

	// Overflow → split equally, weights are ignored.
	if weightSum > 100 {
		var doneCount int
		for _, it := range items {
			if it.Done {
				doneCount++
			}
		}
		return math.Round((float64(doneCount)/float64(len(items)))*100) / 100
	}

	unweighted := len(items) - weightedCount
	var autoShare float64
	if unweighted > 0 {
		autoShare = (100 - weightSum) / float64(unweighted)
	}

	var done float64
	for _, it := range items {
		share := it.Weight
		if share == 0 {
			share = autoShare
		}
		if it.Done {
			done += share
		}
	}

	return math.Round(done) / 100
}

// recalculateTaskPercentDone recomputes the percent_done field for the task
// with the given id based on its real subtasks (task relations with kind
// "subtask") and its embedded checklist items.
//
// Rules (matching the user's spec):
//   - When a parent has both real subtasks AND checklist items, every item
//     contributes an equal share of 100% (weights ignored).
//   - Otherwise, the weight rules from calculateWeightedProgress are applied to
//     whichever group is present (or both independently).
//
// The task's percent_done is only updated if the computed value differs from
// what is already stored.
func recalculateTaskPercentDone(s *xorm.Session, parentID int64) error {
	parent, err := GetTaskByIDSimple(s, parentID)
	if err != nil {
		return err
	}

	// Collect real subtasks via task_relations.
	relations := []*TaskRelation{}
	err = s.Where("task_id = ? AND relation_kind = ?", parentID, RelationKindSubtask).
		Find(&relations)
	if err != nil {
		return err
	}

	subtaskIDs := make([]int64, 0, len(relations))
	for _, r := range relations {
		subtaskIDs = append(subtaskIDs, r.OtherTaskID)
	}

	subtasks := []*Task{}
	if len(subtaskIDs) > 0 {
		err = s.In("id", subtaskIDs).Find(&subtasks)
		if err != nil {
			return err
		}
	}

	checklistItems := parent.ChecklistItems

	// Nothing to compute from.
	if len(subtasks) == 0 && len(checklistItems) == 0 {
		return nil
	}

	var newPercent float64

	switch {
	case len(subtasks) > 0 && len(checklistItems) > 0:
		// Both present: every item gets an equal share of 100% regardless of
		// its individual weight.
		items := make([]progressItem, 0, len(subtasks)+len(checklistItems))
		for _, st := range subtasks {
			items = append(items, progressItem{Weight: 0, Done: st.Done})
		}
		for _, ci := range checklistItems {
			items = append(items, progressItem{Weight: 0, Done: ci.Done})
		}
		total := len(items)
		var doneCount int
		for _, it := range items {
			if it.Done {
				doneCount++
			}
		}
		newPercent = math.Round((float64(doneCount)/float64(total))*100) / 100

	case len(subtasks) > 0:
		// Only real subtasks. Real subtasks don't carry a weight of their own
		// (weights live on checklist items), so they share equally.
		items := make([]progressItem, 0, len(subtasks))
		for _, st := range subtasks {
			items = append(items, progressItem{Weight: 0, Done: st.Done})
		}
		newPercent = calculateWeightedProgress(items)

	case len(checklistItems) > 0:
		items := make([]progressItem, 0, len(checklistItems))
		for _, ci := range checklistItems {
			items = append(items, progressItem{Weight: ci.Weight, Done: ci.Done})
		}
		newPercent = calculateWeightedProgress(items)
	}

	if math.Abs(newPercent-parent.PercentDone) < 0.001 {
		return nil
	}

	_, err = s.ID(parent.ID).Cols("percent_done").Update(&Task{PercentDone: newPercent})
	return err
}

// recalculateParentTasksPercentDone finds all real parent tasks of the given
// subtask and updates their percent_done. It is a no-op if the task has no
// parents.
func recalculateParentTasksPercentDone(s *xorm.Session, subtaskID int64) error {
	parents := []*TaskRelation{}
	err := s.Where("other_task_id = ? AND relation_kind = ?", subtaskID, RelationKindSubtask).
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
		if err := recalculateTaskPercentDone(s, p.TaskID); err != nil {
			return err
		}
	}
	return nil
}
