<template>
	<Multiselect
		v-model="assignees"
		class="edit-assignees"
		:class="{'has-assignees': assignees.length > 0}"
		:loading="projectUserService.loading"
		:placeholder="$t('task.assignee.placeholder')"
		:multiple="true"
		:search-results="foundUsers"
		label="name"
		:select-placeholder="$t('task.assignee.selectPlaceholder')"
		:autocomplete-enabled="false"
		@search="findUser"
		@select="addAssignee"
	>
		<template #items="{items}">
			<AssigneeList
				:assignees="items"
				:disabled="disabled"
				can-remove
				@remove="removeAssignee"
			/>
		</template>
		<template #searchResult="{option: user}">
			<div class="assignee-search-row">
				<User
					:avatar-size="24"
					:show-username="true"
					:user="user"
				/>
				<BaseButton
					v-tooltip="isAlreadyReviewer(user as IUser)
						? $t('task.feedback.alreadyReviewer')
						: $t('task.feedback.addAsReviewer')"
					class="add-reviewer"
					:class="{'is-disabled': isAlreadyReviewer(user as IUser)}"
					:disabled="isAlreadyReviewer(user as IUser)"
					@click.stop.prevent="addAsReviewer(user as IUser)"
				>
					<span class="reviewer-label-full">{{ $t('task.feedback.action') }}</span>
					<span class="reviewer-label-short">{{ $t('task.feedback.actionShort') }}</span>
				</BaseButton>
			</div>
		</template>
	</Multiselect>
</template>

<script setup lang="ts">
import {ref, shallowReactive, watch, nextTick} from 'vue'
import {useI18n} from 'vue-i18n'

import User from '@/components/misc/User.vue'
import Multiselect from '@/components/input/Multiselect.vue'
import BaseButton from '@/components/base/BaseButton.vue'

import {includesById} from '@/helpers/utils'
import ProjectUserService from '@/services/projectUsers'
import {FeedbackReviewerService} from '@/services/taskFeedback'
import {success, error as msgError} from '@/message'
import {useTaskStore} from '@/stores/tasks'

import type {IUser} from '@/modelTypes/IUser'
import {getDisplayName} from '@/models/user'
import AssigneeList from '@/components/tasks/partials/AssigneeList.vue'

const props = withDefaults(defineProps<{
	modelValue: IUser[] | undefined,
	taskId: number,
	projectId: number,
	disabled?: boolean,
	/** Ids of users already on the feedback reviewer list — greys the envelope button. */
	feedbackReviewerIds?: number[],
}>(), {
	disabled: false,
	feedbackReviewerIds: () => [],
})

const emit = defineEmits<{
	'update:modelValue': [value: IUser[] | undefined],
	'feedbackReviewerAdded': [user: IUser],
}>()

const taskStore = useTaskStore()
const {t} = useI18n({useScope: 'global'})

const projectUserService = shallowReactive(new ProjectUserService())
const foundUsers = ref<IUser[]>([])
const assignees = ref<IUser[]>([])
let isAdding = false

watch(
	() => props.modelValue,
	(value) => {
		assignees.value = value
	},
	{
		immediate: true,
		deep: true,
	},
)

async function addAssignee(user: IUser) {
	if (isAdding) {
		return
	}

	try {
		nextTick(() => isAdding = true)

		await taskStore.addAssignee({user: user, taskId: props.taskId})
		emit('update:modelValue', assignees.value)
		success({message: t('task.assignee.assignSuccess')})
	} finally {
		nextTick(() => isAdding = false)
	}
}

async function removeAssignee(user: IUser) {
	await taskStore.removeAssignee({user: user, taskId: props.taskId})

	// Remove the assignee from the project
	for (const a in assignees.value) {
		if (assignees.value[a].id === user.id) {
			assignees.value.splice(a, 1)
		}
	}
	success({message: t('task.assignee.unassignSuccess')})
}

function isAlreadyReviewer(user: IUser): boolean {
	return (props.feedbackReviewerIds || []).includes(user.id)
}

async function addAsReviewer(user: IUser) {
	if (isAlreadyReviewer(user)) return
	try {
		const svc = new FeedbackReviewerService()
		await svc.create({taskId: props.taskId, userId: user.id})
		success({message: t('task.feedback.sentToReviewer', {name: getDisplayName(user)})})
		emit('feedbackReviewerAdded', user)
	} catch (e) {
		msgError(e)
	}
}

async function findUser(query: string) {
	const response = await projectUserService.getAll({projectId: props.projectId}, {s: query}) as IUser[]

	// Filter the results to not include users who are already assigned
	foundUsers.value = response
		.filter(({id}) => !includesById(assignees.value, id))
		.map(u => {
			// Users may not have a display name set, so we fall back on the username in that case
			u.name = getDisplayName(u)
			return u
		})
}
</script>

<style lang="scss">
.edit-assignees.has-assignees.multiselect .input {
	padding-inline-start: 0;
}

.assignee-search-row {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 0.5rem;
	inline-size: 100%;
}

.add-reviewer {
	display: inline-flex;
	align-items: center;
	color: var(--primary);
	font-size: 0.85rem;
	font-weight: 600;
	padding: 0.2rem 0.55rem;
	border: 1px solid var(--primary);
	border-radius: 999px;
	line-height: 1;
	flex-shrink: 0;
	max-inline-size: 100%;

	.reviewer-label-full {
		display: inline;
	}
	.reviewer-label-short {
		display: none;
	}

	// Collapse to "ОС" if the surrounding cell is narrow.
	@container assignee-search-row (max-width: 220px) {
		.reviewer-label-full { display: none; }
		.reviewer-label-short { display: inline; }
	}

	// Fallback for browsers without container queries — same trigger via
	// screen width once the sidebar becomes cramped.
	@media screen and (max-width: 640px) {
		.reviewer-label-full { display: none; }
		.reviewer-label-short { display: inline; }
	}

	&.is-disabled,
	&:disabled {
		color: var(--grey-400);
		border-color: var(--grey-400);
		cursor: not-allowed;
	}
}

.assignee-search-row {
	container-type: inline-size;
	container-name: assignee-search-row;
}
</style>
