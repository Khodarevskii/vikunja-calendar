<template>
	<div class="task-decomposition">
		<p class="help-text">
			{{ $t('task.decompose.help') }}
		</p>
		<div class="subtask-list">
			<div
				v-for="(item, index) in subtasks"
				:key="item.existingTaskId || `new-${index}`"
				class="subtask-row"
				:class="{'is-done': item.done}"
			>
				<span class="row-number">{{ index + 1 }}.</span>
				<Icon
					v-if="item.done"
					icon="check"
					class="done-icon"
				/>
				<input
					:ref="el => setInputRef(index, el)"
					v-model="item.title"
					type="text"
					class="input subtask-title"
					:class="{'is-strikethrough': item.done}"
					:placeholder="$t('task.decompose.titlePlaceholder')"
					:disabled="isCreating"
					@keydown.enter.prevent="addRow(index)"
				>
				<div class="assignee-wrapper">
					<Multiselect
						v-model="item.assignees"
						:placeholder="$t('task.decompose.assigneePlaceholder')"
						:loading="userSearchLoading"
						:search-results="foundUsers"
						label="name"
						:multiple="true"
						:disabled="isCreating"
						@search="findUser"
					>
						<template #tag="{item: assignee}">
							<span class="assignee-chip">
								<User
									:avatar-size="24"
									:show-username="false"
									:user="(assignee as IUser)"
								/>
								<BaseButton
									v-if="!isCreating"
									class="remove-assignee"
									@click="() => removeAssigneeFromRow(item, assignee as IUser)"
								>
									<Icon icon="times" />
								</BaseButton>
							</span>
						</template>
						<template #searchResult="{option: user}">
							<User
								:avatar-size="24"
								:show-username="true"
								:user="(user as IUser)"
							/>
						</template>
					</Multiselect>
				</div>
				<div class="weight-wrapper">
					<input
						v-model.number="item.weight"
						type="number"
						class="input subtask-weight"
						min="0"
						max="100"
						:disabled="isCreating"
						@keydown.enter.prevent="addRow(index)"
					>
					<span class="weight-sign">%</span>
				</div>
				<BaseButton
					v-if="subtasks.length > 1"
					class="remove-row"
					:disabled="isCreating"
					@click="removeRow(index)"
				>
					<Icon icon="times" />
				</BaseButton>
			</div>
		</div>
		<div class="list-controls">
			<BaseButton
				class="add-row-button"
				:disabled="isCreating || totalWeight >= 100"
				@click="addRow(subtasks.length - 1)"
			>
				<Icon icon="plus" />
				{{ $t('task.decompose.addRow') }}
			</BaseButton>
			<span
				class="weight-total"
				:class="{
					'is-valid': totalWeight === 100,
					'is-warning': totalWeight > 0 && totalWeight < 100,
					'is-error': totalWeight > 100,
				}"
			>
				{{ $t('task.decompose.totalWeight') }}: {{ totalWeight }}%
				<span
					v-if="totalWeight > 100"
					class="weight-error-text"
				>
					({{ $t('task.decompose.weightExceeded') }})
				</span>
			</span>
		</div>
		<div class="actions">
			<XButton
				:loading="isCreating"
				:disabled="!canCreate"
				icon="project-diagram"
				@click="createSubtasks"
			>
				{{ $t('task.decompose.create') }}
			</XButton>

			<span
				v-if="errorMessage"
				class="error-notice has-text-danger"
			>
				{{ errorMessage }}
			</span>
		</div>
	</div>
</template>

<script setup lang="ts">
import {ref, reactive, computed, shallowReactive, watch, nextTick, type ComponentPublicInstance} from 'vue'
import {useI18n} from 'vue-i18n'

import TaskService from '@/services/task'
import TaskModel from '@/models/task'
import TaskRelationService from '@/services/taskRelation'
import TaskRelationModel from '@/models/taskRelation'
import ProjectUserService from '@/services/projectUsers'
import {RELATION_KIND} from '@/types/IRelationKind'

import type {ITask} from '@/modelTypes/ITask'
import type {IUser} from '@/modelTypes/IUser'

import BaseButton from '@/components/base/BaseButton.vue'
import Multiselect from '@/components/input/Multiselect.vue'
import User from '@/components/misc/User.vue'
import {getDisplayName} from '@/models/user'
import {success} from '@/message'
import {useTaskStore} from '@/stores/tasks'

interface SubtaskRow {
	title: string
	weight: number
	assignees: IUser[]
	existingTaskId?: number
	done?: boolean
}

