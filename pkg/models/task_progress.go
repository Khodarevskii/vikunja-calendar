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
	"time"

	"xorm.io/xorm"
)

// progressRelationKinds lists the relation kinds that contribute to a task's
// percent_done when they appear as outgoing relations (task_id → other_task_id).
// Subtasks are the classic case; "related" tasks also count now.
// Stored as []interface{} so we can pass it directly to xorm's In() method.
var progressRelationKinds = []interface{}{
	RelationKindSubtask,
	RelationKindRelated,
}

// isProgressRelationKind returns true if the given kind contributes to
// percent_done calculations.
func isProgressRelationKind(kind RelationKind) bool {
	for _, k := range progressRelationKinds {
		if k == kind {
			return true
		}
	}
	// Also check the inverse: if someone creates a "parenttask" relation,
	// the real parent (OtherTaskID) needs recalculation because the forward
	// direction (subtask) is progress-contributing.
	inverse := getInverseRelation(kind)
	for _, k := range progressRelationKinds {
		if k == inverse {
			return true
		}
	}
	return false
}

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
// with the given id based on its related tasks (subtasks and "related" tasks)
// and its embedded checklist items.
//
// Both related tasks (via Task.SubtaskWeight) and checklist items (via
// TaskChecklistItem.Weight) may carry an optional weight. The weight rules
// from calculateWeightedProgress are applied to the combined list:
//
//   - Items with Weight > 0 contribute exactly that percentage.
//   - Unweighted items (Weight == 0) split whatever is left of 100% equally.
//     For example: if a parent has three related tasks and one has a weight
//     of 10, the remaining two split 90% → 45% each.
//   - If the sum of set weights exceeds 100, every item gets an equal share.
//
// When the computed percent_done reaches 1.0 the parent task is automatically
// marked as done. If it drops below 1.0 and the task was previously at 100%
// (auto-done), it is marked as not done again.
//
// The task's percent_done is only updated if the computed value differs from
// what is already stored.
func recalculateTaskPercentDone(s *xorm.Session, parentID int64) error {
	parent, err := GetTaskByIDSimple(s, parentID)
	if err != nil {
		return err
	}

	// Collect related tasks via task_relations for all progress-contributing
	// relation kinds (subtask, related).
	relations := []*TaskRelation{}
	err = s.Where("task_id = ?", parentID).
		In("relation_kind", progressRelationKinds).
		Find(&relations)
	if err != nil {
		return err
	}

	relatedIDs := make([]int64, 0, len(relations))
	for _, r := range relations {
		relatedIDs = append(relatedIDs, r.OtherTaskID)
	}

	relatedTasks := []*Task{}
	if len(relatedIDs) > 0 {
		err = s.In("id", relatedIDs).Find(&relatedTasks)
		if err != nil {
			return err
		}
	}

	checklistItems := parent.ChecklistItems

	// Nothing to compute from.
	if len(relatedTasks) == 0 && len(checklistItems) == 0 {
		return nil
	}

	items := make([]progressItem, 0, len(relatedTasks)+len(checklistItems))
	for _, rt := range relatedTasks {
		items = append(items, progressItem{Weight: rt.SubtaskWeight, Done: rt.Done})
	}
	for _, ci := range checklistItems {
		items = append(items, progressItem{Weight: ci.Weight, Done: ci.Done})
	}

	newPercent := calculateWeightedProgress(items)

	changed := math.Abs(newPercent-parent.PercentDone) >= 0.001

	if !changed {
		return nil
	}

	_, err = s.ID(parent.ID).Cols("percent_done").Update(&Task{PercentDone: newPercent})
	if err != nil {
		return err
	}

	// Auto-done: when progress reaches 100%, mark the task as done.
	// When it drops below 100%, mark it as not done (only if the task was
	// previously auto-completed at 100%).
	doneChanged := false
	if newPercent >= 1.0 && !parent.Done {
		now := time.Now()
		_, err = s.ID(parent.ID).Cols("done", "done_at").Update(&Task{
			Done:   true,
			DoneAt: now,
		})
		if err != nil {
			return err
		}
		doneChanged = true
	} else if newPercent < 1.0 && parent.Done && parent.PercentDone >= 1.0 {
		// The parent was done at 100% — undo it since progress dropped.
		_, err = s.ID(parent.ID).Cols("done", "done_at").Update(&Task{
			Done:   false,
			DoneAt: time.Time{},
		})
		if err != nil {
			return err
		}
		doneChanged = true
	}

	if doneChanged {
		// When auto-done, cascade downward: mark any remaining children
		// as done so the task tree stays consistent. This covers the case
		// where weighted children sum to 100% but unweighted siblings are
		// still open.
		if newPercent >= 1.0 {
			if err := markChildrenDone(s, parentID, true, nil); err != nil {
				return err
			}
		}

		// Cascade upward so that grandparent tasks also recalculate their
		// progress. The recursion is bounded because each level only fires
		// when percent_done actually changed, and a task can only be
		// auto-done once (the !parent.Done guard above prevents re-entry).
		if err := recalculateRelatedTasksPercentDone(s, parentID); err != nil {
			return err
		}
	}

	return nil
}

// markChildrenDone recursively marks all children (subtasks and related tasks)
// of the given task as done (or not done). It traverses the entire hierarchy
// downward via progress-contributing relations. A visited set prevents cycles.
//
// When done is true each child gets Done=true, DoneAt=now, PercentDone=1.0.
// When done is false each child gets Done=false, DoneAt=zero, PercentDone=0.
func markChildrenDone(s *xorm.Session, taskID int64, done bool, visited map[int64]bool) error {
	if visited == nil {
		visited = make(map[int64]bool)
	}
	if visited[taskID] {
		return nil
	}
	visited[taskID] = true

	// Find outgoing progress-contributing relations (subtask, related).
	relations := []*TaskRelation{}
	err := s.Where("task_id = ?", taskID).
		In("relation_kind", progressRelationKinds).
		Find(&relations)
	if err != nil {
		return err
	}

	childIDs := make([]int64, 0, len(relations))
	for _, r := range relations {
		if !visited[r.OtherTaskID] {
			childIDs = append(childIDs, r.OtherTaskID)
		}
	}

	if len(childIDs) == 0 {
		return nil
	}

	children := []*Task{}
	err = s.In("id", childIDs).Find(&children)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, child := range children {
		if child.Done != done {
			if done {
				_, err = s.ID(child.ID).Cols("done", "done_at", "percent_done").Update(&Task{
					Done:        true,
					DoneAt:      now,
					PercentDone: 1.0,
				})
			} else {
				_, err = s.ID(child.ID).Cols("done", "done_at", "percent_done").Update(&Task{
					Done:        false,
					DoneAt:      time.Time{},
					PercentDone: 0,
				})
			}
			if err != nil {
				return err
			}
		}

		// Always recurse — a child may already be done but its own children
		// might not be.
		if err := markChildrenDone(s, child.ID, done, visited); err != nil {
			return err
		}
	}

	return nil
}

// recalculateRelatedTasksPercentDone finds all tasks that reference the given
// task via a progress-contributing relation (subtask or related) and
// recomputes their percent_done. It is a no-op if the task has no such
// relations.
func recalculateRelatedTasksPercentDone(s *xorm.Session, taskID int64) error {
	// Find every task that has *this* task as "other" side with a
	// progress-contributing kind. That means the found task_id is a "parent"
	// whose percent_done depends on taskID.
	parents := []*TaskRelation{}
	err := s.Where("other_task_id = ?", taskID).
		In("relation_kind", progressRelationKinds).
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
