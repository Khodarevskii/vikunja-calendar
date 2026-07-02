<template>
	<div class="task-feedback">
		<h3>
			<span class="icon is-grey">
				<Icon icon="comments" />
			</span>
			{{ $t('task.feedback.heading') }}
		</h3>

		<p
			v-if="!task.feedbackRequested"
			class="feedback-help"
		>
			{{ $t('task.feedback.description') }}
		</p>
		<p
			v-else
			class="feedback-help"
		>
			<span>{{ $t('task.feedback.managerLabel') }}: </span>
			<strong>{{ managerName }}</strong>
		</p>

		<div
			v-if="task.feedbackRequested && (task.feedbackReviewers || []).length > 0"
			class="reviewer-chips"
		>
			<span
				v-for="reviewer in task.feedbackReviewers"
				:key="reviewer.id"
				class="reviewer-chip"
			>
				<User
					:avatar-size="24"
					:show-username="true"
					:user="reviewer"
				/>
				<BaseButton
					v-if="canManage"
					class="chip-remove"
					@click="removeReviewer(reviewer)"
				>
					<Icon icon="times" />
				</BaseButton>
			</span>
		</div>

		<div
			v-if="canEdit"
			class="reviewer-search"
		>
			<Multiselect
				:model-value="selectedForAdd"
				:placeholder="$t('task.feedback.addReviewer')"
				:loading="userSearchLoading"
				:search-results="foundUsers"
				label="name"
				:multiple="false"
				:autocomplete-enabled="false"
				@search="findUser"
				@select="addReviewer"
			>
				<template #searchResult="{option: user}">
					<div class="reviewer-search-row">
						<User
							:avatar-size="24"
							:show-username="true"
							:user="(user as IUser)"
						/>
						<BaseButton
							class="chip-add"
							:disabled="isAlreadyReviewer(user as IUser)"
							@click.stop.prevent="addReviewer(user as IUser)"
						>
							<Icon icon="plus" />
						</BaseButton>
					</div>
				</template>
			</Multiselect>
		</div>

		<div
			v-if="task.feedbackRequested && canManage"
			class="actions"
		>
			<XButton
				variant="tertiary"
				@click="disableFeedback"
			>
				{{ $t('task.feedback.disable') }}
			</XButton>
		</div>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, shallowReactive} from 'vue'
import {useI18n} from 'vue-i18n'

import Multiselect from '@/components/input/Multiselect.vue'
import User from '@/components/misc/User.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import ProjectUserService from '@/services/projectUsers'
import {FeedbackReviewerService} from '@/services/taskFeedback'
import {getDisplayName} from '@/models/user'
import {useAuthStore} from '@/stores/auth'
import type {ITask} from '@/modelTypes/ITask'
import type {IUser} from '@/modelTypes/IUser'

const props = defineProps<{
	task: ITask,
	projectId: number,
}>()

const emit = defineEmits<{
	'changed': [],
}>()

const {t} = useI18n({useScope: 'global'})
const authStore = useAuthStore()

const projectUserService = shallowReactive(new ProjectUserService())
const userSearchLoading = computed(() => projectUserService.loading)
const foundUsers = ref<IUser[]>([])
const selectedForAdd = ref<IUser | null>(null)

const currentUserId = computed(() => authStore.info?.id || 0)
const managerName = computed(() => {
	const m = (props.task.feedbackReviewers || []).find(() => false)
	// The manager may not be in the reviewer list; find them separately from
	// the assignees or fall back to the manager id.
	const inAssignees = (props.task.assignees || []).find(a => a.id === props.task.feedbackManagerId)
	return inAssignees ? getDisplayName(inAssignees) : String(props.task.feedbackManagerId)
})

const isManager = computed(() => currentUserId.value === props.task.feedbackManagerId)
const canManage = computed(() => props.task.feedbackRequested && isManager.value)
// Anyone with write access can enable feedback (become manager) or, once
// enabled, only the manager keeps editing rights.
const canEdit = computed(() => !props.task.feedbackRequested || canManage.value)

function isAlreadyReviewer(u: IUser): boolean {
	return (props.task.feedbackReviewers || []).some(r => r.id === u.id)
}

async function findUser(query: string) {
	const response = await projectUserService.getAll({projectId: props.projectId}, {s: query}) as IUser[]
	foundUsers.value = response.map(u => {
		u.name = getDisplayName(u)
		return u
	})
}

async function addReviewer(user: IUser) {
	if (!user || isAlreadyReviewer(user)) {
		selectedForAdd.value = null
		return
	}
	const svc = new FeedbackReviewerService()
	await svc.create({taskId: props.task.id, userId: user.id})
	selectedForAdd.value = null
	emit('changed')
}

async function removeReviewer(user: IUser) {
	const svc = new FeedbackReviewerService()
	await svc.delete({taskId: props.task.id, userId: user.id})
	emit('changed')
}

async function disableFeedback() {
	// Removing every reviewer implicitly disables feedback mode server-side
	// via the same DELETE endpoint.
	const svc = new FeedbackReviewerService()
	for (const r of props.task.feedbackReviewers || []) {
		await svc.delete({taskId: props.task.id, userId: r.id})
	}
	emit('changed')
}

// Keep the `t` reference alive for future i18n usage without triggering
// eslint no-unused warnings when this component is trimmed further.
void t
</script>

<style lang="scss" scoped>
.task-feedback {
	.feedback-help {
		color: var(--grey-500);
		font-size: 0.9rem;
		margin-block-end: 0.75rem;
	}

	.reviewer-chips {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		margin-block-end: 0.75rem;
	}

	.reviewer-chip {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		padding: 0.25rem 0.5rem;
		border: 1px solid var(--border);
		border-radius: 999px;
		background: var(--scheme-main-bis);
		color: var(--text);
	}

	.chip-remove {
		color: var(--danger);
		padding: 0.15rem;
		line-height: 1;
	}

	.reviewer-search {
		margin-block-end: 0.75rem;
	}

	.reviewer-search-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
	}

	.chip-add {
		color: var(--primary);
		padding: 0.15rem 0.35rem;
		line-height: 1;

		&:disabled,
		&[disabled] {
			color: var(--grey-400);
			cursor: not-allowed;
		}
	}

	.actions {
		display: flex;
		justify-content: flex-end;
	}
}
</style>