const WEIGHT_REGEX = /<!-- decompose-weight:(\d+) -->/

const props = defineProps<{
	taskId: ITask['id'],
	projectId: ITask['projectId'],
	existingSubtasks?: ITask[],
	parentTask?: ITask,
}>()

const emit = defineEmits<{
	'created': [tasks: ITask[]],
	'relationRemoved': [taskId: number],
}>()

const {t} = useI18n({useScope: 'global'})
const taskStore = useTaskStore()

const subtasks = reactive<SubtaskRow[]>([
	{title: '', weight: 100, assignees: []},
])

// Fetch full task data for subtask rows missing assignees (fallback when API
// does not include assignees in the related-tasks response).
async function loadMissingAssignees() {
	const taskService = new TaskService()
	for (const row of subtasks) {
		if (row.existingTaskId && row.assignees.length === 0) {
			try {
				const full = await taskService.get({id: row.existingTaskId})
				if (full.assignees && full.assignees.length > 0) {
					row.assignees = full.assignees.map((u: IUser) => ({...u, name: getDisplayName(u)}))
				}
			} catch {
				// silently skip – assignees won't be pre-filled
			}
		}
	}
}

// Load existing subtasks into the form and sync when Related Tasks changes
watch(
	() => props.existingSubtasks,
	(existing) => {
		const existingIds = new Set((existing || []).map(s => s.id))

		// Remove rows whose existing task was deleted from Related Tasks
		for (let i = subtasks.length - 1; i >= 0; i--) {
			if (subtasks[i].existingTaskId && !existingIds.has(subtasks[i].existingTaskId!)) {
				subtasks.splice(i, 1)
			}
		}

		if (!existing || existing.length === 0) {
			// If all existing removed and no new rows left, ensure at least one empty row
			if (subtasks.length === 0) {
				subtasks.push({title: '', weight: 100, assignees: []})
			}
			return
		}

		// Add/update existing subtasks that aren't already in the list
		const currentExistingIds = new Set(
			subtasks.filter(s => s.existingTaskId).map(s => s.existingTaskId),
		)

		let needsAssigneeLoad = false

		for (const sub of existing) {
			if (currentExistingIds.has(sub.id)) {
				// Update done status and assignees for existing rows
				const row = subtasks.find(s => s.existingTaskId === sub.id)
				if (row) {
					row.done = sub.done
					// Sync assignees if the API now provides them but the row is empty
					if (row.assignees.length === 0 && sub.assignees && sub.assignees.length > 0) {
						row.assignees = sub.assignees.map((u: IUser) => ({...u, name: getDisplayName(u)}))
					}
				}
				continue
			}

			const weightMatch = WEIGHT_REGEX.exec(sub.description || '')
			const weight = weightMatch ? parseInt(weightMatch[1], 10) : 0
			const assignees: IUser[] = (sub.assignees || []).map((u: IUser) => ({...u, name: getDisplayName(u)}))

			if (assignees.length === 0) {
				needsAssigneeLoad = true
			}

			// Insert before the last empty row (if any)
			const insertAt = subtasks.length > 0 && !subtasks[subtasks.length - 1].existingTaskId && subtasks[subtasks.length - 1].title === ''
				? subtasks.length - 1
				: subtasks.length

			subtasks.splice(insertAt, 0, {
				title: sub.title,
				weight,
				assignees,
				existingTaskId: sub.id,
				done: sub.done,
			})
		}

		// Recalculate weight of empty new rows so total doesn't exceed 100
		const usedWeight = subtasks
			.filter(s => s.existingTaskId || s.title.trim() !== '')
			.reduce((sum, r) => sum + r.weight, 0)
		const remaining = Math.max(0, 100 - usedWeight)

		// Ensure there's at least one empty row for adding new subtasks
		const hasEmptyNew = subtasks.some(s => !s.existingTaskId && s.title === '')
		if (!hasEmptyNew) {
			subtasks.push({title: '', weight: remaining, assignees: []})
		} else {
			for (const s of subtasks) {
				if (!s.existingTaskId && s.title === '') {
					s.weight = remaining
				}
			}
		}

		// If any subtask rows have no assignees, fetch them individually
		if (needsAssigneeLoad) {
			loadMissingAssignees()
		}
	},
	{immediate: true},
)

const inputRefs = ref<Record<number, HTMLInputElement | null>>({})

