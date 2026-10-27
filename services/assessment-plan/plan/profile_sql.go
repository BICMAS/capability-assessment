package plan

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

func (s *SQLStore) SaveProfileBaseline(ctx context.Context, baseline ProfileBaseline) (ProfileBaseline, error) {
	if !present(baseline.OrganisationID) || !present(baseline.PlanID) || !present(baseline.ProfileVersionID) || !present(baseline.PersonID) {
		return ProfileBaseline{}, ErrInvalid
	}
	existing, err := s.ProfileBaseline(ctx, baseline.OrganisationID, baseline.PlanID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return ProfileBaseline{}, err
	}
	id, err := newID()
	if err != nil {
		return ProfileBaseline{}, err
	}
	payload, err := json.Marshal(copySkillSnapshots(baseline.Skills))
	if err != nil {
		return ProfileBaseline{}, err
	}
	_, err = s.database.ExecContext(ctx, `
		INSERT INTO profile_plan_baselines (
			id, organisation_id, plan_id, assignment_id, person_id, profile_id,
			profile_version_id, version_number, role_id, role_name, skills, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now()
		)`,
		id, baseline.OrganisationID, baseline.PlanID, baseline.AssignmentID, baseline.PersonID, baseline.ProfileID,
		baseline.ProfileVersionID, baseline.VersionNumber, baseline.RoleID, baseline.RoleName, string(payload),
	)
	if err != nil {
		return ProfileBaseline{}, err
	}
	return s.ProfileBaseline(ctx, baseline.OrganisationID, baseline.PlanID)
}

func (s *SQLStore) ProfileBaseline(ctx context.Context, organisationID, planID string) (ProfileBaseline, error) {
	row := s.database.QueryRowContext(ctx, `
		SELECT id, organisation_id, plan_id, assignment_id, person_id, profile_id,
			profile_version_id, version_number, role_id, role_name, skills
		FROM profile_plan_baselines
		WHERE organisation_id = $1 AND plan_id = $2`, organisationID, planID)
	return scanBaseline(row)
}

func (s *SQLStore) LatestPlan(ctx context.Context, organisationID, assignmentID string) (Plan, error) {
	var id string
	err := s.database.QueryRowContext(ctx, `
		SELECT id FROM plans
		WHERE organisation_id = $1 AND assignment_id = $2
		ORDER BY version_number DESC
		LIMIT 1`, organisationID, assignmentID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return Plan{}, ErrNotFound
	}
	if err != nil {
		return Plan{}, err
	}
	return s.GetPlan(ctx, organisationID, id)
}

func (s *SQLStore) ListProfileBaselines(ctx context.Context, organisationID string) ([]ProfileBaseline, error) {
	if !present(organisationID) {
		return nil, ErrInvalid
	}
	rows, err := s.database.QueryContext(ctx, `
		SELECT id, organisation_id, plan_id, assignment_id, person_id, profile_id,
			profile_version_id, version_number, role_id, role_name, skills
		FROM profile_plan_baselines
		WHERE organisation_id = $1`, organisationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	baselines := []ProfileBaseline{}
	for rows.Next() {
		baseline, err := scanBaseline(rows)
		if err != nil {
			return nil, err
		}
		baselines = append(baselines, baseline)
	}
	return baselines, rows.Err()
}

type baselineScanner interface {
	Scan(dest ...any) error
}

func scanBaseline(row baselineScanner) (ProfileBaseline, error) {
	var baseline ProfileBaseline
	var payload string
	err := row.Scan(
		&baseline.ID, &baseline.OrganisationID, &baseline.PlanID, &baseline.AssignmentID, &baseline.PersonID,
		&baseline.ProfileID, &baseline.ProfileVersionID, &baseline.VersionNumber, &baseline.RoleID, &baseline.RoleName, &payload,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ProfileBaseline{}, ErrNotFound
	}
	if err != nil {
		return ProfileBaseline{}, err
	}
	if err := json.Unmarshal([]byte(payload), &baseline.Skills); err != nil {
		return ProfileBaseline{}, err
	}
	return baseline, nil
}
