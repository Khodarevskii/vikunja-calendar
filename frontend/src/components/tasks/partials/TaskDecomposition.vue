<template>
	<div class="task-decomposition">
		<p class="help-text">
			{{ $t('task.decompose.help') }}
		</p>
		<div class="subtask-list">
			<div
				v-for="(row, index) in rows"
				:key="row.key"
				class="subtask-row"
				:class="{'is-done': row.done}"
			>
				<span class="row-number">{{ index + 1 }}.</span>
				<RouterLink
					v-if="row.taskId"
					v-tooltip="row.title"
					class="subtask-title subtask-link"
					:class="{'is-strikethrough': row.done}"
					:to="{name: 'task.detail', params: {id: row.taskId}, state: {backdropView: $route.fullPath}}"
				>
					{{ row.title }}
				</RouterLink>
				<input
					v-else
					:ref="el => setInputRef(index, el)"
					v-model="row.title"
					type="text"
					class="input subtask-title"
					:placeholder="$t('task.decompose.titlePlaceholder')"
					:disabled="isBusy"
					@keydown.enter.prevent="addRow(index)"
				>
				<div class="assignee-wrapper">
					<Multiselect
						:model-value="row.assignees"
						:placeholder="$t('task.decompose.assigneePlaceholder')"
						:loading="userSearchLoading"
						:search-results="foundUsers"
						label="name"
						:multiple="true"
						:disabled="isBusy"
						@search="findUser"
						@select="(user: IUser) => onAssigneeAdded(row, user)"
						@remove="(user: IUser) => onAssigneeRemoved(row, user)"
					>
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
						v-model.number="row.weight"
						type="number"
						class="input subtask-weight"
						min="0"
						max="100"
						step="10"
						:disabled="isBusy"
						@change="onWeightChange(row)"
						@keydown.enter.prevent="addRow(index)"
					>
					<span class="weight-sign">%</span>
				</div>
				<BaseButton
					v-if="rows.length > 1 || row.taskId"
					class="remove-row"
					:disabled="isBusy"
					@click="removeRow(index)"
				>
					<Icon icon="times" />
				</BaseButton>
			</div>
		</div>
		<div class="list-controls">
			<BaseButton
				class="add-row-button"
				:disabled="isBusy"
				@click="addRow(rows.length - 1)"
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
				@click="createNewSubtasks"
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
import {createRandomID} from '@/helpers/randomId'

interface SubtaskRow {
	key: string
	taskId: number | null
	title: string
	weight: number
	assignees: IUser[]
	done: boolean
}

const props = defineProps<{
	taskId: ITask['id'],
	projectId: ITask['projectId'],
	parentTask?: ITask,
}>()

const emit = defineEmits<{
	'created': [tasks: ITask[]],
	'changed': [],
}>()

const {t} = useI18n({useScope: 'global'})
const taskStore = useTaskStore()

const rows = reactive<SubtaskRow[]>([])
const inputRefs = ref<Record<number, HTMLInputElement | null>>({})
const isCreating = ref(false)
const isSaving = ref(false)
const errorMessage = ref('')

const isBusy = computed(() => isCreating.value || isSaving.value)

function setInputRef(index: number, el: HTMLInputElement | Element | ComponentPublicInstance | null) {
	inputRefs.value[index] = el as HTMLInputElement | null
}

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

function makeEmptyRow(): SubtaskRow {
	return {
		key: createRandomID(12),
		taskId: null,
		title: '',
		weight: 0,
		assignees: [],
		done: false,
	}
}

function rowFromTask(task: ITask): SubtaskRow {
	return {
		key: `task-${task.id}`,
		taskId: task.id,
		title: task.title,
		weight: Number(task.subtaskWeight) || 0,
		assignees: Array.isArray(task.assignees) ? [...task.assignees] : [],
		done: !!task.done,
	}
}

// Sync rows with parentTask.relatedTasks.subtask. Existing tasks become
// pre-filled rows; one empty row at the end is kept so the user can add new
// subtasks. Local edits on existing rows are preserved by matching on taskId.
watch(
	() => props.parentTask?.relatedTasks?.[RELATION_KIND.SUBTASK],
	(subtasks) => {
		const existing = Array.isArray(subtasks) ? subtasks : []
		const pendingRows = rows.filter(r => r.taskId === null && r.title.trim().length > 0)

		const newRows: SubtaskRow[] = existing.map(t => rowFromTask(t))
		newRows.push(...pendingRows)
		if (newRows.length === 0) {
			newRows.push(makeEmptyRow())
		}
		rows.splice(0, rows.length, ...newRows)
	},
	{immediate: true, deep: true},
)

const totalWeight = computed(() => {
	return rows.reduce((sum, r) => sum + (Number(r.weight) || 0), 0)
})

