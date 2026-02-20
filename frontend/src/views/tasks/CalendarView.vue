<template>
	<div class="calendar-view-container">
		<h1 class="title">
			{{ $t('navigation.calendar') }}
		</h1>

		<div class="calendar-wrapper">
			<FullCalendar
				ref="calendarRef"
				:options="calendarOptions"
			/>
		</div>

		<!-- Modal: Create Task -->
		<div
			v-if="showCreateModal"
			class="modal is-active"
			@click.self="closeCreateModal"
		>
			<div class="modal-background" @click="closeCreateModal" />
			<div class="modal-card">
				<header class="modal-card-head">
					<p class="modal-card-title">
						{{ $t('task.new') }}
					</p>
					<button
						class="delete"
						aria-label="close"
						@click="closeCreateModal"
					/>
				</header>
				<section class="modal-card-body">
					<div class="field">
						<label class="label">{{ $t('task.attributes.title') }}</label>
						<div class="control">
							<input
								v-model="newTask.title"
								v-focus
								class="input"
								type="text"
								:placeholder="$t('task.attributes.title')"
								@keyup.enter="createTask"
							/>
						</div>
					</div>
					<div class="field">
						<label class="label">{{ $t('task.attributes.project') }}</label>
						<div class="control">
							<div class="select is-fullwidth">
								<select v-model="newTask.projectId">
									<option
										v-for="project in projects"
										:key="project.id"
										:value="project.id"
									>
										{{ project.title }}
									</option>
								</select>
							</div>
						</div>
					</div>
					<div class="field">
						<label class="label">{{ $t('task.attributes.dueDate') }}</label>
						<div class="control">
							<input
								v-model="newTask.dueDateStr"
								class="input"
								type="datetime-local"
							/>
						</div>
					</div>
					<div class="columns">
						<div class="column">
							<div class="field">
								<label class="label">{{ $t('task.attributes.startDate') }}</label>
								<div class="control">
									<input
										v-model="newTask.startDateStr"
										class="input"
										type="datetime-local"
									/>
								</div>
							</div>
						</div>
						<div class="column">
							<div class="field">
								<label class="label">{{ $t('task.attributes.endDate') }}</label>
								<div class="control">
									<input
										v-model="newTask.endDateStr"
										class="input"
										type="datetime-local"
									/>
								</div>
							</div>
						</div>
					</div>
				</section>
				<footer class="modal-card-foot">
					<XButton
						:loading="creating"
						:disabled="!newTask.title || !newTask.projectId"
						@click="createTask"
					>
						{{ $t('task.new') }}
					</XButton>
					<XButton
						variant="secondary"
						@click="closeCreateModal"
					>
						{{ $t('misc.cancel') }}
					</XButton>
				</footer>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import {ref, computed, onMounted} from 'vue'
import {useRouter} from 'vue-router'
import {useI18n} from 'vue-i18n'
import {setTitle} from '@/helpers/setTitle'

import FullCalendar from '@fullcalendar/vue3'
import type {CalendarOptions, EventInput, DateSelectArg, EventClickArg} from '@fullcalendar/core'
import dayGridPlugin from '@fullcalendar/daygrid'
import timeGridPlugin from '@fullcalendar/timegrid'
import interactionPlugin from '@fullcalendar/interaction'
import listPlugin from '@fullcalendar/list'
import ruLocale from '@fullcalendar/core/locales/ru'

import TaskService from '@/services/task'
import TaskModel from '@/models/task'
import type {ITask} from '@/modelTypes/ITask'
import {useProjectStore} from '@/stores/projects'
import XButton from '@/components/input/Button.vue'

const {t} = useI18n()
const router = useRouter()
const projectStore = useProjectStore()

setTitle(t('navigation.calendar'))

// Calendar ref
const calendarRef = ref<InstanceType<typeof FullCalendar> | null>(null)

// Loading state
const loading = ref(false)

// Tasks displayed on calendar
const calendarEvents = ref<EventInput[]>([])

