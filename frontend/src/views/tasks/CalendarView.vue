<template>
	<div class="calendar-view-container">
		<h1 class="title">
			{{ $t('navigation.calendar') }}
		</h1>

		<!-- Выбор проекта -->
		<div class="project-selector">
			<div class="field has-addons">
				<div
					class="project-tab"
					:class="{ 'is-active': selectedProjectId === null }"
					@click="selectProject(null)"
				>
					{{ $t('navigation.allProjects') }}
				</div>
				<div
					v-for="project in allProjects"
					:key="project.id"
					class="project-tab"
					:class="{ 'is-active': selectedProjectId === project.id }"
					@click="selectProject(project.id)"
				>
					{{ project.title }}
				</div>
			</div>
		</div>

		<div class="calendar-wrapper">
			<FullCalendar
				ref="calendarRef"
				:options="calendarOptions"
			/>
		</div>

		<!-- Модальное окно: создание задачи -->
		<div
			v-if="showCreateModal"
			class="modal is-active"
			@click.self="closeCreateModal"
		>
			<div class="modal-background" @click="closeCreateModal"></div>
			<div class="modal-card">
				<header class="modal-card-head">
					<p class="modal-card-title">
						{{ $t('task.new') }}
					</p>
					<button
						class="delete"
						aria-label="close"
						@click="closeCreateModal"
					></button>
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
						<label class="label">{{ $t('task.attributes.description') }}</label>
						<div class="control">
							<textarea
								v-model="newTask.description"
								class="textarea"
								:placeholder="$t('task.attributes.description')"
								rows="3"
							></textarea>
						</div>
					</div>
					<div class="field">
						<label class="label">{{ $t('task.attributes.project') }}</label>
						<div class="control">
							<div class="select is-fullwidth">
								<select v-model="newTask.projectId">
									<option
										v-for="project in modalProjects"
										:key="project.id"
										:value="project.id"
									>
										{{ project.title }}
									</option>
								</select>
							</div>
						</div>
					</div>
					<div
						class="field"
						@click.capture="closeDatepickersExcept('dueDate')"
					>
						<label class="label">{{ $t('task.attributes.dueDate') }}</label>
						<div class="control">
							<Datepicker
								ref="dueDatePicker"
								v-model="newTask.dueDate"
								:choose-date-label="$t('task.detail.chooseDueDate')"
							/>
						</div>
					</div>
					<div class="columns">
						<div class="column">
							<div
								class="field"
								@click.capture="closeDatepickersExcept('startDate')"
							>
								<label class="label">{{ $t('task.attributes.startDate') }}</label>
								<div class="control">
									<Datepicker
										ref="startDatePicker"
										v-model="newTask.startDate"
										:choose-date-label="$t('task.detail.chooseStartDate')"
									/>
								</div>
							</div>
						</div>
						<div class="column">
							<div
								class="field"
								@click.capture="closeDatepickersExcept('endDate')"
							>
								<label class="label">{{ $t('task.attributes.endDate') }}</label>
								<div class="control">
									<Datepicker
										ref="endDatePicker"
										v-model="newTask.endDate"
										:choose-date-label="$t('task.detail.chooseEndDate')"
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

import {PERMISSIONS} from '@/constants/permissions'
import TaskService from '@/services/task'
import TaskModel from '@/models/task'
import ProjectService from '@/services/project'
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
import {getDisplayName, fetchAvatarBlobUrl} from '@/models/user'
import Datepicker from '@/components/input/Datepicker.vue'

const {t} = useI18n()
const router = useRouter()
const projectStore = useProjectStore()
const authStore = useAuthStore()

setTitle(t('navigation.calendar'))

// Ссылка на компонент FullCalendar
const calendarRef = ref<InstanceType<typeof FullCalendar> | null>(null)

// Состояние загрузки
const loading = ref(false)

// Все неархивные проекты
const allProjects = computed(() =>
	projectStore.projectsArray.filter(p => !p.isArchived && p.id > 0),
)

