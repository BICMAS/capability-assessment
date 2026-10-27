package framework

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/capability-framework/openapi"
	"github.com/bicmas-academy/learn/services/token"
)

func ListVersions(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		rows, err := store.ListVersions(r.Context(), claims.OrganisationID)
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		response := make([]openapi.FrameworkVersion, 0, len(rows))
		for _, row := range rows {
			response = append(response, versionJSON(row))
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func ReadVersion(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		version, err := store.GetVersion(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"))
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, versionJSON(version))
	}
}

func CreateDraft(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		var body openapi.CreateFramework
		if !decode(w, r, &body) {
			return
		}
		if body.Name == "" {
			writeError(w, http.StatusBadRequest)
			return
		}
		version, err := store.CreateDraft(r.Context(), claims.OrganisationID, body.Name)
		writeResult(w, http.StatusCreated, version, err)
	}
}

func AddRole(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		var body openapi.CreateRole
		if !decode(w, r, &body) {
			return
		}
		if body.Name == "" {
			writeError(w, http.StatusBadRequest)
			return
		}
		role, err := store.AddRole(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), body.Name)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, openapi.Role{Id: role.ID, Name: role.Name})
	}
}

func AddCapability(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		var body openapi.CreateCapability
		if !decode(w, r, &body) {
			return
		}
		if body.Name == "" {
			writeError(w, http.StatusBadRequest)
			return
		}
		capability, err := store.AddCapability(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), body.Name)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, capabilityJSON(capability))
	}
}

func AddRequirement(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		var body openapi.CreateRequirement
		if !decode(w, r, &body) {
			return
		}
		if body.RoleId == "" || body.CapabilityId == "" {
			writeError(w, http.StatusBadRequest)
			return
		}
		requirement, err := store.AddRequirement(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), body.RoleId, body.CapabilityId, body.RequiredLevel)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, requirementJSON(requirement))
	}
}

func AddRoleCapability(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		var body openapi.CreateRoleCapability
		if !decode(w, r, &body) {
			return
		}
		if body.Name == "" || body.RoleId == "" {
			writeError(w, http.StatusBadRequest)
			return
		}
		capability, requirement, err := store.AddRoleCapability(
			r.Context(),
			claims.OrganisationID,
			chi.URLParam(r, "versionId"),
			body.RoleId,
			body.Name,
			body.RequiredLevel,
		)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, openapi.RoleCapability{
			Capability:  capabilityJSON(capability),
			Requirement: requirementJSON(requirement),
		})
	}
}

func AddIndicator(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		var body openapi.CreateIndicator
		if !decode(w, r, &body) {
			return
		}
		if body.CapabilityId == "" || body.Statement == "" {
			writeError(w, http.StatusBadRequest)
			return
		}
		indicator, err := store.AddIndicator(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), body.CapabilityId, body.Level, body.Statement)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, openapi.Indicator{Id: indicator.ID, Statement: indicator.Statement})
	}
}

func PatchRole(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		var body openapi.PatchRole
		if !decode(w, r, &body) {
			return
		}
		if body.RoleId == "" || body.RoleName == "" {
			writeError(w, http.StatusBadRequest)
			return
		}
		version, err := store.PatchRole(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), body.RoleId, body.RoleName)
		writeResult(w, http.StatusOK, version, err)
	}
}

func Publish(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		versionID := chi.URLParam(r, "versionId")
		occurredAt := time.Now().UTC().Truncate(time.Second)
		event, err := newPublicationEvent(claims.OrganisationID, versionID, claims.PersonID, correlationID(r), occurredAt)
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		version, err := store.PublishWithEvent(r.Context(), claims.OrganisationID, versionID, event)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, versionJSON(version))
	}
}

func Approve(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		version, err := store.Approve(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"))
		writeResult(w, http.StatusOK, version, err)
	}
}

func UpdateIndicator(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		var body openapi.UpdateIndicator
		if !decode(w, r, &body) {
			return
		}
		if body.Statement == "" {
			writeError(w, http.StatusBadRequest)
			return
		}
		indicator, err := store.UpdateIndicator(
			r.Context(),
			claims.OrganisationID,
			chi.URLParam(r, "versionId"),
			chi.URLParam(r, "indicatorId"),
			body.Statement,
		)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, openapi.Indicator{Id: indicator.ID, Statement: indicator.Statement})
	}
}

func ListPublished(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := token.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized)
			return
		}
		rows, err := store.ListVersions(r.Context(), claims.OrganisationID)
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		response := make([]openapi.FrameworkVersion, 0)
		for _, row := range rows {
			if row.Status != StatusPublished {
				continue
			}
			response = append(response, versionJSON(row))
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func CreateNextDraft(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		version, err := store.CreateNextDraft(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"))
		writeResult(w, http.StatusCreated, version, err)
	}
}

func writeResult(w http.ResponseWriter, status int, version Version, err error) {
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, status, versionJSON(version))
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound)
	case errors.Is(err, ErrImmutable), errors.Is(err, ErrPending):
		writeError(w, http.StatusConflict)
	case errors.Is(err, ErrInvalid):
		writeError(w, http.StatusBadRequest)
	default:
		writeError(w, http.StatusInternalServerError)
	}
}

func requireAdmin(w http.ResponseWriter, r *http.Request) (token.Claims, bool) {
	claims, ok := token.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized)
		return token.Claims{}, false
	}
	if !claims.HasRole("admin") {
		writeError(w, http.StatusForbidden)
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

func versionJSON(version Version) openapi.FrameworkVersion {
	roles := make([]openapi.Role, 0, len(version.Roles))
	for _, role := range version.Roles {
		roles = append(roles, openapi.Role{Id: role.ID, Name: role.Name})
	}
	capabilities := make([]openapi.Capability, 0, len(version.Capabilities))
	for _, capability := range version.Capabilities {
		capabilities = append(capabilities, capabilityJSON(capability))
	}
	requirements := make([]openapi.Requirement, 0, len(version.Requirements))
	for _, requirement := range version.Requirements {
		requirements = append(requirements, requirementJSON(requirement))
	}
	return openapi.FrameworkVersion{
		Id:             version.ID,
		OrganisationId: version.OrganisationID,
		FrameworkId:    version.FrameworkID,
		FrameworkName:  version.FrameworkName,
		VersionNumber:  version.VersionNumber,
		Status:         version.Status,
		Roles:          roles,
		Capabilities:   capabilities,
		Requirements:   requirements,
		RowErrors:      rowErrorsJSON(version.RowErrors),
		Proposals:      proposalsJSON(version.Proposals),
	}
}

func capabilityJSON(capability Capability) openapi.Capability {
	levels := make([]openapi.ProficiencyLevel, 0, len(capability.Levels))
	for _, level := range capability.Levels {
		indicators := make([]openapi.Indicator, 0, len(level.Indicators))
		for _, indicator := range level.Indicators {
			indicators = append(indicators, openapi.Indicator{Id: indicator.ID, Statement: indicator.Statement})
		}
		levels = append(levels, openapi.ProficiencyLevel{
			Id:         level.ID,
			Level:      level.Level,
			Indicators: indicators,
		})
	}
	return openapi.Capability{Id: capability.ID, Name: capability.Name, Levels: levels}
}

func requirementJSON(requirement Requirement) openapi.Requirement {
	return openapi.Requirement{
		Id:            requirement.ID,
		RoleId:        requirement.RoleID,
		CapabilityId:  requirement.CapabilityID,
		RequiredLevel: requirement.RequiredLevel,
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
