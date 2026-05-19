<template>
	<div class="task-decomposition">
		<p class="help-text">
			{{ $t('task.decompose.help') }}
		</p>
<<<<<<< HEAD

		<div
			v-if="existingSubtasks.length > 0"
			class="existing-subtasks"
		>
			<div
				v-for="subtask in existingSubtasks"
				:key="subtask.id"
				class="subtask-row"
				:class="{'is-done': subtask.done}"
			>
				<span class="row-title">
					<Icon
						v-if="subtask.done"
						icon="check"
						class="done-icon"
					/>
					<RouterLink
						v-tooltip="subtask.title"
						class="subtask-link"
						:class="{'is-strikethrough': subtask.done}"
						:to="{name: 'task.detail', params: {id: subtask.id}, state: {backdropView: $route.fullPath}}"
					>
						{{ subtask.title }}
					</RouterLink>
				</span>
				<div class="weight-wrapper">
					<input
=======
		<div class="subtask-list">
			<div
				v-for="(item, index) in subtasks"
				:key="index"
				class="subtask-row"
			>
				<span class="row-number">{{ index + 1 }}.</span>
				<input
					:ref="el => setInputRef(index, el)"
					v-model="item.title"
					type="text"
					class="input subtask-title"
					:placeholder="$t('task.decompose.titlePlaceholder')"
					:disabled="isCreating"
					@keydown.enter.prevent="addRow(index)"
				>
				<div class="assignee-wrapper">
					<Multiselect
						v-model="item.assignee"
						:placeholder="$t('task.decompose.assigneePlaceholder')"
						:loading="userSearchLoading"
						:search-results="foundUsers"
						label="name"
						:multiple="false"
						:disabled="isCreating"
						@search="findUser"
						@select="(user: IUser) => item.assignee = user"
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
						v-model.number="item.weight"
>>>>>>> 7fc1c720ece1da4740ed7550883c54ba254bb738
						type="number"
						class="input subtask-weight"
						min="0"
						max="100"
<<<<<<< HEAD
						step="10"
						:value="subtask.subtaskWeight || ''"
						:disabled="isSaving"
						placeholder="auto"
						@change="updateWeight(subtask, ($event.target as HTMLInputElement).value)"
						@keydown.enter.prevent="($event.target as HTMLInputElement).blur()"
=======
						:disabled="isCreating"
						@keydown.enter.prevent="addRow(index)"
>>>>>>> 7fc1c720ece1da4740ed7550883c54ba254bb738
					>
					<span class="weight-sign">%</span>
				</div>
				<BaseButton
<<<<<<< HEAD
					class="remove-row"
					:disabled="isSaving"
					@click="removeSubtask(subtask)"
=======
					v-if="subtasks.length > 1"
					class="remove-row"
					:disabled="isCreating"
					@click="removeRow(index)"
>>>>>>> 7fc1c720ece1da4740ed7550883c54ba254bb738
				>
					<Icon icon="times" />
				</BaseButton>
			</div>
<<<<<<< HEAD
			<div class="weight-summary">
				{{ $t('task.decompose.totalWeight') }}: {{ totalWeight }}%
			</div>
		</div>

		<div class="field">
			<textarea
				ref="textareaRef"
				v-model="subtaskText"
				class="textarea"
				:placeholder="$t('task.decompose.placeholder')"
				rows="6"
				:disabled="isCreating"
				@keydown.meta.enter="createSubtasks"
				@keydown.ctrl.enter="createSubtasks"
			/>
=======
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
>>>>>>> 7fc1c720ece1da4740ed7550883c54ba254bb738
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
				v-if="createdCount > 0"
				class="created-notice has-text-success"
			>
				{{ $t('task.decompose.created', {count: createdCount}) }}
			</span>
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
import {ref, reactive, computed, shallowReactive, nextTick, type ComponentPublicInstance} from 'vue'
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
<<<<<<< HEAD
import BaseButton from '@/components/base/BaseButton.vue'
=======

interface SubtaskRow {
	title: string
	weight: number
	assignee: IUser | null
}
>>>>>>> 7fc1c720ece1da4740ed7550883c54ba254bb738

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
<<<<<<< HEAD
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const subtaskText = ref('')
=======

const subtasks = reactive<SubtaskRow[]>([
	{title: '', weight: 100, assignee: null},
])

const inputRefs = ref<Record<number, HTMLInputElement | null>>({})

function setInputRef(index: number, el: HTMLInputElement | Element | ComponentPublicInstance | null) {
	inputRefs.value[index] = el as HTMLInputElement | null
}

>>>>>>> 7fc1c720ece1da4740ed7550883c54ba254bb738
const isCreating = ref(false)
const isSaving = ref(false)
const createdCount = ref(0)
const errorMessage = ref('')

<<<<<<< HEAD
const existingSubtasks = computed<ITask[]>(() => {
	return props.parentTask?.relatedTasks?.[RELATION_KIND.SUBTASK] || []
})

const totalWeight = computed(() => {
	return existingSubtasks.value.reduce((sum, s) => sum + (Number(s.subtaskWeight) || 0), 0)
})

const canCreate = computed(() => {
	const lines = subtaskText.value
		.split(/[\r\n]+/)
		.filter(l => l.trim().length > 0)
	return lines.length > 0
})

function focus() {
	nextTick(() => textareaRef.value?.focus())
=======
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
>>>>>>> 7fc1c720ece1da4740ed7550883c54ba254bb738
}

const totalWeight = computed(() => {
	return subtasks.reduce((sum, item) => sum + (item.weight || 0), 0)
})

const canCreate = computed(() => {
	return subtasks.some(item => item.title.trim().length > 0) && totalWeight.value <= 100
})

