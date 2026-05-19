<template>
	<div class="task-decomposition">
		<p class="help-text">
			{{ $t('task.decompose.help') }}
		</p>
		<div class="subtask-list">
			<div
				v-for="(item, index) in items"
				:key="item.id"
				class="subtask-row"
				:class="{'is-done': item.done}"
			>
				<span class="row-number">{{ index + 1 }}.</span>
				<BaseButton
					class="done-toggle"
					:class="{'is-done': item.done}"
					:disabled="isSaving"
					@click="toggleDone(item)"
				>
					<Icon :icon="item.done ? 'check-square' : ['far', 'square']" />
				</BaseButton>
				<input
					:ref="el => setInputRef(index, el)"
					v-model="item.title"
					type="text"
					class="input subtask-title"
					:class="{'is-strikethrough': item.done}"
					:placeholder="$t('task.decompose.titlePlaceholder')"
					:disabled="isSaving"
					@keydown.enter.prevent="addRow(index)"
					@blur="persist"
				>
				<div class="assignee-wrapper">
					<Multiselect
						:model-value="selectedAssigneeList(item)"
						:placeholder="$t('task.decompose.assigneePlaceholder')"
						:loading="userSearchLoading"
						:search-results="foundUsers"
						label="name"
						:multiple="true"
						:disabled="isSaving"
						@update:model-value="users => setAssignee(item, (users as IUser[])[0] ?? null)"
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
									v-if="!isSaving"
									class="remove-assignee"
									@click="() => setAssignee(item, null)"
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
						:disabled="isSaving"
						@keydown.enter.prevent="addRow(index)"
						@blur="persist"
					>
					<span class="weight-sign">%</span>
				</div>
				<BaseButton
					v-if="items.length > 1"
					class="remove-row"
					:disabled="isSaving"
					@click="removeRow(index)"
				>
					<Icon icon="times" />
				</BaseButton>
			</div>
		</div>
		<div class="list-controls">
			<BaseButton
				class="add-row-button"
				:disabled="isSaving"
				@click="addRow(items.length - 1)"
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
				:loading="isSaving"
				:disabled="!canSave"
				icon="project-diagram"
				@click="persist"
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
import ProjectUserService from '@/services/projectUsers'

import type {ITask, ITaskChecklistItem} from '@/modelTypes/ITask'
import type {IUser} from '@/modelTypes/IUser'

import BaseButton from '@/components/base/BaseButton.vue'
import Multiselect from '@/components/input/Multiselect.vue'
import User from '@/components/misc/User.vue'
import {getDisplayName} from '@/models/user'
import {success} from '@/message'
import {createRandomID} from '@/helpers/randomId'

interface ChecklistRow extends ITaskChecklistItem {
	assignee?: IUser | null
}

const props = defineProps<{
	taskId: ITask['id'],
	projectId: ITask['projectId'],
	parentTask?: ITask,
}>()

const emit = defineEmits<{
	'updated': [items: ITaskChecklistItem[]],
}>()

const {t} = useI18n({useScope: 'global'})

function makeRow(overrides: Partial<ChecklistRow> = {}): ChecklistRow {
	return {
		id: overrides.id ?? createRandomID(16),
		title: overrides.title ?? '',
		weight: overrides.weight ?? 0,
		done: overrides.done ?? false,
		assigneeId: overrides.assigneeId ?? 0,
		position: overrides.position ?? 0,
		assignee: overrides.assignee ?? null,
	}
}

const items = reactive<ChecklistRow[]>([])
const isSaving = ref(false)
const errorMessage = ref('')
const inputRefs = ref<Record<number, HTMLInputElement | null>>({})

function setInputRef(index: number, el: HTMLInputElement | Element | ComponentPublicInstance | null) {
	inputRefs.value[index] = el as HTMLInputElement | null
}

// Load initial state from the parent task.
watch(
	() => props.parentTask?.checklistItems,
	async (existing) => {
		items.splice(0, items.length)
		const list = Array.isArray(existing) ? existing : []
		for (const it of list) {
			items.push(makeRow({
				id: it.id,
				title: it.title,
				weight: it.weight,
				done: it.done,
				assigneeId: it.assigneeId,
				position: it.position,
			}))
		}
		// Ensure at least one empty row so the user can start editing
		if (items.length === 0) {
			items.push(makeRow())
		}
		await resolveAssignees()
	},
	{immediate: true, deep: true},
)

