// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package cmd

import (
	"fmt"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/initialize"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/user"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(seedCmd)
}

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed the database with test users, projects, tasks and permissions.",
	PreRun: func(_ *cobra.Command, _ []string) {
		initialize.FullInit()
	},
	Run: func(_ *cobra.Command, _ []string) {
		s := db.NewSession()
		defer s.Close()

		if err := s.Begin(); err != nil {
			log.Fatalf("Could not start transaction: %s", err)
		}

		now := time.Now()
		password := "kirill123"

		// =====================
		// 1. Create users
		// =====================
		users := []*user.User{
			{
				Username: "admin",
				Email:    "admin@example.com",
				Password: password,
				Name:     "Админ",
			},
			{
				Username: "reader1",
				Email:    "reader1@example.com",
				Password: password,
				Name:     "Читатель 1",
			},
			{
				Username: "reader2",
				Email:    "reader2@example.com",
				Password: password,
				Name:     "Читатель 2",
			},
			{
				Username: "editor",
				Email:    "editor@example.com",
				Password: password,
				Name:     "Редактор",
			},
		}

		userIDs := make(map[string]int64)
		for _, u := range users {
			newUser, err := user.CreateUser(s, u)
			if err != nil {
				_ = s.Rollback()
				log.Fatalf("Error creating user %s: %s", u.Username, err)
			}

			// Create default inbox project for each user
			err = models.CreateNewProjectForUser(s, newUser)
			if err != nil {
				_ = s.Rollback()
				log.Fatalf("Error creating default project for user %s: %s", u.Username, err)
			}

			userIDs[u.Username] = newUser.ID
			fmt.Printf("Created user: %s (ID: %d)\n", u.Username, newUser.ID)
		}

		adminID := userIDs["admin"]
		reader1ID := userIDs["reader1"]
		reader2ID := userIDs["reader2"]
		editorID := userIDs["editor"]

		// =====================
		// 2. Create projects (directly via xorm for simplicity)
		// =====================
		type projectDef struct {
			Title    string
			HexColor string
			Desc     string
		}

		projectDefs := []projectDef{
			{Title: "Маркетинг", HexColor: "4287f5", Desc: "Маркетинговые задачи и кампании"},
			{Title: "Разработка", HexColor: "42f554", Desc: "Задачи по разработке продукта"},
			{Title: "Дизайн", HexColor: "f54242", Desc: "Дизайн и UI/UX задачи"},
		}

		projectIDs := make([]int64, 3)
		for i, pd := range projectDefs {
			p := &models.Project{
				Title:       pd.Title,
				Description: pd.Desc,
				HexColor:    pd.HexColor,
				OwnerID:     adminID,
				Position:    float64((i + 1) * 100),
			}

			_, err := s.Insert(p)
			if err != nil {
				_ = s.Rollback()
				log.Fatalf("Error creating project %s: %s", pd.Title, err)
			}
			projectIDs[i] = p.ID
			fmt.Printf("Created project: %s (ID: %d)\n", pd.Title, p.ID)

			// Create default views for each project (List, Gantt, Table, Kanban)
			views := []*models.ProjectView{
				{
					ProjectID: p.ID,
					Title:     "List",
					ViewKind:  models.ProjectViewKindList,
					Position:  100,
				},
				{
					ProjectID: p.ID,
					Title:     "Gantt",
					ViewKind:  models.ProjectViewKindGantt,
					Position:  200,
				},
				{
					ProjectID: p.ID,
					Title:     "Table",
					ViewKind:  models.ProjectViewKindTable,
					Position:  300,
				},
				{
					ProjectID: p.ID,
					Title:     "Kanban",
					ViewKind:  models.ProjectViewKindKanban,
					Position:  400,
				},
			}
			for _, v := range views {
				_, err := s.Insert(v)
				if err != nil {
					_ = s.Rollback()
					log.Fatalf("Error creating view for project %s: %s", pd.Title, err)
				}
			}
		}

		marketingID := projectIDs[0]
		devID := projectIDs[1]
		designID := projectIDs[2]

		// =====================
		// 3. Set up permissions
		// =====================
		// Permission: 0 = Read, 1 = Write, 2 = Admin

		// reader1 and reader2 get read-only access to "Маркетинг"
		// editor gets write access to "Дизайн"
		permissions := []models.ProjectUser{
			{UserID: reader1ID, ProjectID: marketingID, Permission: models.PermissionRead},
			{UserID: reader2ID, ProjectID: marketingID, Permission: models.PermissionRead},
			{UserID: editorID, ProjectID: designID, Permission: models.PermissionWrite},
		}

		for _, perm := range permissions {
			_, err := s.Insert(&perm)
			if err != nil {
				_ = s.Rollback()
				log.Fatalf("Error creating permission: %s", err)
			}
		}
		fmt.Println("Set up permissions: reader1, reader2 -> Маркетинг (read); editor -> Дизайн (write)")

		// =====================
		// 4. Create tasks with colors
		// =====================
		type taskDef struct {
			Title     string
			Desc      string
			ProjectID int64
			HexColor  string
			Done      bool
			Priority  int64
			DueDate   *time.Time
			StartDate *time.Time
			EndDate   *time.Time
		}

		// Helper to create dates relative to now
		daysFromNow := func(days int) *time.Time {
			t := now.AddDate(0, 0, days)
			return &t
		}

		taskDefs := []taskDef{
			// === Маркетинг (5 tasks) ===
			{
				Title:     "Создать контент-план на март",
				Desc:      "Подготовить подробный контент-план для всех каналов на март",
				ProjectID: marketingID,
				HexColor:  "e74c3c",
				DueDate:   daysFromNow(7),
				Priority:  3,
			},
			{
				Title:     "Запустить рекламную кампанию",
				Desc:      "Настроить и запустить рекламу в Google Ads и VK",
				ProjectID: marketingID,
				HexColor:  "3498db",
				DueDate:   daysFromNow(14),
				StartDate: daysFromNow(5),
				Priority:  2,
			},
			{
				Title:     "Подготовить отчёт по ROI",
				Desc:      "Собрать данные и подготовить отчёт по эффективности маркетинга за Q4",
				ProjectID: marketingID,
				HexColor:  "2ecc71",
				Done:      true,
			},
			{
				Title:     "Обновить социальные сети",
				Desc:      "Обновить шапки профилей, описания и ссылки во всех соцсетях",
				ProjectID: marketingID,
				HexColor:  "f39c12",
				DueDate:   daysFromNow(3),
				Priority:  4,
			},
			{
				Title:     "Провести анализ конкурентов",
				Desc:      "Исследовать маркетинговые стратегии топ-5 конкурентов",
				ProjectID: marketingID,
				HexColor:  "9b59b6",
				DueDate:   daysFromNow(10),
				StartDate: daysFromNow(2),
				EndDate:   daysFromNow(10),
			},

			// === Разработка (6 tasks) ===
			{
				Title:     "Исправить баг в авторизации",
				Desc:      "При входе через OAuth иногда теряется сессия пользователя",
				ProjectID: devID,
				HexColor:  "e74c3c",
				Priority:  5,
				DueDate:   daysFromNow(2),
			},
			{
				Title:     "Добавить API эндпоинт для фильтров",
				Desc:      "Реализовать GET /api/v1/filters с поддержкой пагинации",
				ProjectID: devID,
				HexColor:  "3498db",
				DueDate:   daysFromNow(7),
				StartDate: daysFromNow(1),
			},
			{
				Title:     "Оптимизировать SQL запросы",
				Desc:      "Добавить индексы и оптимизировать N+1 запросы в модуле задач",
				ProjectID: devID,
				HexColor:  "1abc9c",
				Done:      true,
			},
			{
				Title:     "Написать unit-тесты",
				Desc:      "Покрыть тестами модуль уведомлений (>80% coverage)",
				ProjectID: devID,
				HexColor:  "f1c40f",
				DueDate:   daysFromNow(5),
				Priority:  2,
			},
			{
				Title:     "Обновить зависимости",
				Desc:      "Обновить Go модули и npm пакеты до актуальных версий",
				ProjectID: devID,
				HexColor:  "e67e22",
				DueDate:   daysFromNow(12),
			},
			{
				Title:     "Рефакторинг модуля уведомлений",
				Desc:      "Вынести логику отправки в отдельный сервис, добавить очередь",
				ProjectID: devID,
				HexColor:  "9b59b6",
				DueDate:   daysFromNow(20),
				StartDate: daysFromNow(8),
				EndDate:   daysFromNow(20),
				Priority:  1,
			},

			// === Дизайн (5 tasks) ===
			{
				Title:     "Создать макет главной страницы",
				Desc:      "Разработать макет новой главной страницы в Figma",
				ProjectID: designID,
				HexColor:  "e74c3c",
				DueDate:   daysFromNow(5),
				Priority:  4,
			},
			{
				Title:     "Обновить UI Kit",
				Desc:      "Добавить новые компоненты: карточки, модальные окна, тултипы",
				ProjectID: designID,
				HexColor:  "3498db",
				DueDate:   daysFromNow(9),
				StartDate: daysFromNow(1),
				EndDate:   daysFromNow(9),
			},
			{
				Title:     "Подготовить иконки для мобильного",
				Desc:      "Экспортировать все иконки в SVG и PNG для iOS/Android",
				ProjectID: designID,
				HexColor:  "2ecc71",
				Done:      true,
			},
			{
				Title:     "Редизайн формы регистрации",
				Desc:      "Упростить форму регистрации, добавить социальную авторизацию",
				ProjectID: designID,
				HexColor:  "f39c12",
				DueDate:   daysFromNow(8),
				Priority:  3,
			},
			{
				Title:     "Создать анимации переходов",
				Desc:      "Разработать плавные анимации переходов между экранами",
				ProjectID: designID,
				HexColor:  "8e44ad",
				DueDate:   daysFromNow(15),
				StartDate: daysFromNow(5),
				EndDate:   daysFromNow(15),
				Priority:  1,
			},
		}

		taskIndex := make(map[int64]int64) // project_id -> current index

		for _, td := range taskDefs {
			taskIndex[td.ProjectID]++

			t := &models.Task{
				Title:       td.Title,
				Description: td.Desc,
				ProjectID:   td.ProjectID,
				HexColor:    td.HexColor,
				Done:        td.Done,
				Priority:    td.Priority,
				CreatedByID: adminID,
				Index:       taskIndex[td.ProjectID],
			}

			if td.DueDate != nil {
				t.DueDate = *td.DueDate
			}
			if td.StartDate != nil {
				t.StartDate = *td.StartDate
			}
			if td.EndDate != nil {
				t.EndDate = *td.EndDate
			}
			if td.Done {
				t.DoneAt = now
			}

			_, err := s.Insert(t)
			if err != nil {
				_ = s.Rollback()
				log.Fatalf("Error creating task '%s': %s", td.Title, err)
			}
			fmt.Printf("Created task: %s (color: #%s)\n", td.Title, td.HexColor)
		}

		// =====================
		// 5. Create labels with colors
		// =====================
		labels := []*models.Label{
			{Title: "Срочно", HexColor: "e74c3c", CreatedByID: adminID},
			{Title: "В работе", HexColor: "3498db", CreatedByID: adminID},
			{Title: "Готово", HexColor: "2ecc71", CreatedByID: adminID},
			{Title: "На проверке", HexColor: "f39c12", CreatedByID: adminID},
			{Title: "Заблокировано", HexColor: "95a5a6", CreatedByID: adminID},
		}

		for _, l := range labels {
			_, err := s.Insert(l)
			if err != nil {
				_ = s.Rollback()
				log.Fatalf("Error creating label '%s': %s", l.Title, err)
			}
			fmt.Printf("Created label: %s (color: #%s)\n", l.Title, l.HexColor)
		}

		// Commit the transaction
		if err := s.Commit(); err != nil {
			log.Fatalf("Error committing transaction: %s", err)
		}

		fmt.Println("\n=== Seed completed successfully! ===")
		fmt.Println("\nUsers (all with password 'kirill123'):")
		fmt.Printf("  admin  (ID: %d) - Владелец всех проектов (admin)\n", adminID)
		fmt.Printf("  reader1 (ID: %d) - Только чтение в 'Маркетинг'\n", reader1ID)
		fmt.Printf("  reader2 (ID: %d) - Только чтение в 'Маркетинг'\n", reader2ID)
		fmt.Printf("  editor  (ID: %d) - Может добавлять задачи в 'Дизайн'\n", editorID)
		fmt.Println("\nProjects:")
		fmt.Printf("  Маркетинг  (ID: %d) - 5 задач, синий\n", marketingID)
		fmt.Printf("  Разработка (ID: %d) - 6 задач, зелёный\n", devID)
		fmt.Printf("  Дизайн     (ID: %d) - 5 задач, красный\n", designID)
	},
}
