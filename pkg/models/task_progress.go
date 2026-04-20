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
	"regexp"
	"time"

	"xorm.io/xorm"
)

// descriptionTaskItemRE finds every TipTap task-list checkbox rendered into
// the task description. Each <li data-type="taskItem"> node carries a
// data-checked="true|false" attribute — that is the only place in the editor's
// HTML output where this attribute appears, so a simple regex is enough to
// count the items and their state without pulling in a full HTML parser.
var descriptionTaskItemRE = regexp.MustCompile(`data-checked="(true|false)"`)

// extractDescriptionChecklistItems scans the description HTML for TipTap task
// list checkboxes and returns one unweighted progress item per checkbox. The
// zero Weight lets these items share the "auto" remainder of 100% equally
// with unweighted subtasks and structured checklist items.
func extractDescriptionChecklistItems(description string) []progressItem {
	if description == "" {
		return nil
	}
	matches := descriptionTaskItemRE.FindAllStringSubmatch(description, -1)
	if len(matches) == 0 {
		return nil
	}
	items := make([]progressItem, 0, len(matches))
	for _, m := range matches {
		items = append(items, progressItem{Done: m[1] == "true"})
	}
	return items
}

// syncTaskBucketForDoneChange moves the task into the done bucket (or back to
// the default bucket) on all manual-kanban views of its project. Called from
// the progress cascade because raw Update() calls on the done column bypass
// updateSingleTask's bucket handling.
func syncTaskBucketForDoneChange(s *xorm.Session, taskID int64, done bool) error {
	task, err := GetTaskByIDSimple(s, taskID)
	if err != nil {
		return err
	}

	views := []*ProjectView{}
	err = s.
		Where("project_id = ? AND view_kind = ? AND bucket_configuration_mode = ?",
			task.ProjectID, ProjectViewKindKanban, BucketConfigurationModeManual).
		Find(&views)
	if err != nil {
		return err
	}

	for _, view := range views {
		current := &TaskBucket{}
		_, err := s.Where("task_id = ? AND project_view_id = ?", taskID, view.ID).Get(current)
		if err != nil {
			return err
		}

		var targetBucketID int64
		if done {
			if view.DoneBucketID == 0 {
				continue
			}
			targetBucketID = view.DoneBucketID
		} else {
			if current.BucketID != view.DoneBucketID {
				continue
			}
			targetBucketID, err = getDefaultBucketID(s, view)
			if err != nil {
				return err
			}
		}

		if current.BucketID == targetBucketID {
			continue
		}

		_, err = s.Where("task_id = ? AND project_view_id = ?", taskID, view.ID).
			Cols("bucket_id").
			Update(&TaskBucket{BucketID: targetBucketID})
		if err != nil {
			return err
		}
	}
	return nil
}

// progressRelationKinds lists the relation kinds that contribute to a task's
// percent_done when they appear as outgoing relations (task_id → other_task_id).
// Only real subtasks count — other relation kinds (including "related") are
// not cascaded either for progress or for auto-done status.
// Stored as []interface{} so we can pass it directly to xorm's In() method.
var progressRelationKinds = []interface{}{
	RelationKindSubtask,
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
	descriptionItems := extractDescriptionChecklistItems(parent.Description)

	// Nothing to compute from.
	if len(relatedTasks) == 0 && len(checklistItems) == 0 && len(descriptionItems) == 0 {
		return nil
	}

	items := make([]progressItem, 0, len(relatedTasks)+len(checklistItems)+len(descriptionItems))
	for _, rt := range relatedTasks {
		items = append(items, progressItem{Weight: rt.SubtaskWeight, Done: rt.Done})
	}
	for _, ci := range checklistItems {
		items = append(items, progressItem{Weight: ci.Weight, Done: ci.Done})
	}
	items = append(items, descriptionItems...)

	newPercent := calculateWeightedProgress(items)

	percentChanged := math.Abs(newPercent-parent.PercentDone) >= 0.001

	if percentChanged {
		_, err = s.ID(parent.ID).Cols("percent_done").Update(&Task{PercentDone: newPercent})
		if err != nil {
			return err
		}
	}

	// Auto-done: when progress reaches 100%, mark the task as done.
	// When it drops below 100%, mark it as not done (only if the task was
	// previously auto-completed at 100%).
	//
	// This check runs even when percent_done did not change so that a stale
	// done/percent pair gets reconciled (e.g. percent already at 1.0 but done
	// still false).
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
		if err := syncTaskBucketForDoneChange(s, parent.ID, true); err != nil {
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
		if err := syncTaskBucketForDoneChange(s, parent.ID, false); err != nil {
			return err
		}
		doneChanged = true
	}

	// When auto-done, cascade downward so weighted children that summed to
	// 100% don't leave unweighted siblings open.
	if doneChanged && newPercent >= 1.0 {
		if err := markChildrenDone(s, parentID, true, nil); err != nil {
			return err
		}
	}

	// Cascade upward on ANY change (percent or done) so grandparents
	// recompute their progress even when the intermediate parent did not
	// cross the 100% threshold. The recursion is bounded because each
	// ancestor only propagates further when its own state actually changed.
	if percentChanged || doneChanged {
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
			if err := syncTaskBucketForDoneChange(s, child.ID, done); err != nil {
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
