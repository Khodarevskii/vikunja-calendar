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
					<div
						v-if="newTask.projectId"
						class="field"
					>
						<label class="label">{{ $t('task.attributes.assignees') }}</label>
						<Multiselect
							v-model="selectedAssignees"
							:loading="assigneeSearchLoading"
							:placeholder="$t('task.assignee.placeholder')"
							:multiple="true"
							:search-results="foundUsers"
							label="name"
							:select-placeholder="$t('task.assignee.selectPlaceholder')"
							:autocomplete-enabled="false"
							@search="findUser"
							@select="selectAssignee"
						>
							<template #items="{items}">
								<AssigneeList
									:assignees="items"
									can-remove
									@remove="removeAssignee"
								/>
							</template>
							<template #searchResult="{option: user}">
								<User
									:avatar-size="24"
									:show-username="true"
									:user="user"
								/>
							</template>
						</Multiselect>
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
import {ref, computed, watch, onMounted} from 'vue'
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
import type {IUser} from '@/modelTypes/IUser'
import {useProjectStore} from '@/stores/projects'
import {useAuthStore} from '@/stores/auth'
import XButton from '@/components/input/Button.vue'
import Multiselect from '@/components/input/Multiselect.vue'
import User from '@/components/misc/User.vue'
import AssigneeList from '@/components/tasks/partials/AssigneeList.vue'
import ProjectUserService from '@/services/projectUsers'
import {includesById} from '@/helpers/utils'
import {getDisplayName} from '@/models/user'

const {t} = useI18n()
const router = useRouter()
const projectStore = useProjectStore()
const authStore = useAuthStore()

setTitle(t('navigation.calendar'))

// Calendar ref
const calendarRef = ref<InstanceType<typeof FullCalendar> | null>(null)

// Loading state
const loading = ref(false)

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

// Assignee selection state
const selectedAssignees = ref<IUser[]>([])
const foundUsers = ref<IUser[]>([])
const assigneeSearchLoading = ref(false)
const projectUserService = new ProjectUserService()

// Clear selected assignees when project changes
watch(() => newTask.value.projectId, () => {
	selectedAssignees.value = []
	foundUsers.value = []
})

async function findUser(query: string) {
	if (!newTask.value.projectId) return

	assigneeSearchLoading.value = true
	try {
		const response = await projectUserService.getAll(
			{projectId: newTask.value.projectId},
			{s: query},
		) as IUser[]

		foundUsers.value = response
			.filter(({id}) => !includesById(selectedAssignees.value, id))
			.map(u => {
				u.name = getDisplayName(u)
				return u
			})
	} finally {
		assigneeSearchLoading.value = false
	}
}

function selectAssignee(user: IUser) {
	selectedAssignees.value.push(user)
}

function removeAssignee(user: IUser) {
	selectedAssignees.value = selectedAssignees.value.filter(a => a.id !== user.id)
}

// Current visible date range
const currentDateRange = ref<{start: Date, end: Date} | null>(null)

// Generation counter for cancelling stale loadTasks responses
let loadGeneration = 0