// User search
const projectUserService = shallowReactive(new ProjectUserService())
const userSearchLoading = computed(() => projectUserService.loading)
const foundUsers = ref<IUser[]>([])
const assigneeCache = reactive<Record<number, IUser>>({})

async function findUser(query: string) {
	const response = await projectUserService.getAll({projectId: props.projectId}, {s: query}) as IUser[]
	foundUsers.value = response.map(u => {
		u.name = getDisplayName(u)
		assigneeCache[u.id] = u
		return u
	})
}

async function resolveAssignees() {
	// Fetch every user we don't already have in the cache.
	const needed = items
		.filter(i => i.assigneeId > 0 && !assigneeCache[i.assigneeId])
		.map(i => i.assigneeId)
	if (needed.length === 0) {
		for (const row of items) {
			row.assignee = row.assigneeId ? assigneeCache[row.assigneeId] || null : null
		}
		return
	}

	try {
		const response = await projectUserService.getAll({projectId: props.projectId}, {}) as IUser[]
		for (const u of response) {
			u.name = getDisplayName(u)
			assigneeCache[u.id] = u
		}
	} catch {
		// silently ignore — the assignee just won't be displayed
	}

	for (const row of items) {
		row.assignee = row.assigneeId ? assigneeCache[row.assigneeId] || null : null
	}
}

function selectedAssigneeList(item: ChecklistRow): IUser[] {
	if (!item.assigneeId) {
		return []
	}
	const cached = assigneeCache[item.assigneeId]
	if (cached) {
		return [cached]
	}
	return item.assignee ? [item.assignee] : []
}

function setAssignee(item: ChecklistRow, user: IUser | null) {
	if (!user) {
		item.assigneeId = 0
		item.assignee = null
	} else {
		item.assigneeId = user.id
		item.assignee = user
		assigneeCache[user.id] = user
	}
	persist()
}

const totalWeight = computed(() => {
	return items.reduce((sum, item) => sum + (Number(item.weight) || 0), 0)
})

const canSave = computed(() => {
	return items.some(item => item.title.trim().length > 0)
})

function addRow(afterIndex: number) {
	items.splice(afterIndex + 1, 0, makeRow())
	nextTick(() => {
		inputRefs.value[afterIndex + 1]?.focus()
	})
}

function removeRow(index: number) {
	if (items.length <= 1) return
	items.splice(index, 1)
	persist()
}

function toggleDone(item: ChecklistRow) {
	item.done = !item.done
	persist()
}

function cleanItems(): ITaskChecklistItem[] {
	return items
		.filter(i => i.title.trim().length > 0)
		.map((i, idx) => ({
			id: i.id,
			title: i.title.trim(),
			weight: Number(i.weight) || 0,
			done: i.done,
			assigneeId: i.assigneeId,
			position: idx,
		}))
}

async function persist() {
	if (isSaving.value) {
		return
	}
	isSaving.value = true
	errorMessage.value = ''

	const payload = cleanItems()

	try {
		const taskService = new TaskService()
		const updated = await taskService.update(new TaskModel({
			id: props.taskId,
			title: props.parentTask?.title,
			projectId: props.projectId,
			checklistItems: payload,
		}))
		emit('updated', updated.checklistItems || [])
		if (payload.length > 0) {
			success({message: t('task.decompose.success', {count: payload.length})})
		}
	} catch (e) {
		errorMessage.value = t('task.decompose.error')
		throw e
	} finally {
		isSaving.value = false
	}
}

function focus() {
	nextTick(() => inputRefs.value[0]?.focus())
}

defineExpose({focus})
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

	.done-toggle {
		color: var(--grey-500);
		padding: 0.25rem;
		flex-shrink: 0;
		line-height: 1;

		&.is-done {
			color: var(--success);
		}
	}

	.subtask-title {
		flex: 1;
		min-inline-size: 0;

		&.is-strikethrough {
			text-decoration: line-through;
			color: var(--grey-400);
		}
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
