package plan

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/assessment-plan/openapi"
	"github.com/bicmas-academy/learn/services/token"
)

func AssignPeople(store Store, frameworks FrameworkReader, directory DirectoryReader, publisher Publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		if !claims.HasRole("admin") {
			writeError(w, http.StatusForbidden)
			return
		}
		var body openapi.AssignRequest
		if !decode(w, r, &body) {
			return
		}
		people := make([]AssignPerson, 0, len(body.People))
		for _, person := range body.People {
			reason := ""
			if person.ExceptionReason != nil {
				reason = *person.ExceptionReason
			}
			people = append(people, AssignPerson{PersonID: person.PersonID, ExceptionReason: reason})
		}
		results, err := Assign(r.Context(), store, frameworks, directory, publisher, AssignInput{
			Authorization:      r.Header.Get("Authorization"),
			OrganisationID:     claims.OrganisationID,
			ActorPersonID:      claims.PersonID,
			FrameworkVersionID: body.FrameworkVersionID,
			RoleID:             body.RoleID,
			People:             people,
			AssignedAt:         time.Now().UTC(),
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, batchJSON(results))
	}
}

func ReadAssignment(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		assignment, err := store.GetAssignment(r.Context(), claims.OrganisationID, chi.URLParam(r, "assignmentId"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if !claims.HasRole("admin") {
			writeError(w, http.StatusForbidden)
			return
		}
		writeJSON(w, http.StatusOK, assignmentJSON(assignment))
	}
}

func GeneratePlan(store Store, frameworks FrameworkReader, drafter Drafter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		if !claims.HasRole("admin") {
			writeError(w, http.StatusForbidden)
			return
		}
		var body openapi.GeneratePlan
		if !decode(w, r, &body) {
			return
		}
		assignment, err := store.GetAssignment(r.Context(), claims.OrganisationID, body.AssignmentID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		created, err := Generate(r.Context(), store, frameworks, drafter, PlanInput{
			OrganisationID: claims.OrganisationID,
			AssignmentID:   assignment.ID,
		}, r.Header.Get("Authorization"), assignment.FrameworkVersionID, assignment.RoleID)
		writeResult(w, http.StatusCreated, planJSON(created), err)
	}
}

func CreatePlan(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		var body openapi.CreatePlan
		if !decode(w, r, &body) {
			return
		}
		assignment, err := store.GetAssignment(r.Context(), claims.OrganisationID, body.AssignmentID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if !canAssign(claims, assignment.PersonID) {
			writeError(w, http.StatusForbidden)
			return
		}
		created, err := store.CreatePlan(r.Context(), PlanInput{
			OrganisationID: claims.OrganisationID,
			AssignmentID:   body.AssignmentID,
			Skills:         skillsFromJSON(body.Skills),
		})
		writeResult(w, http.StatusCreated, planJSON(created), err)
	}
}

func EditSkill(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := adminClaims(w, r)
		if !ok {
			return
		}
		var body openapi.EditSkill
		if !decode(w, r, &body) {
			return
		}
		items := make([]ItemInput, 0, len(body.AssessmentItems))
		for _, item := range body.AssessmentItems {
			items = append(items, ItemInput{Prompt: item.Prompt})
		}
		bands := make([]RubricInput, 0, len(body.Rubric))
		for _, band := range body.Rubric {
			bands = append(bands, RubricInput{Level: band.Level, Descriptor: band.Descriptor})
		}
		edited, err := store.EditSkill(r.Context(), EditSkillInput{
			OrganisationID:    claims.OrganisationID,
			PlanID:            chi.URLParam(r, "planId"),
			SkillID:           chi.URLParam(r, "skillId"),
			EditorPersonID:    claims.PersonID,
			RecommendedMethod: body.RecommendedMethod,
			Instructions:      body.Instructions,
			AssessmentItems:   items,
			Rubric:            bands,
		})
		writeResult(w, http.StatusOK, planJSON(edited), err)
	}
}

func SubmitPlan(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := adminClaims(w, r)
		if !ok {
			return
		}
		updated, err := store.SubmitPlan(r.Context(), ReviewInput{
			OrganisationID: claims.OrganisationID,
			PlanID:         chi.URLParam(r, "planId"),
			ActorPersonID:  claims.PersonID,
		})
		writeResult(w, http.StatusOK, planJSON(updated), err)
	}
}