// Convert date to datetime-local string (local time)
function toDatetimeLocal(date: Date): string {
	const pad = (n: number) => String(n).padStart(2, '0')
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

// Format date as RFC 3339 without milliseconds (Go's time.RFC3339 format)
function toRFC3339(date: Date): string {
	return date.toISOString().replace(/\.\d{3}Z$/, 'Z')
}

// Returns true if a Date falls exactly at midnight UTC (= Vikunja all-day marker)
function isMidnightUTC(d: Date): boolean {
	return d.getUTCHours() === 0 && d.getUTCMinutes() === 0 && d.getUTCSeconds() === 0
}

// Convert task to FullCalendar event
function taskToEvent(task: ITask): EventInput | null {
	const hasStart = task.startDate && new Date(task.startDate).getTime() > 0
	const hasEnd = task.endDate && new Date(task.endDate).getTime() > 0
	const hasDue = task.dueDate && new Date(task.dueDate).getTime() > 0

	if (!hasStart && !hasDue) {
		return null
	}

	let start: Date
	let end: Date
	let allDay = false

	if (hasStart) {
		start = new Date(task.startDate as Date)
		if (hasEnd) {
			end = new Date(task.endDate as Date)
			// Both at midnight UTC → all-day range event
			if (isMidnightUTC(start) && isMidnightUTC(end)) {
				allDay = true
			}
		} else {
			// No end date: all-day if midnight UTC, otherwise 1-hour duration
			if (isMidnightUTC(start)) {
				allDay = true
				end = new Date(start)
				end.setUTCDate(end.getUTCDate() + 1)
			} else {
				end = new Date(start.getTime() + 60 * 60 * 1000)
			}
		}
	} else {
		// Only dueDate
		start = new Date(task.dueDate as Date)
		if (isMidnightUTC(start)) {
			allDay = true
			end = new Date(start)
			end.setUTCDate(end.getUTCDate() + 1)
		} else {
			end = new Date(start.getTime() + 60 * 60 * 1000)
		}
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

// Fetch all pages for a given filter query
async function fetchAllPages(params: Record<string, unknown>): Promise<ITask[]> {
	const service = new TaskService()
	const results: ITask[] = []
	let page = 1
	do {
		const tasks = await service.getAll({} as ITask, {
			...params,
			filter_timezone: authStore.settings.timezone,
			per_page: 500,
		}, page)
		results.push(...(tasks as ITask[]))
		page++
	} while (page <= service.totalPages)
	return results
}

// Load tasks for visible calendar range
async function loadTasks(start: Date, end: Date) {
	const generation = ++loadGeneration
	loading.value = true

	try {
		const startIso = toRFC3339(start)
		const endIso = toRFC3339(end)

		// Three parallel queries:
		// 1. Tasks whose due date falls in the visible range
		// 2. Tasks whose start date falls in the visible range
		// 3. Tasks that span the visible range (started before, end after range start)
		const [dueTasks, startTasks, spanTasks] = await Promise.all([
			fetchAllPages({
				filter: `due_date >= '${startIso}' && due_date <= '${endIso}'`,
				filter_include_nulls: false,
			}),
			fetchAllPages({
				filter: `start_date >= '${startIso}' && start_date <= '${endIso}'`,
				filter_include_nulls: false,
			}),
			fetchAllPages({
				filter: `start_date <= '${startIso}' && end_date >= '${startIso}'`,
				filter_include_nulls: false,
			}),
		])

		// Discard stale results if a newer loadTasks was triggered
		if (generation !== loadGeneration) {
			return
		}

		// Merge and deduplicate by task id
		const taskMap = new Map<number, ITask>()
		for (const task of [...dueTasks, ...startTasks, ...spanTasks]) {
			if (!taskMap.has(task.id)) {
				taskMap.set(task.id, task)
			}
		}

		// Render events in the calendar
		const calApi = calendarRef.value?.getApi()
		if (calApi) {
			calApi.removeAllEvents()
			for (const task of taskMap.values()) {
				const event = taskToEvent(task)
				if (event) {
					calApi.addEvent(event)
				}
			}
		}
	} catch (e) {
		console.error('Failed to load calendar tasks', e)
	} finally {
		loading.value = false
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
	selectedAssignees.value = []
	foundUsers.value = []
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
			dueDate: newTask.value.dueDateStr ? new Date(newTask.value.dueDateStr).toISOString() : null,
			startDate: newTask.value.startDateStr ? new Date(newTask.value.startDateStr).toISOString() : null,
			endDate: newTask.value.endDateStr ? new Date(newTask.value.endDateStr).toISOString() : null,
			assignees: selectedAssignees.value,
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

:deep(.fc-scrollgrid-sync-inner){
	background: var(--white);
	
}
::v-deep(.fc-theme-standard td, .fc-theme-standard th){
	border:1px solid var(--border) !important;
}

::v-deep(.fc-col-header-cell.fc-day){
	border:1px solid var(--border) !important;
}

:deep(.fc-theme-standard .fc-scrollgrid){
	border:1px solid var(--border);
}
:deep(.fc .fc-daygrid-day-number){
	color: var(--border)
}
:deep(.fc .fc-col-header-cell-cushion){
	color: var(--border)
}
:deep(.fc .fc-scrollgrid-section-sticky > *){
	background: none;
	border-right:1px solid var(--border) !important;
}
:deep(.modal-card-head){
	background-color: var(--modal-color) !important;
	border-color: var(--border) !important;
}
:deep(.modal-card-foot){
	background-color: var(--modal-color) !important;
	border-color: var(--border) !important;
}
:deep(.input::-webkit-calendar-picker-indicator){
	filter: invert(var(--filter));
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
