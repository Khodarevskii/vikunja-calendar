<template>
	<div class="task-decomposition">
		<p class="help-text">
			{{ $t('task.decompose.help') }}
		</p>
		<div class="subtask-list">
			<div
				v-for="(item, index) in subtasks"
				:key="index"
				class="subtask-row"
				:class="{'is-existing': item.existingTaskId, 'is-done': item.done}"
			>
				<span class="row-number">{{ index + 1 }}.</span>
				<span
					v-if="item.existingTaskId"
					class="subtask-title existing-title"
					:class="{'is-strikethrough': item.done}"
				>
					<Icon
						v-if="item.done"
						icon="check"
						class="done-icon"
					/>
					{{ item.title }}
				</span>
				<input
					v-else
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
						v-if="!item.existingTaskId"
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
					<span
						v-else-if="item.assignee"
						class="existing-assignee"
					>
						<User
							:avatar-size="20"
							:show-username="false"
							:user="item.assignee"
						/>
					</span>
				</div>
				<div class="weight-wrapper">
					<input
						v-if="!item.existingTaskId"
						v-model.number="item.weight"
						type="number"
						class="input subtask-weight"
						min="0"
						max="100"
						:disabled="isCreating"
						@keydown.enter.prevent="addRow(index)"
					>
					<span
						v-else
						class="subtask-weight existing-weight"
					>{{ item.weight }}</span>
					<span class="weight-sign">%</span>
				</div>
				<BaseButton
					v-if="subtasks.length > 1 && !item.existingTaskId"
					class="remove-row"
					:disabled="isCreating"
					@click="removeRow(index)"
				>
					<Icon icon="times" />
				</BaseButton>
				<span
					v-else-if="item.existingTaskId"
					class="existing-marker"
				/>
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
	assignee: IUser | null
	existingTaskId?: number
	done?: boolean
}

const WEIGHT_REGEX = /<!-- decompose-weight:(\d+) -->/

const props = defineProps<{
	taskId: ITask['id'],
	projectId: ITask['projectId'],
	existingSubtasks?: ITask[],
}>()

const emit = defineEmits<{
	'created': [tasks: ITask[]],
}>()

const {t} = useI18n({useScope: 'global'})
const taskStore = useTaskStore()

const subtasks = reactive<SubtaskRow[]>([
	{title: '', weight: 100, assignee: null},
])

// Load existing subtasks into the form
watch(
	() => props.existingSubtasks,
	(existing) => {
		if (!existing || existing.length === 0) {
			return
		}

		const rows: SubtaskRow[] = existing.map(sub => {
			const weightMatch = WEIGHT_REGEX.exec(sub.description || '')
			const weight = weightMatch ? parseInt(weightMatch[1], 10) : 0
			const assignee = sub.assignees?.length > 0 ? sub.assignees[0] : null

			return {
				title: sub.title,
				weight,
				assignee,
				existingTaskId: sub.id,
				done: sub.done,
			} as SubtaskRow
		})

		// Add an empty row for new subtasks
		const usedWeight = rows.reduce((sum, r) => sum + r.weight, 0)
		const remaining = Math.max(0, 100 - usedWeight)
		rows.push({title: '', weight: remaining, assignee: null})

		subtasks.splice(0, subtasks.length, ...rows)
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
	return subtasks.some(item => item.title.trim().length > 0 && !item.existingTaskId) && totalWeight.value <= 100
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

async function createSubtasks() {
	const validSubtasks = subtasks.filter(item => item.title.trim().length > 0 && !item.existingTaskId)

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

	.existing-title {
		font-size: 0.9rem;
		color: var(--text);
		display: flex;
		align-items: center;
		gap: 0.35rem;

		&.is-strikethrough {
			text-decoration: line-through;
			color: var(--grey-400);
		}
	}

	.done-icon {
		color: var(--success);
		font-size: 0.8rem;
	}

	.is-existing {
		opacity: 0.7;
	}

	.existing-assignee {
		display: flex;
		align-items: center;
	}

	.existing-weight {
		display: inline-block;
		inline-size: 4rem;
		text-align: center;
		font-size: 0.9rem;
		color: var(--grey-500);
	}

	.existing-marker {
		// Placeholder to keep alignment with rows that have a remove button
		inline-size: 1.25rem;
		flex-shrink: 0;
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