func RequestChanges(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := adminClaims(w, r)
		if !ok {
			return
		}
		var body openapi.ChangeRequest
		if !decode(w, r, &body) {
			return
		}
		updated, err := store.RequestChanges(r.Context(), ReviewInput{
			OrganisationID: claims.OrganisationID,
			PlanID:         chi.URLParam(r, "planId"),
			ActorPersonID:  claims.PersonID,
			Reason:         body.Reason,
		})
		writeResult(w, http.StatusOK, planJSON(updated), err)
	}
}

func ApprovePlan(store Store, publisher Publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := adminClaims(w, r)
		if !ok {
			return
		}
		approvedAt := time.Now().UTC()
		updated, err := store.ApprovePlan(r.Context(), ReviewInput{
			OrganisationID: claims.OrganisationID,
			PlanID:         chi.URLParam(r, "planId"),
			ActorPersonID:  claims.PersonID,
			At:             approvedAt,
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}
		assignment, err := store.GetAssignment(r.Context(), claims.OrganisationID, updated.AssignmentID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if publisher != nil {
			if err := publisher.PublishPlanApproved(r.Context(), PlanApproved{
				Type:           "assessment.plan_approved",
				OrganisationID: claims.OrganisationID,
				PersonID:       assignment.PersonID,
				AssignmentID:   assignment.ID,
				PlanID:         updated.ID,
				ActorPersonID:  claims.PersonID,
				VersionNumber:  updated.VersionNumber,
				OccurredAt:     approvedAt.Format(time.RFC3339),
			}); err != nil {
				writeError(w, http.StatusBadGateway)
				return
			}
		}
		writeJSON(w, http.StatusOK, planJSON(updated))
	}
}

func CopyApprovedPlan(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := adminClaims(w, r)
		if !ok {
			return
		}
		created, err := store.CopyApprovedPlan(r.Context(), claims.OrganisationID, chi.URLParam(r, "planId"))
		writeResult(w, http.StatusCreated, planJSON(created), err)
	}
}

func adminClaims(w http.ResponseWriter, r *http.Request) (token.Claims, bool) {
	claims, ok := claimsFrom(w, r)
	if !ok {
		return token.Claims{}, false
	}
	if !claims.HasRole("admin") {
		writeError(w, http.StatusForbidden)
		return token.Claims{}, false
	}
	return claims, true
}

