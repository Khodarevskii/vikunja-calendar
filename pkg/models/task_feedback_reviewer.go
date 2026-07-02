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
	"time"

	"code.vikunja.io/api/pkg/notifications"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// TaskFeedbackReviewer marks a user from whom feedback is expected on a
// given task while the task is in feedback mode.
type TaskFeedbackReviewer struct {
	ID      int64     `xorm:"bigint autoincr not null unique pk" json:"id"`
	TaskID  int64     `xorm:"bigint INDEX not null" json:"-" param:"projecttask"`
	UserID  int64     `xorm:"bigint INDEX not null" json:"user_id" param:"user"`
	Created time.Time `xorm:"created not null" json:"created"`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName gives the table a proper name.
func (*TaskFeedbackReviewer) TableName() string {
	return "task_feedback_reviewers"
}

// taskFeedbackReviewerWithUser is a helper for loading reviewer users in one query.
type taskFeedbackReviewerWithUser struct {
	TaskID    int64
	user.User `xorm:"extends"`
}

// getRawTaskFeedbackReviewersForTasks returns rows joined with the user table.
func getRawTaskFeedbackReviewersForTasks(s *xorm.Session, taskIDs []int64) (rows []*taskFeedbackReviewerWithUser, err error) {
	rows = []*taskFeedbackReviewerWithUser{}
	if len(taskIDs) == 0 {
		return
	}
	err = s.Table("task_feedback_reviewers").
		Select("task_feedback_reviewers.task_id, users.*").
		In("task_id", taskIDs).
		Join("INNER", "users", "task_feedback_reviewers.user_id = users.id").
		Find(&rows)
	return
}

// isFeedbackReviewer returns true if the given user is on the reviewer list.
func isFeedbackReviewer(s *xorm.Session, taskID, userID int64) (bool, error) {
	return s.Where("task_id = ? AND user_id = ?", taskID, userID).Exist(&TaskFeedbackReviewer{})
}

// addFeedbackReviewersToTasks populates the FeedbackReviewers slice on each
// task in taskMap. Called from the shared task-loading paths so the field
// is present on any task returned by the API.
func addFeedbackReviewersToTasks(s *xorm.Session, taskIDs []int64, taskMap map[int64]*Task) error {
	rows, err := getRawTaskFeedbackReviewersForTasks(s, taskIDs)
	if err != nil {
		return err
	}
	for i, r := range rows {
		if r == nil {
			continue
		}
		r.Email = "" // Obfuscate the email
		taskMap[r.TaskID].FeedbackReviewers = append(taskMap[r.TaskID].FeedbackReviewers, &rows[i].User)
	}
	return nil
}

// CanCreate: adding a reviewer activates feedback mode when it is off; only
// the current feedback manager can add reviewers once it is on. In both cases
// the acting user must have write access to the project.
func (r *TaskFeedbackReviewer) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	project, err := GetProjectSimpleByTaskID(s, r.TaskID)
	if err != nil {
		return false, err
	}
	canWrite, err := project.CanUpdate(s, a)
	if err != nil || !canWrite {
		return false, err
	}
	task, err := GetTaskByIDSimple(s, r.TaskID)
	if err != nil {
		return false, err
	}
	if !task.FeedbackRequested {
		return true, nil
	}
	return a.GetID() == task.FeedbackManagerID, nil
}

// CanDelete: only the current feedback manager can remove a reviewer.
func (r *TaskFeedbackReviewer) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	task, err := GetTaskByIDSimple(s, r.TaskID)
	if err != nil {
		return false, err
	}
	if !task.FeedbackRequested {
		return false, nil
	}
	return a.GetID() == task.FeedbackManagerID, nil
}

// CanRead: anyone with read access on the project can see the reviewer list.
func (r *TaskFeedbackReviewer) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	project, err := GetProjectSimpleByTaskID(s, r.TaskID)
	if err != nil {
		return false, 0, err
	}
	return project.CanRead(s, a)
}

// Create adds a reviewer to the task's feedback list. If feedback mode is
// currently off, activate it and set the acting user as the manager.
// @Summary Add a feedback reviewer to a task
// @Description Adds a user from whom feedback is expected. Activates feedback mode on the task if it is off and sets the acting user as the feedback manager.
// @tags task-feedback
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param taskID path int true "Task ID"
// @Param reviewer body models.TaskFeedbackReviewer true "The reviewer object"
// @Success 201 {object} models.TaskFeedbackReviewer "The created reviewer object."
// @Failure 400 {object} web.HTTPError "Invalid reviewer object provided."
// @Failure 500 {object} models.Message "Internal error"
// @Router /tasks/{taskID}/feedback/reviewers [put]
func (r *TaskFeedbackReviewer) Create(s *xorm.Session, a web.Auth) error {
	task, err := GetTaskByIDSimple(s, r.TaskID)
	if err != nil {
		return err
	}

	// Verify the target user has access to the project.
	project, err := GetProjectSimpleByTaskID(s, r.TaskID)
	if err != nil {
		return err
	}
	target, err := user.GetUserByID(s, r.UserID)
	if err != nil {
		return err
	}
	canRead, _, err := project.CanRead(s, target)
	if err != nil {
		return err
	}
	if !canRead {
		return ErrUserDoesNotHaveAccessToProject{ProjectID: project.ID, UserID: r.UserID}
	}

	// Reject duplicates rather than double-inserting.
	exists, err := s.Where("task_id = ? AND user_id = ?", r.TaskID, r.UserID).Exist(&TaskFeedbackReviewer{})
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// Activate feedback mode on the task if this is the first reviewer.
	if !task.FeedbackRequested {
		task.FeedbackRequested = true
		task.FeedbackManagerID = a.GetID()
		_, err = s.ID(task.ID).Cols("feedback_requested", "feedback_manager_id").Update(&task)
		if err != nil {
			return err
		}
	}

	if _, err = s.Insert(&TaskFeedbackReviewer{TaskID: r.TaskID, UserID: r.UserID}); err != nil {
		return err
	}

	// Reload the task with the manager attached so the notification can name them.
	fullTask, err := GetTaskByIDSimple(s, r.TaskID)
	if err != nil {
		return err
	}
	manager, err := user.GetUserByID(s, fullTask.FeedbackManagerID)
	if err == nil {
		_ = notifications.Notify(target, &TaskFeedbackRequestedNotification{
			Manager: manager,
			Task:    &fullTask,
			Target:  target,
		})
	}

	return nil
}

// Delete removes a reviewer from a task's feedback list.
// @Summary Remove a feedback reviewer from a task
// @Description Removes a user from the task's feedback reviewers.
// @tags task-feedback
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param taskID path int true "Task ID"
// @Param userID path int true "Reviewer user ID"
// @Success 200 {object} models.Message "The reviewer was successfully deleted."
// @Failure 500 {object} models.Message "Internal error"
// @Router /tasks/{taskID}/feedback/reviewers/{userID} [delete]
func (r *TaskFeedbackReviewer) Delete(s *xorm.Session, _ web.Auth) error {
	_, err := s.Where("task_id = ? AND user_id = ?", r.TaskID, r.UserID).Delete(&TaskFeedbackReviewer{})
	return err
}