const canCreate = computed(() => {
	return rows.some(r => r.taskId === null && r.title.trim().length > 0)
})

function addRow(afterIndex: number) {
	rows.splice(afterIndex + 1, 0, makeEmptyRow())
	nextTick(() => {
		inputRefs.value[afterIndex + 1]?.focus()
	})
}

async function removeRow(index: number) {
	const row = rows[index]
	if (!row) return

	if (row.taskId) {
		// Remove the subtask relation from the parent
		isSaving.value = true
		try {
			const taskRelationService = new TaskRelationService()
			await taskRelationService.delete(new TaskRelationModel({
				taskId: props.taskId,
				otherTaskId: row.taskId,
				relationKind: RELATION_KIND.SUBTASK,
			}))
			rows.splice(index, 1)
			if (rows.length === 0) {
				rows.push(makeEmptyRow())
			}
			emit('changed')
		} finally {
			isSaving.value = false
		}
		return
	}

	rows.splice(index, 1)
	if (rows.length === 0) {
		rows.push(makeEmptyRow())
	}
}

async function onWeightChange(row: SubtaskRow) {
	if (!row.taskId) {
		return
	}
	const parsed = Number(row.weight)
	const weight = Number.isFinite(parsed) && parsed >= 0 ? Math.min(parsed, 100) : 0
	row.weight = weight

	const existing = (props.parentTask?.relatedTasks?.[RELATION_KIND.SUBTASK] || [])
		.find(t => t.id === row.taskId)
	if (!existing || weight === (Number(existing.subtaskWeight) || 0)) {
		return
	}

	isSaving.value = true
	try {
		await taskStore.update(new TaskModel({...existing, subtaskWeight: weight}))
		emit('changed')
	} finally {
		isSaving.value = false
	}
}

async function onAssigneeAdded(row: SubtaskRow, user: IUser) {
	// Multiselect already pushed the user into row.assignees in-place
	// (it mutates the bound array). For pending rows, that's all we need.
	if (!row.taskId) {
		return
	}
	isSaving.value = true
	try {
		await taskStore.addAssignee({user, taskId: row.taskId})
		emit('changed')
	} finally {
		isSaving.value = false
	}
}

async function onAssigneeRemoved(row: SubtaskRow, user: IUser) {
	if (!row.taskId) {
		return
	}
	isSaving.value = true
	try {
		await taskStore.removeAssignee({user, taskId: row.taskId})
		emit('changed')
	} finally {
		isSaving.value = false
	}
}

async function createNewSubtasks() {
	const pending = rows.filter(r => r.taskId === null && r.title.trim().length > 0)
	if (pending.length === 0 || isCreating.value) {
		return
	}

	isCreating.value = true
	errorMessage.value = ''

	const taskService = new TaskService()
	const taskRelationService = new TaskRelationService()
	const createdTasks: ITask[] = []

	try {
		for (const row of pending) {
			const weight = Number(row.weight) || 0
			const newTask = await taskService.create(new TaskModel({
				title: row.title.trim(),
				projectId: props.projectId,
				subtaskWeight: weight,
			}))

			await taskRelationService.create(new TaskRelationModel({
				taskId: props.taskId,
				otherTaskId: newTask.id,
				relationKind: RELATION_KIND.SUBTASK,
			}))

			for (const u of row.assignees) {
				await taskStore.addAssignee({user: u, taskId: newTask.id})
			}

			createdTasks.push(newTask)
		}

		// Clear the pending rows so they don't get re-rendered alongside
		// the freshly fetched existing subtasks when the parent reloads.
		for (const row of pending) {
			const idx = rows.indexOf(row)
			if (idx >= 0) rows.splice(idx, 1)
		}
		if (rows.length === 0 || rows.every(r => r.taskId !== null)) {
			rows.push(makeEmptyRow())
		}

		emit('created', createdTasks)
		success({message: t('task.decompose.success', {count: createdTasks.length})})
	} catch (e) {
		errorMessage.value = t('task.decompose.error')
		throw e
	} finally {
		isCreating.value = false
	}
}

function focus() {
	nextTick(() => {
		const idx = rows.findIndex(r => r.taskId === null)
		if (idx >= 0) inputRefs.value[idx]?.focus()
	})
}

defineExpose({focus})
</script>

<style lang="scss" scoped>
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
	}

	.subtask-link {
		color: var(--text);
		text-decoration: none;
		padding: 0.4rem 0.5rem;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;

		&:hover {
			color: var(--primary);
		}

		&.is-strikethrough {
			text-decoration: line-through;
			color: var(--grey-400);
		}
	}

	.assignee-wrapper {
		flex-shrink: 0;
		inline-size: 10rem;

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

	.error-notice {
		font-size: 0.85rem;
	}
}
</style>
