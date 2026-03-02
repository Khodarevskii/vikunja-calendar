<template>
	<div class="task-decomposition">
		<p class="help-text">
			{{ $t('task.decompose.help') }}
		</p>
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

const props = defineProps<{
	taskId: ITask['id'],
	projectId: ITask['projectId'],
}>()

const emit = defineEmits<{
	'created': [tasks: ITask[]],
}>()

const {t} = useI18n({useScope: 'global'})

const textareaRef = ref<HTMLTextAreaElement | null>(null)
const subtaskText = ref('')
const isCreating = ref(false)
const createdCount = ref(0)
const errorMessage = ref('')

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
