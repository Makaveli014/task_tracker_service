package repository

import (
	"context"
	"database/sql"

	"TestTask_Bazis/internal/models"
)

type AnalyticsRepository struct {
	db *sql.DB
}

func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// TeamStats — для каждой команды: название, кол-во участников, кол-во задач done за 7 дней
func (r *AnalyticsRepository) TeamStats(ctx context.Context) ([]models.TeamStats, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			t.name,
			COUNT(DISTINCT tm.user_id) AS member_count,
			COUNT(DISTINCT CASE
				WHEN tk.status = 'done' AND tk.updated_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
				THEN tk.id
			END) AS done_tasks_week
		FROM teams t
		LEFT JOIN team_members tm ON tm.team_id = t.id
		LEFT JOIN tasks tk ON tk.team_id = t.id
		GROUP BY t.id, t.name
		ORDER BY t.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.TeamStats
	for rows.Next() {
		var s models.TeamStats
		if err := rows.Scan(&s.TeamName, &s.MemberCount, &s.DoneTasksWeek); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// TopCreators — топ-3 пользователя по созданным задачам в каждой команде за месяц (оконная функция)
func (r *AnalyticsRepository) TopCreators(ctx context.Context) ([]models.TopCreator, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH ranked AS (
			SELECT
				tk.team_id,
				t.name AS team_name,
				u.username,
				COUNT(*) AS task_count,
				ROW_NUMBER() OVER (PARTITION BY tk.team_id ORDER BY COUNT(*) DESC) AS rn
			FROM tasks tk
			JOIN teams t ON t.id = tk.team_id
			JOIN users u ON u.id = tk.created_by
			WHERE tk.created_at >= DATE_SUB(NOW(), INTERVAL 1 MONTH)
			GROUP BY tk.team_id, t.name, u.id, u.username
		)
		SELECT team_name, username, task_count
		FROM ranked
		WHERE rn <= 3
		ORDER BY team_name, task_count DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creators []models.TopCreator
	for rows.Next() {
		var c models.TopCreator
		if err := rows.Scan(&c.TeamName, &c.Username, &c.TaskCount); err != nil {
			return nil, err
		}
		creators = append(creators, c)
	}
	return creators, rows.Err()
}

// IntegrityViolations — задачи, где assignee не является членом команды
func (r *AnalyticsRepository) IntegrityViolations(ctx context.Context) ([]models.IntegrityViolation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tk.id, tk.title, tk.assignee_id, tk.team_id, u.username
		FROM tasks tk
		JOIN users u ON u.id = tk.assignee_id
		LEFT JOIN team_members tm ON tm.team_id = tk.team_id AND tm.user_id = tk.assignee_id
		WHERE tk.assignee_id IS NOT NULL AND tm.id IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var violations []models.IntegrityViolation
	for rows.Next() {
		var v models.IntegrityViolation
		if err := rows.Scan(&v.TaskID, &v.Title, &v.AssigneeID, &v.TeamID, &v.AssigneeName); err != nil {
			return nil, err
		}
		violations = append(violations, v)
	}
	return violations, rows.Err()
}