// Выбранный проект: null — «Все проекты», число — конкретный проект
const selectedProjectId = ref<number | null>(null)

// Карта: id проекта → максимальное разрешение (загружается при монтировании)
const projectPermissions = ref<Map<number, number>>(new Map())

// Проекты, в которых у пользователя есть права на запись
const writableProjects = computed(() =>
	allProjects.value.filter(p => {
		const perm = projectPermissions.value.get(p.id)
		return perm !== undefined && perm > PERMISSIONS.READ
	}),
)

// Проекты, отображаемые в выпадающем списке модального окна:
// — вкладка «Все проекты» → все проекты с правами на запись
// — вкладка конкретного проекта → только этот проект (если есть права на запись)
const modalProjects = computed(() => {
	if (selectedProjectId.value === null) {
		return writableProjects.value
	}
	return writableProjects.value.filter(p => p.id === selectedProjectId.value)
})

// Может ли текущий пользователь создавать задачи в выбранном контексте
const canCreateTasks = computed(() => {
	if (authStore.isLinkShareAuth) {
		return false
	}

	if (selectedProjectId.value === null) {
		// «Все проекты» — разрешаем только если ВСЕ проекты доступны для записи
		if (allProjects.value.length === 0) {
			return false
		}
		return allProjects.value.every(p => {
			const perm = projectPermissions.value.get(p.id)
			return perm !== undefined && perm > PERMISSIONS.READ
		})
	}

	// Конкретный проект — разрешаем только если у этого проекта есть права на запись
	const perm = projectPermissions.value.get(selectedProjectId.value)
	return perm !== undefined && perm > PERMISSIONS.READ
})

function selectProject(id: number | null) {
	selectedProjectId.value = id
	// Обновляем календарь для перезагрузки задач с новым фильтром
	if (calendarRef.value) {
		calendarRef.value.getApi().refetchEvents()
	}
}

// Состояние модального окна создания задачи
const showCreateModal = ref(false)
const creating = ref(false)

// Ссылки на компоненты Datepicker для координации открытия/закрытия
const dueDatePicker = ref<InstanceType<typeof Datepicker> | null>(null)
const startDatePicker = ref<InstanceType<typeof Datepicker> | null>(null)
const endDatePicker = ref<InstanceType<typeof Datepicker> | null>(null)

const datepickerRefs = {
	dueDate: dueDatePicker,
	startDate: startDatePicker,
	endDate: endDatePicker,
} as const

function closeDatepickersExcept(except: string) {
	for (const [key, pickerRef] of Object.entries(datepickerRefs)) {
		if (key !== except && pickerRef.value?.show) {
			pickerRef.value.show = false
		}
	}
}

interface NewTaskForm {
	title: string
	description: string
	projectId: number | null
	dueDate: Date | null
	startDate: Date | null
	endDate: Date | null
}

const newTask = ref<NewTaskForm>({
	title: '',
	description: '',
	projectId: null,
	dueDate: null,
	startDate: null,
	endDate: null,
})

// Состояние выбора исполнителей
const selectedAssignees = ref<IUser[]>([])
const foundUsers = ref<IUser[]>([])
const assigneeSearchLoading = ref(false)
const projectUserService = new ProjectUserService()

// Очищаем выбранных исполнителей при смене проекта
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


// Форматирование даты в RFC 3339 без миллисекунд (формат Go time.RFC3339)
function toRFC3339(date: Date): string {
	return date.toISOString().replace(/\.\d{3}Z$/, 'Z')
}

// Проверяет, попадает ли дата ровно на полночь UTC (маркер «весь день» в Vikunja)
function isMidnightUTC(d: Date): boolean {
	return d.getUTCHours() === 0 && d.getUTCMinutes() === 0 && d.getUTCSeconds() === 0
}

