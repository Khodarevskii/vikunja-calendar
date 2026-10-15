# Vikunja Calendar

Доработанная версия [Vikunja](https://github.com/go-vikunja/vikunja) — open-source менеджера задач (Go + Vue 3) — под процессы рабочей команды: календарь задач, декомпозиция с весами подзадач, автоматический расчёт прогресса и «частота контроля» задач.

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vuedotjs&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-3178C6?logo=typescript&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=white)
![License: AGPL-3.0](https://img.shields.io/badge/license-AGPL--3.0-blue)

## Что добавлено поверх Vikunja

### Календарь задач
Новый раздел `/tasks/by/calendar` на базе FullCalendar: виды «Месяц», «Неделя», «День» и «Повестка дня», русская локализация.
- создание задачи выделением диапазона дат прямо в календаре;
- подзадачи наследуют даты родителя и отображаются внутри родительской задачи вместе с аватарами исполнителей;
- исправлено наложение событий, время события видно всегда, длинные названия обрезаются.

### Декомпозиция задач
Компонент `TaskDecomposition` в карточке задачи: задачу можно разбить на подзадачи построчно, указать вес каждой и назначить несколько исполнителей. Вес синхронизируется в обе стороны между строкой декомпозиции и подзадачей.

### Автоматический прогресс задач (backend)
Логика в `pkg/models/task_progress.go`:
- `percent_done` родителя считается как взвешенная сумма прогресса подзадач и пунктов чек-листа (в том числе чек-листа из описания задачи);
- изменения каскадно поднимаются вверх по всей иерархии задач;
- когда выполнены все подзадачи и пункты чек-листа, задача закрывается автоматически и переносится в колонку «Готово» на канбан-доске;
- снятие отметки с родителя каскадно снимает её с дочерних задач (с защитой от циклов);
- покрыто unit- и интеграционными тестами (`task_progress_test.go`, `task_progress_integration_test.go`).

### Частота контроля
Новое поле задачи `control_frequency`: ежедневно / еженедельно / раз в две недели / по завершении / «просто сделай».
- цветная метка на карточках канбана и в карточке задачи (светлая и тёмная темы);
- колонка «Контроль» в табличном виде с сортировкой по весу частоты;
- фильтрация задач по `controlFrequency` в строке фильтра;
- уведомление исполнителям при изменении частоты.

### Инфраструктура
- миграции БД для новых полей (`checklist_items` → `jsonb` на PostgreSQL, `subtask_weight`, `control_frequency`);
- команда `vikunja seed` — наполняет базу тестовыми пользователями, проектами, иерархиями задач и правами доступа;
- `Dockerfile.local` — многоэтапная сборка фронтенда и бэкенда в один образ на Alpine, PostgreSQL по умолчанию;
- перевод новых интерфейсов на русский язык.

## Стек

**Backend:** Go, Echo, xorm, PostgreSQL, Mage
**Frontend:** Vue 3, TypeScript, Pinia, Vite, FullCalendar, Bulma
**Инфраструктура:** Docker (multi-stage build)

## Запуск

Сборка образа:

```bash
docker build -f Dockerfile.local -t vikunja-calendar .
```

Запуск вместе с PostgreSQL:

```bash
docker network create vikunja

docker run -d --name vikunja-db --network vikunja \
  -e POSTGRES_USER=vikunja -e POSTGRES_PASSWORD=secret -e POSTGRES_DB=vikunja \
  postgres:16-alpine

docker run -d --name vikunja --network vikunja -p 3456:3456 \
  -e VIKUNJA_DATABASE_HOST=vikunja-db \
  -e VIKUNJA_DATABASE_USER=vikunja \
  -e VIKUNJA_DATABASE_PASSWORD=secret \
  -e VIKUNJA_DATABASE_DATABASE=vikunja \
  -e VIKUNJA_SERVICE_PUBLICURL=http://localhost:3456 \
  vikunja-calendar
```

Приложение будет доступно на http://localhost:3456.

Тестовые данные:

```bash
docker exec vikunja /app/vikunja/vikunja seed
```

## Разработка

```bash
# backend
go install github.com/magefile/mage@latest
mage build
go test ./pkg/models/ -run Progress

# frontend
cd frontend
pnpm install
pnpm dev
```

## Лицензия

Проект основан на [Vikunja](https://vikunja.io) и распространяется на тех же условиях — [AGPL-3.0](LICENSE).
