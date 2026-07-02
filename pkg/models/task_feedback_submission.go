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
	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/notifications"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// TaskFeedbackSubmission is the request body used by reviewers to send
// feedback on a task in feedback mode. The task itself is not modified —
// closing it is still the manager's responsibility. The submission is
// delivered to the manager via email.
type TaskFeedbackSubmission struct {
	TaskID        int64   `xorm:"-" json:"-" param:"projecttask"`
	Text          string  `xorm:"-" json:"text"`
	AttachmentIDs []int64 `xorm:"-" json:"attachment_ids"`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName is unused but required by the CRUD contract.
func (*TaskFeedbackSubmission) TableName() string { return "tasks" }

// CanCreate: any reviewer of a task in feedback mode may submit.
func (f *TaskFeedbackSubmission) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	task, err := GetTaskByIDSimple(s, f.TaskID)
	if err != nil {
		return false, err
	}
	if !task.FeedbackRequested {
		return false, nil
	}
	return isFeedbackReviewer(s, f.TaskID, a.GetID())
}

// Create sends an email to the manager with the feedback body and links to
// any referenced attachments. It does not close the task or persist the
// submission — this is a fire-and-forget delivery.
// @Summary Submit feedback for a task
// @Description Sends the feedback body to the task's feedback manager via email. Referenced attachments must already exist on the task; their URLs are included in the notification.
// @tags task-feedback
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param taskID path int true "Task ID"
// @Param submission body models.TaskFeedbackSubmission true "The feedback submission"
// @Success 200 {object} models.Message "Feedback was sent."
// @Failure 403 {object} web.HTTPError "Not a reviewer or feedback mode is off."
// @Failure 500 {object} models.Message "Internal error"
// @Router /tasks/{taskID}/feedback/submit [post]
func (f *TaskFeedbackSubmission) Create(s *xorm.Session, a web.Auth) error {
	task, err := GetTaskByIDSimple(s, f.TaskID)
	if err != nil {
		return err
	}
	manager, err := user.GetUserByID(s, task.FeedbackManagerID)
	if err != nil {
		return err
	}
	submitter, err := user.GetUserByID(s, a.GetID())
	if err != nil {
		return err
	}

	attachmentURL := ""
	if len(f.AttachmentIDs) > 0 {
		attachmentURL = config.ServicePublicURL.GetString() + "tasks/" + itoa(f.TaskID)
	}

	return notifications.Notify(manager, &TaskFeedbackSubmittedNotification{
		Submitter:     submitter,
		Task:          &task,
		Manager:       manager,
		FeedbackText:  f.Text,
		AttachmentURL: attachmentURL,
	})
}

// itoa is a tiny helper for int64→string in the message body.
func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
