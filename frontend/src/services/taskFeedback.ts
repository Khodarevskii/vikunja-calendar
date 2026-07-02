import AbstractService from './abstractService'
import type {ITask} from '@/modelTypes/ITask'
import type {IUser} from '@/modelTypes/IUser'

interface IFeedbackReviewer {
	id?: number
	taskId: ITask['id']
	userId: IUser['id']
}

/**
 * Reviewers CRUD: add/remove a user to the task's feedback reviewer list.
 * The first add flips the task into feedback mode and pins the caller as
 * the feedback manager.
 */
export class FeedbackReviewerService extends AbstractService<IFeedbackReviewer> {
	constructor() {
		super({
			create: '/tasks/{taskId}/feedback/reviewers',
			delete: '/tasks/{taskId}/feedback/reviewers/{userId}',
		})
	}

	modelFactory(data): IFeedbackReviewer {
		return data as IFeedbackReviewer
	}
}

interface IFeedbackSubmission {
	taskId: ITask['id']
	text: string
	attachmentIds: number[]
}

/**
 * Submits a feedback message + optional attachment ids to the task's
 * feedback manager. Does not close the task.
 */
export class FeedbackSubmissionService extends AbstractService<IFeedbackSubmission> {
	constructor() {
		super({
			create: '/tasks/{taskId}/feedback/submit',
		})
	}

	modelFactory(data): IFeedbackSubmission {
		return data as IFeedbackSubmission
	}
}
