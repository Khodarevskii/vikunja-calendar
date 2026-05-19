<template>
	<div class="task-decomposition">
		<p class="help-text">
			{{ $t('task.decompose.help') }}
		</p>

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
						type="number"
						class="input subtask-weight"
						min="0"
						max="100"
						step="10"
						:value="subtask.subtaskWeight || ''"
						:disabled="isSaving"
						placeholder="auto"
						@change="updateWeight(subtask, ($event.target as HTMLInputElement).value)"
						@keydown.enter.prevent="($event.target as HTMLInputElement).blur()"
					>
					<span class="weight-sign">%</span>
				</div>
				<BaseButton
					class="remove-row"
					:disabled="isSaving"
					@click="removeSubtask(subtask)"
				>
					<Icon icon="times" />
				</BaseButton>
			</div>
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
import {ref, computed, nextTick} from 'vue'
import {useI18n} from 'vue-i18n'
import TaskService from '@/services/task'
import TaskModel from '@/models/task'
import TaskRelationService from '@/services/taskRelation'
import TaskRelationModel from '@/models/taskRelation'
import {RELATION_KIND} from '@/types/IRelationKind'
import type {ITask} from '@/modelTypes/ITask'
import {success} from '@/message'
import {useTaskStore} from '@/stores/tasks'
import BaseButton from '@/components/base/BaseButton.vue'

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
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const subtaskText = ref('')
const isCreating = ref(false)
const isSaving = ref(false)
const createdCount = ref(0)
const errorMessage = ref('')

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
}
defineExpose({focus})

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

async function createSubtasks() {
	if (!canCreate.value || isCreating.value) {
		return
	}
	const lines = subtaskText.value
		.split(/[\r\n]+/)
		.filter(l => l.trim().length > 0)
		.map(l => l.trim())
	if (lines.length === 0) {
		return
	}
	isCreating.value = true
	errorMessage.value = ''
	createdCount.value = 0
	const taskService = new TaskService()
	const taskRelationService = new TaskRelationService()
	const createdTasks: ITask[] = []
	try {
		for (const title of lines) {
			const newTask = await taskService.create(new TaskModel({
				title,
				projectId: props.projectId,
			}))
			await taskRelationService.create(new TaskRelationModel({
				taskId: props.taskId,
				otherTaskId: newTask.id,
				relationKind: RELATION_KIND.SUBTASK,
			}))
			createdTasks.push(newTask)
		}
		createdCount.value = createdTasks.length
		subtaskText.value = ''
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
		margin-block-end: 0.5rem;
	}

	.existing-subtasks {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		margin-block-end: 1rem;
		padding-block-end: 0.75rem;
		border-block-end: 1px solid var(--border);
	}

	.subtask-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;

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
		}
	}

	.weight-wrapper {
		display: flex;
		align-items: center;
		gap: 0.15rem;
		flex-shrink: 0;
	}

	.subtask-weight {
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
	}

	.weight-sign {
		color: var(--grey-500);
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
		font-size: 0.9rem;
		resize: vertical;
		min-block-size: 100px;
		inline-size: 100%;
		padding: 0.75rem;
		border: 1px solid var(--border);
		border-radius: $radius;
		background: var(--scheme-main);
		color: var(--text);
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
	.actions {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin-block-start: 0.5rem;
	}
	.created-notice,
	.error-notice {
		font-size: 0.85rem;
	}
}
</style>