function setInputRef(index: number, el: HTMLInputElement | Element | ComponentPublicInstance | null) {
	inputRefs.value[index] = el as HTMLInputElement | null
}

const isCreating = ref(false)
const createdCount = ref(0)
const errorMessage = ref('')

// User search
const projectUserService = shallowReactive(new ProjectUserService())
const userSearchLoading = computed(() => projectUserService.loading)
const foundUsers = ref<IUser[]>([])

async function findUser(query: string) {
	const response = await projectUserService.getAll({projectId: props.projectId}, {s: query}) as IUser[]
	foundUsers.value = response.map(u => {
		u.name = getDisplayName(u)
		return u
	})
}

const totalWeight = computed(() => {
	return subtasks.reduce((sum, item) => sum + (item.weight || 0), 0)
})

const canCreate = computed(() => {
	const hasNewSubtasks = subtasks.some(item => item.title.trim().length > 0 && !item.existingTaskId)
	const hasExistingSubtasks = subtasks.some(item => item.existingTaskId)
	return (hasNewSubtasks || hasExistingSubtasks) && totalWeight.value <= 100
})

function removeAssigneeFromRow(row: SubtaskRow, user: IUser) {
	row.assignees = row.assignees.filter(a => a.id !== user.id)
}

function addRow(afterIndex: number) {
	const currentTotal = subtasks.reduce((sum, item) => sum + (item.weight || 0), 0)
	const remaining = Math.max(0, 100 - currentTotal)
	subtasks.splice(afterIndex + 1, 0, {title: '', weight: remaining, assignees: []})
	nextTick(() => {
		inputRefs.value[afterIndex + 1]?.focus()
	})
}

async function removeRow(index: number) {
	if (subtasks.length <= 1) return

	const row = subtasks[index]

	// If this is an existing subtask, delete the relation
	if (row.existingTaskId) {
		const taskRelationService = new TaskRelationService()
		await taskRelationService.delete(new TaskRelationModel({
			taskId: props.taskId,
			otherTaskId: row.existingTaskId,
			relationKind: RELATION_KIND.SUBTASK,
		}))
		emit('relationRemoved', row.existingTaskId)
	}

	subtasks.splice(index, 1)
}

function focus() {
	nextTick(() => inputRefs.value[0]?.focus())
}

defineExpose({focus})

async function createSubtasks() {
	const newSubtasks = subtasks.filter(item => item.title.trim().length > 0 && !item.existingTaskId)
	const existingSubtasks = subtasks.filter(item => item.existingTaskId)

	if (newSubtasks.length === 0 && existingSubtasks.length === 0) {
		return
	}

	if (isCreating.value || totalWeight.value > 100) {
		return
	}

	isCreating.value = true
	errorMessage.value = ''
	createdCount.value = 0

	const taskService = new TaskService()
	const taskRelationService = new TaskRelationService()
	const createdTasks: ITask[] = []

	try {
		// Update existing subtasks (title, weight, assignee)
		for (const item of existingSubtasks) {
			const description = `<!-- decompose-weight:${item.weight} -->\n${t('task.decompose.weightLabel', {weight: item.weight})}`

			await taskService.update(new TaskModel({
				id: item.existingTaskId,
				title: item.title.trim(),
				description,
				projectId: props.projectId,
			}))

			// Handle assignee changes for existing subtasks
			const originalTask = props.existingSubtasks?.find(s => s.id === item.existingTaskId)
			const originalIds = new Set((originalTask?.assignees || []).map(u => u.id))
			const newIds = new Set(item.assignees.map(u => u.id))

			// Remove assignees that were removed from the form
			for (const user of (originalTask?.assignees || [])) {
				if (!newIds.has(user.id)) {
					await taskStore.removeAssignee({user, taskId: item.existingTaskId!})
				}
			}
			// Add assignees that were added in the form
			for (const user of item.assignees) {
				if (!originalIds.has(user.id)) {
					await taskStore.addAssignee({user, taskId: item.existingTaskId!})
				}
			}
		}

		// Create new subtasks – inherit dates from parent so they appear in the calendar
		const parentDates: Partial<ITask> = {}
		if (props.parentTask) {
			if (props.parentTask.dueDate) parentDates.dueDate = props.parentTask.dueDate
			if (props.parentTask.startDate) parentDates.startDate = props.parentTask.startDate
			if (props.parentTask.endDate) parentDates.endDate = props.parentTask.endDate
		}

		for (const item of newSubtasks) {
			const description = `<!-- decompose-weight:${item.weight} -->\n${t('task.decompose.weightLabel', {weight: item.weight})}`

			const newTask = await taskService.create(new TaskModel({
				title: item.title.trim(),
				description,
				projectId: props.projectId,
				...parentDates,
			}))

			await taskRelationService.create(new TaskRelationModel({
				taskId: props.taskId,
				otherTaskId: newTask.id,
				relationKind: RELATION_KIND.SUBTASK,
			}))

			// Assign users if selected
			for (const user of item.assignees) {
				await taskStore.addAssignee({user, taskId: newTask.id})
			}

			// Mark the row as existing so the watch won't duplicate it
			item.existingTaskId = newTask.id
			createdTasks.push(newTask)
		}

		createdCount.value = createdTasks.length
		emit('created', createdTasks)

		if (createdTasks.length > 0) {
			success({message: t('task.decompose.success', {count: createdTasks.length})})
		} else {
			success({message: t('task.decompose.updated')})
		}
	} catch (e) {
		errorMessage.value = t('task.decompose.error')
		throw e
	} finally {
		isCreating.value = false
	}
}
</script>

