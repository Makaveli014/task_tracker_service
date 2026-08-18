package repository

import (
	"context"
	"database/sql"

	"TestTask_Bazis/internal/models"
)

type TeamRepository struct {
	db *sql.DB
}

func NewTeamRepository(db *sql.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) Create(ctx context.Context, name, description string, createdBy uint64) (*models.Team, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		"INSERT INTO teams (name, description, created_by) VALUES (?, ?, ?)",
		name, description, createdBy,
	)
	if err != nil {
		return nil, err
	}
	teamID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	if _, err = tx.ExecContext(ctx,
		"INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, 'owner')",
		teamID, createdBy,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, uint64(teamID))
}

func (r *TeamRepository) GetByID(ctx context.Context, id uint64) (*models.Team, error) {
	t := &models.Team{}
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, description, created_by, created_at FROM teams WHERE id = ?", id,
	).Scan(&t.ID, &t.Name, &t.Description, &t.CreatedBy, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (r *TeamRepository) GetByUserID(ctx context.Context, userID uint64) ([]models.Team, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.description, t.created_by, t.created_at
		FROM teams t
		JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = ?
		ORDER BY t.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []models.Team
	for rows.Next() {
		var t models.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.CreatedBy, &t.CreatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

func (r *TeamRepository) AddMember(ctx context.Context, teamID, userID uint64, role string) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)",
		teamID, userID, role,
	)
	return err
}

func (r *TeamRepository) UpdateMemberRole(ctx context.Context, teamID, userID uint64, role string) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE team_members SET role = ? WHERE team_id = ? AND user_id = ? AND role <> 'owner'",
		role, teamID, userID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *TeamRepository) GetMemberRole(ctx context.Context, teamID, userID uint64) (string, error) {
	var role string
	err := r.db.QueryRowContext(ctx,
		"SELECT role FROM team_members WHERE team_id = ? AND user_id = ?", teamID, userID,
	).Scan(&role)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return role, err
}

func (r *TeamRepository) IsMember(ctx context.Context, teamID, userID uint64) (bool, error) {
	role, err := r.GetMemberRole(ctx, teamID, userID)
	return role != "", err
}