func ReadPlan(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		loaded, err := store.GetPlan(r.Context(), claims.OrganisationID, chi.URLParam(r, "planId"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		assignment, err := store.GetAssignment(r.Context(), claims.OrganisationID, loaded.AssignmentID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if !canRead(claims, assignment.PersonID) {
			writeError(w, http.StatusForbidden)
			return
		}
		writeJSON(w, http.StatusOK, planJSON(loaded))
	}
}

func ReplacePlan(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		var body openapi.ReplacePlan
		if !decode(w, r, &body) {
			return
		}
		loaded, err := store.GetPlan(r.Context(), claims.OrganisationID, chi.URLParam(r, "planId"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		assignment, err := store.GetAssignment(r.Context(), claims.OrganisationID, loaded.AssignmentID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if !canAssign(claims, assignment.PersonID) {
			writeError(w, http.StatusForbidden)
			return
		}
		replaced, err := store.ReplacePlan(r.Context(), ReplacePlanInput{
			OrganisationID: claims.OrganisationID,
			PlanID:         loaded.ID,
			Status:         body.Status,
			Skills:         skillsFromJSON(body.Skills),
		})
		writeResult(w, http.StatusOK, planJSON(replaced), err)
	}
}

func CreateSession(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := adminClaims(w, r)
		if !ok {
			return
		}
		created, err := store.CreateSession(r.Context(), claims.OrganisationID, chi.URLParam(r, "planId"))
		writeResult(w, http.StatusCreated, sessionJSON(created, false), err)
	}
}

func ListSessions(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		personID := r.URL.Query().Get("personId")
		includeProposals := false
		if personID == "" || personID == claims.PersonID {
			personID = claims.PersonID
		} else if canAssign(claims, personID) {
			includeProposals = true
		} else {
			writeError(w, http.StatusForbidden)
			return
		}
		sessions, err := store.ListSessions(r.Context(), claims.OrganisationID, personID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		encoded := make([]openapi.Session, 0, len(sessions))
		for _, session := range sessions {
			encoded = append(encoded, sessionJSON(session, includeProposals))
		}
		writeJSON(w, http.StatusOK, encoded)
	}
}

func ReadSession(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		session, err := store.GetSession(r.Context(), claims.OrganisationID, chi.URLParam(r, "sessionId"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if session.PersonID == claims.PersonID {
			writeJSON(w, http.StatusOK, sessionJSON(session, false))
			return
		}
		if !canAssign(claims, session.PersonID) {
			writeError(w, http.StatusForbidden)
			return
		}
		writeJSON(w, http.StatusOK, sessionJSON(session, true))
	}
}

func StartSession(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		started, err := store.StartSession(r.Context(), claims.OrganisationID, chi.URLParam(r, "sessionId"), claims.PersonID, time.Now().UTC())
		writeResult(w, http.StatusOK, sessionJSON(started, false), err)
	}
}

func SaveSessionAnswers(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		var body openapi.SaveSessionAnswers
		if !decode(w, r, &body) {
			return
		}
		answers := make([]AnswerInput, 0, len(body.Answers))
		for _, answer := range body.Answers {
			answers = append(answers, AnswerInput{ItemID: answer.ItemID, Answer: answer.Answer})
		}
		saved, err := store.SaveAnswers(r.Context(), claims.OrganisationID, chi.URLParam(r, "sessionId"), claims.PersonID, answers)
		writeResult(w, http.StatusOK, sessionJSON(saved, false), err)
	}
}

func SubmitSession(store Store, matcher BandMatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		sessionID := chi.URLParam(r, "sessionId")
		current, err := store.GetSession(r.Context(), claims.OrganisationID, sessionID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if current.PersonID != claims.PersonID {
			writeError(w, http.StatusForbidden)
			return
		}
		if current.Status != SessionStarted || !answersComplete(current) {
			proposals := []Proposal(nil)
			submitted, err := store.SubmitSession(r.Context(), claims.OrganisationID, sessionID, claims.PersonID, time.Now().UTC(), proposals)
			writeResult(w, http.StatusOK, sessionJSON(submitted, false), err)
			return
		}
		proposals, err := propose(r.Context(), store, matcher, claims.OrganisationID, current)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		submitted, err := store.SubmitSession(r.Context(), claims.OrganisationID, sessionID, claims.PersonID, time.Now().UTC(), proposals)
		writeResult(w, http.StatusOK, sessionJSON(submitted, false), err)
	}
}

func ConfirmSession(store Store, gaps GapRecorder, publisher Publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		session, err := store.GetSession(r.Context(), claims.OrganisationID, chi.URLParam(r, "sessionId"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if !canAssign(claims, session.PersonID) {
			writeError(w, http.StatusForbidden)
			return
		}
		if session.Status != SessionSubmitted || len(session.Proposals) == 0 {
			writeError(w, http.StatusConflict)
			return
		}
		if !session.ConfirmedAt.IsZero() {
			writeError(w, http.StatusConflict)
			return
		}
		if gaps == nil {
			writeStoreError(w, ErrProvider)
			return
		}
		row, err := store.GetPlan(r.Context(), claims.OrganisationID, session.PlanID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		assignment, err := store.GetAssignment(r.Context(), claims.OrganisationID, row.AssignmentID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		versionID := assignment.FrameworkVersionID
		profileVersionID := ""
		if baselines, ok := store.(interface {
			ProfileBaseline(context.Context, string, string) (ProfileBaseline, error)
		}); ok {
			baseline, baselineErr := baselines.ProfileBaseline(r.Context(), claims.OrganisationID, row.ID)
			if baselineErr == nil {
				versionID = baseline.ProfileVersionID
				profileVersionID = baseline.ProfileVersionID
			} else if !errors.Is(baselineErr, ErrNotFound) {
				writeStoreError(w, baselineErr)
				return
			}
		}
		explanations := make([]openapi.ConfirmedExplanation, 0, len(session.Proposals))
		for _, proposal := range session.Proposals {
			recorded, err := gaps.Record(r.Context(), r.Header.Get("Authorization"), GapAttempt{
				PersonID:           session.PersonID,
				FrameworkVersionID: versionID,
				CapabilityCode:     proposal.CapabilityID,
				Source:             proposal.EvidenceSource,
				Percentage:         proposal.Percentage,
				EvidenceIDs:        []string{session.ID},
				RequiredLevel:      float64(proposal.RequiredLevel),
			})
			if err != nil {
				writeStoreError(w, err)
				return
			}
			explanations = append(explanations, openapi.ConfirmedExplanation{
				CapabilityCode: recorded.CapabilityCode,
				CurrentLevel:   recorded.CurrentLevel,
				Gap:            recorded.Gap,
			})
		}
		confirmedAt := time.Now().UTC()
		confirmed, err := store.MarkSessionConfirmed(r.Context(), claims.OrganisationID, session.ID, confirmedAt)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if publisher != nil {
			if err := publisher.PublishAssessmentCompleted(r.Context(), AssessmentCompleted{
				Type:             "assessment.completed",
				OrganisationID:   claims.OrganisationID,
				PersonID:         session.PersonID,
				SessionID:        session.ID,
				PlanID:           session.PlanID,
				ActorPersonID:    claims.PersonID,
				ProfileVersionID: profileVersionID,
				OccurredAt:       confirmedAt.Format(time.RFC3339),
			}); err != nil {
				writeStoreError(w, err)
				return
			}
		}
		writeJSON(w, http.StatusOK, openapi.SessionConfirmation{
			Session:      sessionJSON(confirmed, true),
			Explanations: explanations,
		})
	}
}

func sessionJSON(row Session, includeProposals bool) openapi.Session {
	skills := make([]openapi.SessionSkill, 0, len(row.Skills))
	for _, skill := range row.Skills {
		items := make([]openapi.SessionItem, 0, len(skill.Items))
		for _, item := range skill.Items {
			items = append(items, openapi.SessionItem{ID: item.ID, Prompt: item.Prompt, Answer: item.Answer})
		}
		skills = append(skills, openapi.SessionSkill{Instructions: skill.Instructions, Items: items})
	}
	encoded := openapi.Session{
		ID:        row.ID,
		PlanID:    row.PlanID,
		PersonID:  row.PersonID,
		Status:    row.Status,
		Confirmed: !row.ConfirmedAt.IsZero(),
		Skills:    skills,
	}
	if includeProposals || !row.ConfirmedAt.IsZero() {
		encoded.Proposals = make([]openapi.SessionProposal, 0, len(row.Proposals))
		for _, proposal := range row.Proposals {
			encoded.Proposals = append(encoded.Proposals, openapi.SessionProposal{
				CapabilityID:  proposal.CapabilityID,
				Source:        proposal.EvidenceSource,
				RequiredLevel: proposal.RequiredLevel,
				Percentage:    proposal.Percentage,
				BandLevels:    append([]int(nil), proposal.BandLevels...),
			})
		}
	}
	return encoded
}

func canAssign(claims token.Claims, personID string) bool {
	if claims.HasRole("admin") {
		return true
	}
	if !claims.HasRole("manager") {
		return false
	}
	for _, id := range claims.ManagerOf {
		if id == personID {
			return true
		}
	}
	return false
}

func canRead(claims token.Claims, personID string) bool {
	return claims.HasRole("admin") || claims.PersonID == personID || canAssign(claims, personID)
}

func skillsFromJSON(inputs []openapi.SkillInput) []SkillInput {
	skills := make([]SkillInput, 0, len(inputs))
	for _, input := range inputs {
		items := make([]ItemInput, 0, len(input.AssessmentItems))
		for _, item := range input.AssessmentItems {
			items = append(items, ItemInput{Prompt: item.Prompt})
		}
		bands := make([]RubricInput, 0, len(input.Rubric))
		for _, band := range input.Rubric {
			bands = append(bands, RubricInput{Level: band.Level, Descriptor: band.Descriptor})
		}
		skills = append(skills, SkillInput{
			RequirementID:     input.RequirementID,
			CapabilityID:      input.CapabilityID,
			RequiredLevel:     input.RequiredLevel,
			RecommendedMethod: input.RecommendedMethod,
			Rationale:         input.Rationale,
			Instructions:      input.Instructions,
			AssessmentItems:   items,
			Rubric:            bands,
		})
	}
	return skills
}

func assignmentJSON(row Assignment) openapi.Assignment {
	encoded := openapi.Assignment{
		ID:                 row.ID,
		OrganisationID:     row.OrganisationID,
		PersonID:           row.PersonID,
		FrameworkVersionID: row.FrameworkVersionID,
		RoleID:             row.RoleID,
		AssignedByPersonID: row.AssignedByPersonID,
		AssignedAt:         row.AssignedAt.UTC().Format(time.RFC3339),
		Status:             row.Status,
	}
	if row.ExceptionReason != "" {
		reason := row.ExceptionReason
		encoded.ExceptionReason = &reason
	}
	return encoded
}

func planJSON(row Plan) openapi.Plan {
	skills := make([]openapi.PlanSkill, 0, len(row.Skills))
	for _, skill := range row.Skills {
		items := make([]openapi.AssessmentItem, 0, len(skill.AssessmentItems))
		for _, item := range skill.AssessmentItems {
			items = append(items, openapi.AssessmentItem{ID: item.ID, Prompt: item.Prompt})
		}
		bands := make([]openapi.RubricBand, 0, len(skill.Rubric))
		for _, band := range skill.Rubric {
			bands = append(bands, openapi.RubricBand{Level: band.Level, Descriptor: band.Descriptor})
		}
		skills = append(skills, openapi.PlanSkill{
			ID:                skill.ID,
			RequirementID:     skill.RequirementID,
			CapabilityID:      skill.CapabilityID,
			RequiredLevel:     skill.RequiredLevel,
			RecommendedMethod: skill.RecommendedMethod,
			Rationale:         skill.Rationale,
			Instructions:      skill.Instructions,
			RequiresSMEReview: skill.RequiresSMEReview,
			RubricNeedsReview: skill.RubricNeedsReview,
			EditedByPersonID:  optionalText(skill.EditedByPersonID),
			AssessmentItems:   items,
			Rubric:            bands,
		})
	}
	return openapi.Plan{
		ID:                 row.ID,
		OrganisationID:     row.OrganisationID,
		AssignmentID:       row.AssignmentID,
		VersionNumber:      row.VersionNumber,
		Status:             row.Status,
		ChangeReason:       optionalText(row.ChangeReason),
		ApprovedByPersonID: optionalText(row.ApprovedByPersonID),
		ApprovedAt:         optionalTime(row.ApprovedAt),
		PromptVersion:      optionalText(row.PromptVersion),
		ModelName:          optionalText(row.ModelName),
		Skills:             skills,
	}
}

func optionalText(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalTime(value time.Time) *string {
	if value.IsZero() {
		return nil
	}
	text := value.UTC().Format(time.RFC3339)
	return &text
}

func claimsFrom(w http.ResponseWriter, r *http.Request) (token.Claims, bool) {
	claims, ok := token.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized)
		return token.Claims{}, false
	}
	return claims, true
}

func decode(w http.ResponseWriter, r *http.Request, body any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(body); err != nil {
		writeError(w, http.StatusBadRequest)
		return false
	}
	return true
}

func writeResult(w http.ResponseWriter, status int, body any, err error) {
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, status, body)
}

func batchJSON(results []PersonResult) openapi.AssignmentBatch {
	encoded := make([]openapi.AssignmentResult, 0, len(results))
	for _, result := range results {
		row := openapi.AssignmentResult{
			PersonID: result.PersonID,
			Outcome:  result.Outcome,
			Reason:   result.Reason,
		}
		if result.AssignmentID != "" {
			id := result.AssignmentID
			row.AssignmentID = &id
		}
		encoded = append(encoded, row)
	}
	return openapi.AssignmentBatch{Results: encoded}
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound)
	case errors.Is(err, ErrInvalid), errors.Is(err, ErrRoleNotOnVersion):
		writeError(w, http.StatusBadRequest)
	case errors.Is(err, ErrImmutable), errors.Is(err, ErrNotPublished), errors.Is(err, ErrProfileUnpublished), errors.Is(err, ErrRolePeriod), errors.Is(err, ErrAlreadyGenerated), errors.Is(err, ErrNotEditable), errors.Is(err, ErrNotApproved):
		writeError(w, http.StatusConflict)
	case errors.Is(err, ErrForbidden):
		writeError(w, http.StatusForbidden)
	case errors.Is(err, ErrProvider):
		writeError(w, http.StatusServiceUnavailable)
	case errors.Is(err, ErrUpstream):
		writeError(w, http.StatusBadGateway)
	default:
		writeError(w, http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}
