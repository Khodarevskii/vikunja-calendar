<template>
	<div>
		<h3>
			<span class="icon is-grey">
				<Icon icon="align-left" />
			</span>
			{{ $t('task.attributes.description') }}
			<span
				v-if="canWrite"
				class="weight-inline"
			>
				<label class="weight-label">{{ $t('task.relation.subtaskWeight') }}:</label>
				<input
					v-model.number="weightValue"
					type="number"
					min="0"
					max="100"
					step="10"
					class="input weight-input"
					placeholder="0"
					@change="saveWeight"
					@blur="saveWeight"
					@keydown.enter.prevent="($event.target as HTMLInputElement).blur()"
				>
				<span class="weight-sign">%</span>
			</span>
			<CustomTransition name="fade">
				<span
					v-if="loading && saving"
					class="is-small is-inline-flex"
				>
					<span class="loader is-inline-block mie-2" />
					{{ $t('misc.saving') }}
				</span>
				<span
					v-else-if="!loading && saved"
					class="is-small has-text-success"
				>
					<Icon icon="check" />
					{{ $t('misc.saved') }}
				</span>
			</CustomTransition>
		</h3>
		<Editor
			v-model="description"
			class="tiptap__task-description"
			:is-edit-enabled="canWrite"
			:upload-callback="uploadCallback"
			:placeholder="$t('task.description.placeholder')"
			:show-save="true"
			edit-shortcut="e"
			:enable-discard-shortcut="true"
			:enable-mentions="true"
			:mention-project-id="modelValue.projectId"
			:storage-key="descriptionStorageKey"
			@update:modelValue="saveWithDelay"
			@save="save"
		/>
	</div>
</template>

<script setup lang="ts">
import {ref, computed, watchEffect,  onBeforeUnmount} from 'vue'
import {onBeforeRouteLeave} from 'vue-router'

import CustomTransition from '@/components/misc/CustomTransition.vue'
import Editor from '@/components/input/AsyncEditor'

import { clearEditorDraft } from '@/helpers/editorDraftStorage'
import type { ITask } from '@/modelTypes/ITask'
import { useTaskStore } from '@/stores/tasks'

export type AttachmentUploadFunction = (file: File, onSuccess: (attachmentUrl: string) => void) => Promise<string>

const props = defineProps<{
	modelValue: ITask,
	attachmentUpload: AttachmentUploadFunction,
	canWrite: boolean,
}>()

const emit = defineEmits<{
	'update:modelValue': [value: ITask]
}>()

const description = ref<string>('')
const hasChanges = ref(false)
const weightValue = ref<number>(0)
watchEffect(() => {
	description.value = props.modelValue.description
	weightValue.value = Number(props.modelValue.subtaskWeight) || 0
	hasChanges.value = false
})

const saved = ref(false)

// Since loading is global state, this variable ensures we're only showing the saving icon when saving the description.
const saving = ref(false)

const taskStore = useTaskStore()
const loading = computed(() => taskStore.isLoading)

const changeTimeout = ref<ReturnType<typeof setTimeout> | null>(null)

const descriptionStorageKey = computed(() => `task-description-${props.modelValue.id}`)

async function saveWithDelay() {
	if (description.value === props.modelValue.description) {
		hasChanges.value = false
		if (changeTimeout.value !== null) {
			clearTimeout(changeTimeout.value)
		}
		return
	}

	hasChanges.value = true
	if (changeTimeout.value !== null) {
		clearTimeout(changeTimeout.value)
	}

	changeTimeout.value = setTimeout(async () => {
		await save()
	}, 5000)
}

onBeforeUnmount(async () => {
	await save() // Save before unmounting to handle modal race condition
	if (changeTimeout.value !== null) {
		clearTimeout(changeTimeout.value)
	}
})

onBeforeRouteLeave(() => save())

async function save() {
	if (!hasChanges.value) {
		return
	}

	hasChanges.value = false
	if (changeTimeout.value !== null) {
		clearTimeout(changeTimeout.value)
	}
	saved.value = false
	saving.value = true

	try {
		const updated = await taskStore.update({
			...props.modelValue,
			description: description.value,
		})
		emit('update:modelValue', updated)

		// Clear draft from localStorage when saved successfully
		clearEditorDraft(descriptionStorageKey.value)

		saved.value = true
		setTimeout(() => {
			saved.value = false
		}, 2000)
	} catch (error) {
		// If the task was deleted (404), silently skip saving
		if (error?.response?.status === 404) {
			return
		}
		hasChanges.value = true
		// Re-throw other errors
		throw error
	} finally {
		saving.value = false
	}
}

async function saveWeight() {
	const parsed = Number(weightValue.value)
	const weight = Number.isFinite(parsed) && parsed >= 0 ? Math.min(parsed, 100) : 0
	if (weight === (Number(props.modelValue.subtaskWeight) || 0)) {
		return
	}
	weightValue.value = weight
	try {
		const updated = await taskStore.update({
			...props.modelValue,
			subtaskWeight: weight,
		})
		emit('update:modelValue', updated)
	} catch (error) {
		if (error?.response?.status === 404) {
			return
		}
		throw error
	}
}

async function uploadCallback(files: File[] | FileList): Promise<string[]> {
	const uploadPromises: Promise<string>[] = []

	files.forEach((file: File) => {
		const promise = new Promise<string>((resolve) => {
			props.attachmentUpload(file, (uploadedFileUrl: string) => resolve(uploadedFileUrl))
		})

		uploadPromises.push(promise)
	})

	return await Promise.all(uploadPromises)
}
</script>

<style lang="scss" scoped>
.tiptap__task-description {
	// The exact amount of pixels we need to make the description icon align with the buttons and the form inside the editor.
	// The icon is not exactly the same length on all sides so we need to hack our way around it.
	margin-inline-start: 4px;
}

.weight-inline {
	display: inline-flex;
	align-items: center;
	gap: 0.25rem;
	margin-inline-start: 1rem;
	font-size: 0.85rem;
	font-weight: 400;
	color: var(--grey-600);
}

.weight-label {
	color: var(--grey-600);
}

.weight-input {
	inline-size: 4.5rem;
	padding: 0.25rem 0.4rem;
	font-size: 0.85rem;
	border: 1px solid var(--border);
	border-radius: $radius;
	background: var(--scheme-main);
	color: var(--text);
	text-align: center;

	&:focus {
		border-color: var(--primary);
		outline: none;
	}
}

.weight-sign {
	color: var(--grey-500);
}
</style>

