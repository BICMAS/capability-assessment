package framework

import (
	"bytes"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/capability-framework/blob"
	"github.com/bicmas-academy/learn/services/capability-framework/csvimport"
	"github.com/bicmas-academy/learn/services/capability-framework/openapi"
)

const maxImportBytes = 8 << 20

func ImportCSV(store Store, blobs blob.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest)
			return
		}
		defer file.Close()
		payload, err := io.ReadAll(io.LimitReader(file, maxImportBytes+1))
		if err != nil {
			writeError(w, http.StatusBadRequest)
			return
		}
		if len(payload) > maxImportBytes {
			writeError(w, http.StatusBadRequest)
			return
		}
		importID, err := newID()
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		objectKey := "organisations/" + claims.OrganisationID + "/imports/" + importID + ".csv"
		if err := blobs.Put(r.Context(), objectKey, payload, "text/csv"); err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		parsed, err := csvimport.Parse(bytes.NewReader(payload))
		if err != nil {
			writeError(w, http.StatusBadRequest)
			return
		}
		if len(parsed.MissingColumns) > 0 {
			writeJSON(w, http.StatusBadRequest, openapi.ImportError{
				ObjectKey:      objectKey,
				MissingColumns: parsed.MissingColumns,
				RowErrors:      []openapi.RowError{},
			})
			return
		}
		imported, err := store.ApplyImport(r.Context(), claims.OrganisationID, objectKey, parsed)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		response := openapi.ImportResult{
			ObjectKey: imported.ObjectKey,
			RowErrors: rowErrorsJSON(imported.RowErrors),
			Proposals: proposalsJSON(imported.Proposals),
		}
		if imported.Version != nil {
			version := versionJSON(*imported.Version)
			response.Version = &version
		}
		status := http.StatusCreated
		if imported.Version == nil {
			status = http.StatusUnprocessableEntity
		}
		writeJSON(w, status, response)
	}
}

func SubmitForReview(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		version, err := store.SubmitForReview(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"))
		writeResult(w, http.StatusOK, version, err)
	}
}

func DecideProposal(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		var body openapi.DecideProposal
		if !decode(w, r, &body) {
			return
		}
		proposal, err := store.DecideProposal(
			r.Context(),
			claims.OrganisationID,
			chi.URLParam(r, "versionId"),
			chi.URLParam(r, "proposalId"),
			body.Status,
			body.Message,
		)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, proposalJSON(proposal))
	}
}

func rowErrorsJSON(errors []RowError) []openapi.RowError {
	response := make([]openapi.RowError, 0, len(errors))
	for _, rowError := range errors {
		response = append(response, openapi.RowError{Row: rowError.Row, Message: rowError.Message})
	}
	return response
}

func proposalsJSON(proposals []Proposal) []openapi.Proposal {
	response := make([]openapi.Proposal, 0, len(proposals))
	for _, proposal := range proposals {
		response = append(response, proposalJSON(proposal))
	}
	return response
}

func proposalJSON(proposal Proposal) openapi.Proposal {
	return openapi.Proposal{
		Id:             proposal.ID,
		Kind:           proposal.Kind,
		Status:         proposal.Status,
		CapabilityCode: proposal.CapabilityCode,
		Message:        proposal.Message,
	}
}