// Преобразование задачи в событие FullCalendar
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
			// Обе даты в полночь UTC → событие на весь день
			if (isMidnightUTC(start) && isMidnightUTC(end)) {
				allDay = true
			}
		} else {
			// Нет даты окончания: весь день если полночь UTC, иначе длительность 1 час
			if (isMidnightUTC(start)) {
				allDay = true
				end = new Date(start)
				end.setUTCDate(end.getUTCDate() + 1)
			} else {
				end = new Date(start.getTime() + 60 * 60 * 1000)
			}
		}
	} else {
		// Есть только дата выполнения (dueDate)
		start = new Date(task.dueDate as Date)
		if (isMidnightUTC(start)) {
			allDay = true
			end = new Date(start)
			end.setUTCDate(end.getUTCDate() + 1)
		} else {
			end = new Date(start.getTime() + 60 * 60 * 1000)
		}
	}

	// Определяем цвет события
	let color: string | undefined
	if (task.hexColor) {
		color = `#${task.hexColor.replace('#', '')}`
	} else if (task.done) {
		color = '#888888'
	}

	// Определяем цвет текста
	let textColor: string | undefined
	if (color) {
		textColor = '#ffffff'
	}

	// Определяем CSS-классы
	let classNames: string[] = []
	if (task.done) {
		classNames = ['fc-event-done']
	}

	return {
		id: String(task.id),
		title: task.title,
		start: start.toISOString(),
		end: end.toISOString(),
		allDay,
		backgroundColor: color,
		borderColor: color,
		textColor,
		extendedProps: {
			task,
			done: task.done,
			createdBy: task.createdBy,
			assignees: task.assignees || [],
		},
		classNames,
	}
}

// Загрузка всех страниц для заданного фильтра
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

