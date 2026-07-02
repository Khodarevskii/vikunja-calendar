<template>
	<Modal
		:enabled="enabled"
		:overflow="true"
		wide
		@close="onClose"
		@submit="submit"
	>
		<template #header>
			<span>{{ $t('task.feedback.modalTitle') }}</span>
		</template>

		<template #text>
			<div class="feedback-form">
				<textarea
					v-model="text"
					class="input feedback-text"
					rows="6"
					:placeholder="$t('task.feedback.textPlaceholder')"
					:disabled="isSending"
				/>

				<div
					class="dropzone"
					:class="{'is-active': dragActive}"
					@dragover.prevent="dragActive = true"
					@dragenter.prevent="dragActive = true"
					@dragleave.prevent="dragActive = false"
					@drop.prevent="handleDrop"
				>
					<Icon icon="cloud-upload-alt" />
					<span>{{ $t('task.feedback.dropHint') }}</span>
					<BaseButton
						class="pick-file"
						:disabled="isSending"
						@click="fileInput?.click()"
					>
						{{ $t('task.feedback.pickFile') }}
					</BaseButton>
					<input
						ref="fileInput"
						type="file"
						multiple
						hidden
						@change="handlePick"
					>
				</div>

				<ul
					v-if="files.length > 0"
					class="file-list"
				>
					<li
						v-for="(file, index) in files"
						:key="index"
					>
						<Icon
							:icon="iconFor(file.name)"
							class="file-icon"
						/>
						<span class="file-name">{{ file.name }}</span>
						<span class="file-size">{{ humanSize(file.size) }}</span>
						<BaseButton
							class="file-remove"
							:disabled="isSending"
							@click="removeFile(index)"
						>
							<Icon icon="times" />
						</BaseButton>
					</li>
				</ul>
			</div>
		</template>

		<template #buttons>
			<XButton
				variant="tertiary"
				:disabled="isSending"
				@click="onClose"
			>
				{{ $t('misc.cancel') }}
			</XButton>
			<XButton
				:loading="isSending"
				:disabled="!canSubmit"
				@click="submit"
			>
				{{ $t('task.feedback.send') }}
			</XButton>
		</template>
	</Modal>
</template>

<script setup lang="ts">
import {ref, computed, watch} from 'vue'
import {useI18n} from 'vue-i18n'

import BaseButton from '@/components/base/BaseButton.vue'
import AttachmentService from '@/services/attachment'
import AttachmentModel from '@/models/attachment'
import {FeedbackSubmissionService} from '@/services/taskFeedback'
import {success as msgSuccess, error as msgError} from '@/message'
import type {ITask} from '@/modelTypes/ITask'

const props = defineProps<{
	enabled: boolean,
	task: ITask,
}>()

const emit = defineEmits<{
	'close': [],
	'submitted': [],
}>()

const {t} = useI18n({useScope: 'global'})

const text = ref('')
const files = ref<File[]>([])
const dragActive = ref(false)
const isSending = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

const canSubmit = computed(() => text.value.trim().length > 0 || files.value.length > 0)

watch(() => props.enabled, (v) => {
	if (v) {
		text.value = ''
		files.value = []
		dragActive.value = false
	}
})

function onClose() {
	if (isSending.value) return
	emit('close')
}

function handleDrop(event: DragEvent) {
	dragActive.value = false
	const dropped = event.dataTransfer?.files
	if (!dropped) return
	for (const f of Array.from(dropped)) files.value.push(f)
}

function handlePick(event: Event) {
	const input = event.target as HTMLInputElement
	if (!input.files) return
	for (const f of Array.from(input.files)) files.value.push(f)
	input.value = ''
}

function removeFile(index: number) {
	files.value.splice(index, 1)
}

function humanSize(bytes: number): string {
	if (bytes < 1024) return `${bytes} B`
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
	if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`
	return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`
}