<style lang="scss" scoped>

:deep(.hint-text){
	display: none!important;
}
.task-decomposition {
	.help-text {
		color: var(--grey-500);
		font-size: 0.9rem;
		margin-block-end: 0.75rem;
	}

	.subtask-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.subtask-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.row-number {
		color: var(--grey-400);
		font-size: 0.85rem;
		min-inline-size: 1.5rem;
		text-align: end;
	}

	.subtask-title {
		flex: 1;
		min-inline-size: 0;

		&.is-strikethrough {
			text-decoration: line-through;
			color: var(--grey-400);
		}
	}

	.done-icon {
		color: var(--success);
		font-size: 0.8rem;
		flex-shrink: 0;
	}


	.assignee-wrapper {
		flex-shrink: 0;
		inline-size: 14rem;

		:deep(.multiselect) {
			.input-wrapper {
				min-block-size: auto;
				padding: 0.2rem 0.4rem;
			}

			.input {
				font-size: 0.85rem;
			}
		}
	}

	.assignee-chip {
		position: relative;
		display: inline-block;
		margin-inline-start: -0.25rem;

		&:first-child {
			margin-inline-start: 0;
		}

		:deep(.user img) {
			border: 2px solid var(--white);
			margin: 0;
		}

		:deep(.user .username) {
			display: none;
		}
	}

	.remove-assignee {
		position: absolute;
		inset-block-start: -2px;
		inset-inline-start: -2px;
		color: var(--danger);
		background: var(--white);
		display: block;
		border-radius: 100%;
		font-size: 0.6rem;
		inline-size: 14px;
		block-size: 14px;
		z-index: 100;
		line-height: 1;
		opacity: 0;
		transition: opacity $transition;
	}

	.assignee-chip:hover .remove-assignee {
		opacity: 1;
	}

	.weight-wrapper {
		display: flex;
		align-items: center;
		gap: 0.15rem;
		flex-shrink: 0;
	}

	.subtask-weight {
		inline-size: 4rem;
		text-align: center;
	}

	.weight-sign {
		color: var(--grey-500);
		font-size: 0.9rem;
	}

	.input {
		padding: 0.4rem 0.5rem;
		border: 1px solid var(--border);
		border-radius: $radius;
		background: var(--scheme-main);
		color: var(--text);
		font-size: 0.9rem;
		transition: border-color $transition;

		&:focus {
			border-color: var(--primary);
			outline: none;
		}

		&::placeholder {
			color: var(--grey-400);
			font-style: italic;
		}
	}

	.remove-row {
		color: var(--danger);
		padding: 0.25rem;
		flex-shrink: 0;
		line-height: 1;
	}

	.list-controls {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-block-start: 0.5rem;
	}

	.add-row-button {
		color: var(--primary);
		font-size: 0.85rem;
		display: flex;
		align-items: center;
		gap: 0.25rem;

		&[disabled] {
			opacity: 0.5;
			cursor: not-allowed;
		}
	}

	.weight-total {
		font-size: 0.85rem;
		font-weight: 600;

		&.is-valid {
			color: var(--success);
		}

		&.is-warning {
			color: var(--warning);
		}

		&.is-error {
			color: var(--danger);
		}
	}

	.weight-error-text {
		font-weight: 400;
	}

	.actions {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin-block-start: 0.75rem;
	}

	.created-notice,
	.error-notice {
		font-size: 0.85rem;
	}
}
</style>
