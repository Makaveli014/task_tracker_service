package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"TestTask_Bazis/internal/models"
)

type AnalyticsRepository struct{ db *sql.DB }

func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository { return &AnalyticsRepository{db: db} }

func (r *AnalyticsRepository) TeamStats(ctx context.Context, teamID uint64) (*models.TeamStats, error) {
	const query = `WITH status_counts AS (
		SELECT status, COUNT(*) count FROM tasks WHERE team_id = ? GROUP BY status
	), top_assignees AS (
		SELECT tk.assignee_id user_id, u.name, COUNT(*) closed_tasks
		FROM tasks tk JOIN users u ON u.id = tk.assignee_id
		WHERE tk.team_id = ? AND tk.status = 'done' AND tk.closed_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
		GROUP BY tk.assignee_id, u.name ORDER BY closed_tasks DESC, tk.assignee_id LIMIT 3
	), summary AS (
		SELECT AVG(TIMESTAMPDIFF(SECOND, created_at, closed_at)) average_seconds
		FROM tasks WHERE team_id = ? AND closed_at IS NOT NULL
	), comments AS (
		SELECT COUNT(tc.id) comment_count FROM tasks tk LEFT JOIN task_comments tc ON tc.task_id = tk.id WHERE tk.team_id = ?
	)
	SELECT
		COALESCE((SELECT JSON_OBJECTAGG(status, count) FROM status_counts), JSON_OBJECT()),
		COALESCE((SELECT JSON_ARRAYAGG(JSON_OBJECT('user_id', user_id, 'name', name, 'closed_tasks', closed_tasks)) FROM top_assignees), JSON_ARRAY()),
		(SELECT average_seconds FROM summary), (SELECT comment_count FROM comments)`
	var statusesJSON, assigneesJSON []byte
	var average sql.NullFloat64
	stats := &models.TeamStats{TeamID: teamID}
	if err := r.db.QueryRowContext(ctx, query, teamID, teamID, teamID, teamID).Scan(&statusesJSON, &assigneesJSON, &average, &stats.CommentCount); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(statusesJSON, &stats.TasksByStatus); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(assigneesJSON, &stats.TopAssignees); err != nil {
		return nil, err
	}
	if average.Valid {
		stats.AverageClosingTimeSeconds = &average.Float64
	}
	return stats, nil
}