// Map common extensions to Font Awesome icons already used elsewhere in the
// codebase (FA5 free set — icons like file-word, file-pdf, file-image…).
function iconFor(name: string): string {
	const ext = name.toLowerCase().split('.').pop() || ''
	if (['pdf'].includes(ext)) return 'file-pdf'
	if (['doc', 'docx', 'rtf', 'odt'].includes(ext)) return 'file-word'
	if (['xls', 'xlsx', 'ods', 'csv'].includes(ext)) return 'file-excel'
	if (['ppt', 'pptx', 'odp'].includes(ext)) return 'file-powerpoint'
	if (['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp'].includes(ext)) return 'file-image'
	if (['mp4', 'mov', 'avi', 'mkv', 'webm'].includes(ext)) return 'file-video'
	if (['mp3', 'wav', 'flac', 'ogg', 'm4a'].includes(ext)) return 'file-audio'
	if (['zip', 'rar', '7z', 'tar', 'gz', 'bz2'].includes(ext)) return 'file-archive'
	if (['txt', 'md', 'log'].includes(ext)) return 'file-alt'
	if (['js', 'ts', 'tsx', 'vue', 'go', 'py', 'rb', 'php', 'java', 'c', 'cpp', 'h', 'json', 'xml', 'html', 'css', 'sh'].includes(ext)) return 'file-code'
	return 'file'
}

async function submit() {
	if (!canSubmit.value || isSending.value) return

	isSending.value = true
	try {
		const attachmentIds: number[] = []
		if (files.value.length > 0) {
			const attachmentService = new AttachmentService()
			const model = new AttachmentModel({taskId: props.task.id})
			const response = await attachmentService.create(model, files.value)
			for (const a of response.success || []) {
				attachmentIds.push(a.id)
			}
			if (response.errors && response.errors.length > 0) {
				const messages = response.errors.map((e: {message: string}) => e.message)
				throw new Error(messages.join('\n'))
			}
		}

		const submissionService = new FeedbackSubmissionService()
		await submissionService.create({
			taskId: props.task.id,
			text: text.value.trim(),
			attachmentIds,
		})

		msgSuccess({message: t('task.feedback.sent')})
		emit('submitted')
		emit('close')
	} catch (e) {
		msgError(e)
	} finally {
		isSending.value = false
	}
}
</script>

<style lang="scss" scoped>
.feedback-form {
	display: flex;
	flex-direction: column;
	gap: 0.75rem;
}

.feedback-text {
	inline-size: 100%;
	padding: 0.6rem 0.75rem;
	border: 1px solid var(--border);
	border-radius: $radius;
	background: var(--scheme-main);
	color: var(--text);
	resize: vertical;
	min-block-size: 120px;
	font-size: 0.95rem;

	&:focus {
		border-color: var(--primary);
		outline: none;
	}
}

.dropzone {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	gap: 0.5rem;
	padding: 1.25rem;
	border: 1px dashed var(--border);
	border-radius: $radius;
	background: var(--scheme-main-bis);
	color: var(--grey-600);
	transition: border-color .15s, background .15s;

	&.is-active {
		border-color: var(--primary);
		background: var(--scheme-main);
		color: var(--text);
	}
}

.pick-file {
	color: var(--primary);
	font-weight: 600;
}

.file-list {
	list-style: none;
	padding: 0;
	margin: 0;
	display: flex;
	flex-direction: column;
	gap: 0.35rem;

	li {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.45rem 0.6rem;
		border: 1px solid var(--border);
		border-radius: $radius;
		background: var(--scheme-main);
	}
}

.file-icon {
	color: var(--primary);
	flex-shrink: 0;
}

.file-name {
	flex: 1;
	min-inline-size: 0;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
	color: var(--text);
}

.file-size {
	color: var(--grey-500);
	font-size: 0.85rem;
}

.file-remove {
	color: var(--danger);
	padding: 0.15rem;
	line-height: 1;
}
</style>