function addRow(afterIndex: number) {
	const currentTotal = subtasks.reduce((sum, item) => sum + (item.weight || 0), 0)
	const remaining = Math.max(0, 100 - currentTotal)
	subtasks.splice(afterIndex + 1, 0, {title: '', weight: remaining, assignee: null})
	nextTick(() => {
		inputRefs.value[afterIndex + 1]?.focus()
	})
}

function removeRow(index: number) {
	if (subtasks.length <= 1) return
	subtasks.splice(index, 1)
}

function focus() {
	nextTick(() => inputRefs.value[0]?.focus())
}

defineExpose({focus})

<<<<<<< HEAD
async function updateWeight(subtask: ITask, rawValue: string) {
	const parsed = rawValue === '' ? 0 : Number(rawValue)
	const weight = Number.isFinite(parsed) && parsed >= 0 ? Math.min(parsed, 100) : 0
	if (weight === (Number(subtask.subtaskWeight) || 0)) {
		return
	}
	isSaving.value = true
	try {
		await taskStore.update(new TaskModel({...subtask, subtaskWeight: weight}))
		emit('changed')
	} finally {
		isSaving.value = false
	}
}

async function removeSubtask(subtask: ITask) {
	isSaving.value = true
	try {
		const taskRelationService = new TaskRelationService()
		await taskRelationService.delete(new TaskRelationModel({
			taskId: props.taskId,
			otherTaskId: subtask.id,
			relationKind: RELATION_KIND.SUBTASK,
		}))
		emit('changed')
	} finally {
		isSaving.value = false
	}
}

=======
>>>>>>> 7fc1c720ece1da4740ed7550883c54ba254bb738
async function createSubtasks() {
	const validSubtasks = subtasks.filter(item => item.title.trim().length > 0)

	if (validSubtasks.length === 0 || isCreating.value || totalWeight.value > 100) {
		return
	}

	isCreating.value = true
	errorMessage.value = ''
	createdCount.value = 0

	const taskService = new TaskService()
	const taskRelationService = new TaskRelationService()
	const createdTasks: ITask[] = []

	try {
		for (const item of validSubtasks) {
			// Store weight as parseable marker + human-readable text
			const description = `<!-- decompose-weight:${item.weight} -->\n${t('task.decompose.weightLabel', {weight: item.weight})}`

			const newTask = await taskService.create(new TaskModel({
				title: item.title.trim(),
				description,
				projectId: props.projectId,
			}))

			await taskRelationService.create(new TaskRelationModel({
				taskId: props.taskId,
				otherTaskId: newTask.id,
				relationKind: RELATION_KIND.SUBTASK,
			}))

			// Assign user if selected
			if (item.assignee) {
				await taskStore.addAssignee({user: item.assignee, taskId: newTask.id})
			}

			createdTasks.push(newTask)
		}

		createdCount.value = createdTasks.length
		subtasks.splice(0, subtasks.length, {title: '', weight: 100, assignee: null})
		emit('created', createdTasks)
		success({message: t('task.decompose.success', {count: createdTasks.length})})
	} catch (e) {
		errorMessage.value = t('task.decompose.error')
		throw e
	} finally {
		isCreating.value = false
	}
}
</script>

<style lang="scss" scoped>
.task-decomposition {
	.help-text {
		color: var(--grey-500);
		font-size: 0.9rem;
		margin-block-end: 0.75rem;
	}

<<<<<<< HEAD
	.existing-subtasks {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		margin-block-end: 1rem;
		padding-block-end: 0.75rem;
		border-block-end: 1px solid var(--border);
=======
	.subtask-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
>>>>>>> 7fc1c720ece1da4740ed7550883c54ba254bb738
	}

	.subtask-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
<<<<<<< HEAD

		&.is-done .row-title {
			color: var(--grey-500);
		}
	}

	.row-title {
		flex: 1;
		min-inline-size: 0;
		display: flex;
		align-items: center;
		gap: 0.4rem;
		overflow: hidden;
	}

	.done-icon {
		color: var(--success);
		flex-shrink: 0;
	}

	.subtask-link {
		color: var(--text);
		text-decoration: none;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;

		&:hover {
			color: var(--primary);
		}

		&.is-strikethrough {
			text-decoration: line-through;
			color: var(--grey-400);
=======
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
>>>>>>> 7fc1c720ece1da4740ed7550883c54ba254bb738
		}
	}

	.weight-wrapper {
		display: flex;
		align-items: center;
		gap: 0.15rem;
		flex-shrink: 0;
	}

	.subtask-weight {
<<<<<<< HEAD
		inline-size: 4.5rem;
		padding: 0.3rem 0.4rem;
		text-align: center;
		font-size: 0.85rem;
		border: 1px solid var(--border);
		border-radius: $radius;
		background: var(--scheme-main);
		color: var(--text);

		&:focus {
			border-color: var(--primary);
			outline: none;
		}
=======
		inline-size: 4rem;
		text-align: center;
>>>>>>> 7fc1c720ece1da4740ed7550883c54ba254bb738
	}

	.weight-sign {
		color: var(--grey-500);
<<<<<<< HEAD
		font-size: 0.85rem;
	}

	.remove-row {
		color: var(--danger);
		padding: 0.25rem;
		flex-shrink: 0;
		line-height: 1;
	}

	.weight-summary {
		font-size: 0.85rem;
		color: var(--grey-600);
		font-weight: 600;
		text-align: end;
	}

	.textarea {
		font-family: monospace;
=======
>>>>>>> 7fc1c720ece1da4740ed7550883c54ba254bb738
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