// Загрузка задач для указанного диапазона дат, возвращает события FullCalendar
async function loadTasksForRange(start: Date, end: Date): Promise<EventInput[]> {
	loading.value = true
	try {
		const startIso = toRFC3339(start)
		const endIso = toRFC3339(end)

		// Фильтр по проекту: если выбран конкретный проект, добавляем условие project_id
		let projectFilter = ''
		if (selectedProjectId.value !== null) {
			projectFilter = ` && project_id = '${selectedProjectId.value}'`
		}

		// Три параллельных запроса:
		// 1. Задачи, у которых дата выполнения попадает в видимый диапазон
		// 2. Задачи, у которых дата начала попадает в видимый диапазон
		// 3. Задачи, которые охватывают видимый диапазон (начались до, заканчиваются после)
		const [dueTasks, startTasks, spanTasks] = await Promise.all([
			fetchAllPages({
				filter: `due_date >= '${startIso}' && due_date <= '${endIso}'${projectFilter}`,
				filter_include_nulls: false,
			}),
			fetchAllPages({
				filter: `start_date >= '${startIso}' && start_date <= '${endIso}'${projectFilter}`,
				filter_include_nulls: false,
			}),
			fetchAllPages({
				filter: `start_date <= '${startIso}' && end_date >= '${startIso}'${projectFilter}`,
				filter_include_nulls: false,
			}),
		])

		// Объединяем и убираем дубликаты по id задачи
		const taskMap = new Map<number, ITask>()
		for (const task of [...dueTasks, ...startTasks, ...spanTasks]) {
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

		return events
	} catch (e) {
		console.error('Не удалось загрузить задачи для календаря', e)
		return []
	} finally {
		loading.value = false
	}
}

// Обработка клика по пустому слоту даты → открыть модалку создания задачи
function handleDateSelect(selectInfo: DateSelectArg) {
	if (!canCreateTasks.value) return

	// Определяем проект по умолчанию для модального окна
	let defaultProjectId: number | null = null
	if (selectedProjectId.value !== null) {
		defaultProjectId = selectedProjectId.value
	} else {
		const firstWritable = writableProjects.value[0]
		if (firstWritable) {
			defaultProjectId = firstWritable.id
		}
	}
	if (!defaultProjectId) return

	// Дата начала = текущее локальное время


	// Дата окончания в 9:00 утра:
	// — выбран один день → следующий день
	// — выбран диапазон → последний выбранный день
	const isSingleDay = (selectInfo.end.getTime() - selectInfo.start.getTime()) <= 24 * 60 * 60 * 1000
	const endDate = new Date(selectInfo.end)
	if (!isSingleDay) {
		endDate.setDate(endDate.getDate() - 1)
	}
	endDate.setHours(9, 0, 0, 0)
	selectInfo.start.setHours(9, 0, 0, 0)
	newTask.value = {
		title: '',
		description: '',
		projectId: defaultProjectId,
		dueDate: null,
		startDate: selectInfo.start,
		endDate,
	}
	selectedAssignees.value = []
	foundUsers.value = []
	showCreateModal.value = true
}

// Обработка клика по существующему событию → переход к деталям задачи
function handleEventClick(clickInfo: EventClickArg) {
	const task = clickInfo.event.extendedProps.task as ITask
	router.push({name: 'task.detail', params: {id: task.id}})
}

// Создание новой задачи
async function createTask() {
	if (!newTask.value.title || !newTask.value.projectId) return

	creating.value = true
	const taskService = new TaskService()

	try {
		// Форматируем даты в ISO-строки
		let dueDateStr: string | null = null
		if (newTask.value.dueDate) {
			dueDateStr = newTask.value.dueDate.toISOString()
		}

		let startDateStr: string | null = null
		if (newTask.value.startDate) {
			startDateStr = newTask.value.startDate.toISOString()
		}

		let endDateStr: string | null = null
		if (newTask.value.endDate) {
			endDateStr = newTask.value.endDate.toISOString()
		}

		const task = new TaskModel({
			title: newTask.value.title,
			description: newTask.value.description,
			projectId: newTask.value.projectId,
			dueDate: dueDateStr,
			startDate: startDateStr,
			endDate: endDateStr,
			assignees: selectedAssignees.value,
		})
		
		await taskService.create(task)
		closeCreateModal()

		// Обновляем календарь, чтобы отобразить новую задачу
		if (calendarRef.value) {
			calendarRef.value.getApi().refetchEvents()
		}
	} catch (e) {
		console.error('Не удалось создать задачу', e)
	} finally {
		creating.value = false
		closeCreateModal()
		if (calendarRef.value) {
			calendarRef.value.getApi().refetchEvents()
		}
	}
}

function closeCreateModal() {
	showCreateModal.value = false
}

// Настройки FullCalendar
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
	height: 'auto',
	events(fetchInfo, successCallback, failureCallback) {
		loadTasksForRange(fetchInfo.start, fetchInfo.end)
			.then(events => successCallback(events))
			.catch(e => failureCallback(e as Error))
	},
	async eventDidMount(info) {
		// Добавляем всплывающую подсказку с названием задачи
		info.el.title = info.event.title

		// Отображаем аватарки всех исполнителей в ряд в правом верхнем углу
		const {assignees} = info.event.extendedProps
		if (!assignees || assignees.length === 0) return

		const inner = info.el.querySelector('.fc-event-main') || info.el.querySelector('.fc-event-title-container') || info.el
		inner.style.position = 'relative'

		const container = document.createElement('div')
		container.className = 'fc-event-avatars'

		for (const user of assignees as IUser[]) {
			const avatarUrl = await fetchAvatarBlobUrl(user, 20)
			if (!avatarUrl) continue

			const img = document.createElement('img')
			img.src = avatarUrl
			img.alt = getDisplayName(user)
			img.title = getDisplayName(user)
			img.className = 'fc-event-avatar'
			container.appendChild(img)
		}

		if (container.children.length > 0) {
			inner.appendChild(container)
		}
	},
}))