// Projects list for task creation
const projects = computed(() =>
	projectStore.projectsArray.filter(p => !p.isArchived && p.id > 0),
)

// Create task modal state
const showCreateModal = ref(false)
const creating = ref(false)

interface NewTaskForm {
	title: string
	projectId: number | null
	dueDateStr: string
	startDateStr: string
	endDateStr: string
}

const newTask = ref<NewTaskForm>({
	title: '',
	projectId: null,
	dueDateStr: '',
	startDateStr: '',
	endDateStr: '',
})

// Current visible date range
const currentDateRange = ref<{start: Date, end: Date} | null>(null)

// Convert date to datetime-local string (local time)
function toDatetimeLocal(date: Date): string {
	const pad = (n: number) => String(n).padStart(2, '0')
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

// Convert task to FullCalendar event
function taskToEvent(task: ITask): EventInput | null {
	const hasStart = task.startDate && new Date(task.startDate).getTime() > 0
	const hasEnd = task.endDate && new Date(task.endDate).getTime() > 0
	const hasDue = task.dueDate && new Date(task.dueDate).getTime() > 0

	let start: Date | null = null
	let end: Date | null = null
	let allDay = false

	if (hasStart && hasEnd) {
		start = new Date(task.startDate as Date)
		end = new Date(task.endDate as Date)
	} else if (hasStart) {
		start = new Date(task.startDate as Date)
		end = start
	} else if (hasDue) {
		start = new Date(task.dueDate as Date)
		end = start
	} else {
		return null
	}

	// Check if this is an all-day event (time is midnight)
	if (
		start.getHours() === 0 &&
		start.getMinutes() === 0 &&
		end.getHours() === 0 &&
		end.getMinutes() === 0
	) {
		allDay = true
	}

	const color = task.hexColor
		? `#${task.hexColor.replace('#', '')}`
		: task.done
			? '#888888'
			: undefined

	return {
		id: String(task.id),
		title: task.title,
		start: start.toISOString(),
		end: end.toISOString(),
		allDay,
		backgroundColor: color,
		borderColor: color,
		textColor: color ? '#ffffff' : undefined,
		extendedProps: {
			task,
			done: task.done,
		},
		classNames: task.done ? ['fc-event-done'] : [],
	}
}

// Load tasks for visible calendar range
async function loadTasks(start: Date, end: Date) {
	loading.value = true
	calendarEvents.value = []

	const taskService = new TaskService()

	try {
		// Build a filter that catches tasks visible in the date range:
		// tasks with dueDate OR startDate OR endDate in [start, end]
		const startIso = start.toISOString()
		const endIso = end.toISOString()

		// Fetch tasks with dueDate in range
		const [dueTasks, startTasks] = await Promise.all([
			taskService.getAll({}, {
				filter: `due_date >= '${startIso}' && due_date <= '${endIso}'`,
				filter_include_nulls: false,
				per_page: 500,
			}),
			taskService.getAll({}, {
				filter: `start_date >= '${startIso}' && start_date <= '${endIso}'`,
				filter_include_nulls: false,
				per_page: 500,
			}),
		])

		// Merge and deduplicate by task id
		const taskMap = new Map<number, ITask>()
		for (const task of [...dueTasks, ...startTasks]) {
			if (!taskMap.has(task.id)) {
				taskMap.set(task.id, task)
			}
		}

		const events: EventInput[] = []
		for (const task of taskMap.values()) {
			const event = taskToEvent(task)
			if (event) {
				events.push(event)
			}
		}

		calendarEvents.value = events
	} catch (e) {
		console.error('Failed to load calendar tasks', e)
	} finally {
		loading.value = false
	}

	// Update the calendar API events
	const calApi = calendarRef.value?.getApi()
	if (calApi) {
		calApi.removeAllEvents()
		calApi.addEventSource(calendarEvents.value)
	}
}

// Handle date range change (navigating the calendar)
function handleDatesSet(info: {start: Date, end: Date}) {
	currentDateRange.value = {start: info.start, end: info.end}
	loadTasks(info.start, info.end)
}

// Handle clicking on empty date slot → open create modal
function handleDateSelect(selectInfo: DateSelectArg) {
	if (projects.value.length === 0) return

	newTask.value = {
		title: '',
		projectId: projects.value[0]?.id ?? null,
		dueDateStr: toDatetimeLocal(selectInfo.start),
		startDateStr: toDatetimeLocal(selectInfo.start),
		endDateStr: toDatetimeLocal(selectInfo.end),
	}
	showCreateModal.value = true
}

// Handle clicking on existing event → navigate to task detail
function handleEventClick(clickInfo: EventClickArg) {
	const task = clickInfo.event.extendedProps.task as ITask
	router.push({name: 'task.detail', params: {id: task.id}})
}

// Create a new task
async function createTask() {
	if (!newTask.value.title || !newTask.value.projectId) return

	creating.value = true
	const taskService = new TaskService()

	try {
		const task = new TaskModel({
			title: newTask.value.title,
			projectId: newTask.value.projectId,
			dueDate: newTask.value.dueDateStr ? new Date(newTask.value.dueDateStr) : null,
			startDate: newTask.value.startDateStr ? new Date(newTask.value.startDateStr) : null,
			endDate: newTask.value.endDateStr ? new Date(newTask.value.endDateStr) : null,
		})

		const created = await taskService.create(task)
		closeCreateModal()

		// Add the new task to the calendar
		const event = taskToEvent(created)
		if (event && calendarRef.value) {
			calendarRef.value.getApi().addEvent(event)
		}
	} catch (e) {
		console.error('Failed to create task', e)
	} finally {
		creating.value = false
	}
}

function closeCreateModal() {
	showCreateModal.value = false
}

// FullCalendar options
const calendarOptions = computed<CalendarOptions>(() => ({
	plugins: [dayGridPlugin, timeGridPlugin, interactionPlugin, listPlugin],
	initialView: 'dayGridMonth',
	headerToolbar: {
		left: 'prev,next today',
		center: 'title',
		right: 'dayGridMonth,timeGridWeek,timeGridDay,listWeek',
	},
	locale: ruLocale,
	firstDay: 1,
	selectable: true,
	selectMirror: true,
	editable: false,
	nowIndicator: true,
	dayMaxEvents: true,
	events: calendarEvents.value,
	select: handleDateSelect,
	eventClick: handleEventClick,
	datesSet: handleDatesSet,
	height: 'auto',
	eventDidMount(info) {
		// Add tooltip with task title
		info.el.title = info.event.title
	},
}))

onMounted(async () => {
	// Ensure projects are loaded
	if (projectStore.projectsArray.length === 0) {
		await projectStore.loadAllProjects()
	}
})
</script>

<style scoped>
.calendar-view-container {
	padding: 1.5rem;
	max-width: 1400px;
	margin: 0 auto;
}

.calendar-wrapper {
	background: var(--white);
	border-radius: 0.5rem;
	padding: 1rem;
	box-shadow: var(--shadow);
}

/* FullCalendar customization */
:deep(.fc-event-done) {
	opacity: 0.6;
	text-decoration: line-through;
}

:deep(.fc-button) {
	background-color: var(--primary) !important;
	border-color: var(--primary) !important;
}

:deep(.fc-button:hover) {
	background-color: var(--primary-dark, #3273dc) !important;
	border-color: var(--primary-dark, #3273dc) !important;
}

:deep(.fc-button-active),
:deep(.fc-button-primary:not(:disabled).fc-button-active) {
	background-color: var(--primary-dark, #3273dc) !important;
	border-color: var(--primary-dark, #3273dc) !important;
}

:deep(.fc-daygrid-event) {
	cursor: pointer;
}

:deep(.fc-highlight) {
	background: color-mix(in srgb, var(--primary) 20%, transparent) !important;
}
</style>