onMounted(async () => {
	// Убеждаемся, что проекты загружены
	if (projectStore.projectsArray.length === 0) {
		await projectStore.loadAllProjects()
	}

	// Загружаем максимальные права для каждого проекта (getAll их не возвращает)
	if (!authStore.isLinkShareAuth) {
		const projectService = new ProjectService()
		const results = await Promise.allSettled(
			allProjects.value.map(p => projectService.get({id: p.id})),
		)
		const perms = new Map<number, number>()
		for (const result of results) {
			if (result.status === 'fulfilled' && result.value.maxPermission !== null) {
				perms.set(result.value.id, result.value.maxPermission)
			}
		}
		projectPermissions.value = perms
	}
})
</script>

<style scoped>
.calendar-view-container {
	padding: 1.5rem;
	max-width: 1400px;
	margin: 0 auto;
}

.project-selector {
	margin-bottom: 1rem;
}

.project-selector .field.has-addons {
	display: flex;
	flex-wrap: wrap;
	gap: 0.5rem;
}

.project-tab {
	padding: 0.5rem 1rem;
	border-radius: 4px;
	cursor: pointer;
	background: var(--white);
	border: 1px solid var(--border);
	color: var(--text);
	font-size: 0.9rem;
	transition: all 0.2s ease;
	user-select: none;
}

.project-tab:hover {
	border-color: var(--primary);
	color: var(--primary);
}

.project-tab.is-active {
	background: var(--primary);
	border-color: var(--primary);
	color: #fff;
}

.calendar-wrapper {
	background: var(--white);
	border-radius: 0.5rem;
	padding: 1rem;
	box-shadow: var(--shadow);
}


/* Кастомизация FullCalendar */
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
::v-deep(.calendar-wrapper .fc-day-today .fc-daygrid-day-frame){
	background: var(--fc-today-bg-color);
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

:deep(.fc-theme-standard .fc-list-day-cushion){
	background: var(--white);
	color:var(--input-color)
}
:deep(.modal-card-foot){
	background-color: var(--modal-color) !important;
	border-color: var(--border) !important;
}
:deep(.modal-card){
	    max-height: calc(80vh - var(--modal-card-spacing));
}

:deep(.modal-card-body) {
	position: relative;
}

:deep(.modal-card-body .datepicker-popup) {
	position: absolute;
	top: 0;
	left: 50%;
	transform: translateX(-50%);
	z-index: 100;
}

:deep(.fc .fc-list-event:hover td ){
    background-color: var(--white);
	opacity: 0.6;
}
:deep(.fc-theme-standard td, .fc-theme-standard th){
	border:1px solid var(--border) !important;
}
:deep(.calendar-wrapper th){
	border:1px solid var(--border) !important;
}
:deep(.calendar-wrapper .fc-list-event-graphic){
	padding: 10px;
}

:deep(.calendar-wrapper .fc-theme-standard .fc-list){
	border:1px solid var(--border) !important;
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

:deep(.fc-timegrid-axis){
	border:none
}

:deep(.fc-event-avatars) {
	display: flex;
	flex-direction: row;
	gap: 5px;
	position: absolute;
	right: 2px;
	top: 2px;
	pointer-events: none;
}
:deep(.modal-card-body .datepicker-popup){
	z-index: 1000;
}
:deep(.fc-event-avatar) {
	width: 20px;
	height: 20px;
	border-radius: 50%;
	border: 1.5px solid rgba(255, 255, 255, 0.8);
	pointer-events: none;
	object-fit: cover;
	flex-shrink: 0;
}

:deep(.fc-highlight) {
	background: color-mix(in srgb, var(--primary) 20%, transparent) !important;
}

.textarea {
	background-color: var(--input-background-color, var(--scheme-main));
	color: var(--input-color, var(--text-strong));
	border: 1px solid var(--border);
	border-radius: 4px;
	padding: 0.625rem;
	width: 100%;
	resize: vertical;
	font-family: inherit;
	font-size: 1rem;
	transition: border-color 0.2s ease;
}

.textarea::placeholder {
	color: var(--input-placeholder-color);
}

.textarea:focus {
	border-color: var(--primary);
	outline: none;
	box-shadow: 0 0 0 0.125em rgba(var(--primary-h), var(--primary-s), var(--primary-l), 0.25);
}
</style>
