package directory

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func PublishOrganogram(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireOrganogramMaintainer(w, r)
		if !ok {
			return
		}
		organogram, err := store.PublishOrganogram(r.Context(), claims.OrganisationID, chi.URLParam(r, "organogramId"))
		if err != nil {
			writeDirectoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, organogram)
	}
}

func UpdatePosition(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireOrganogramMaintainer(w, r)
		if !ok {
			return
		}
		var body PositionInput
		if !decodePosition(w, r, &body) {
			return
		}
		updated, err := store.UpdatePosition(r.Context(), claims.OrganisationID, chi.URLParam(r, "organogramId"), chi.URLParam(r, "positionId"), body)
		if err != nil {
			writeDirectoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

func DeletePosition(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireOrganogramMaintainer(w, r)
		if !ok {
			return
		}
		err := store.DeletePosition(r.Context(), claims.OrganisationID, chi.URLParam(r, "organogramId"), chi.URLParam(r, "positionId"))
		if err != nil {
			writeDirectoryError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func UploadOrganogramFile(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireOrganogramMaintainer(w, r)
		if !ok {
			return
		}
		if err := r.ParseMultipartForm(MaxOrganogramBytes); err != nil {
			http.Error(w, "The file must be 5 MB or smaller.", http.StatusRequestEntityTooLarge)
			return
		}
		upload, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Choose a file to upload.", http.StatusBadRequest)
			return
		}
		defer upload.Close()
		body, err := io.ReadAll(io.LimitReader(upload, MaxOrganogramBytes+1))
		if err != nil {
			writeError(w, http.StatusBadRequest)
			return
		}
		if len(body) > MaxOrganogramBytes {
			http.Error(w, "The file must be 5 MB or smaller.", http.StatusRequestEntityTooLarge)
			return
		}
		result, err := store.ImportOrganogramFile(r.Context(), claims.OrganisationID, chi.URLParam(r, "organogramId"), claims.PersonID, path.Base(header.Filename), body, SheetMapping{
			Title:      r.FormValue("titleColumn"),
			ReportsTo:  r.FormValue("reportsToColumn"),
			Department: r.FormValue("departmentColumn"),
		}, time.Now().UTC().Truncate(time.Second))
		if errors.Is(err, ErrInvalid) {
			http.Error(w, "Upload a PNG, JPG, PDF, XLS, or XLSX file whose contents match the file name.", http.StatusBadRequest)
			return
		}
		if err != nil {
			writeDirectoryError(w, err)
			return
		}
		if result.Rejected == nil {
			result.Rejected = []ImportRejection{}
		}
		if result.Headers == nil {
			result.Headers = []string{}
		}
		writeJSON(w, http.StatusCreated, result)
	}
}

func ListOrganogramFiles(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireOrganogramReader(w, r)
		if !ok {
			return
		}
		files, err := store.ListOrganogramFiles(r.Context(), claims.OrganisationID, chi.URLParam(r, "organogramId"))
		if err != nil {
			writeDirectoryError(w, err)
			return
		}
		if files == nil {
			files = []OrganogramFile{}
		}
		writeJSON(w, http.StatusOK, files)
	}
}

func ReadOrganogramFile(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireOrganogramReader(w, r)
		if !ok {
			return
		}
		file, err := store.ReadOrganogramFile(r.Context(), claims.OrganisationID, chi.URLParam(r, "organogramId"), chi.URLParam(r, "fileId"))
		if err != nil {
			writeDirectoryError(w, err)
			return
		}
		w.Header().Set("Content-Type", file.MediaType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+safeAttachmentName(file.FileName)+"\"")
		w.Header().Set("Cache-Control", "private, no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(file.Body)
	}
}

func safeAttachmentName(name string) string {
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	var cleaned strings.Builder
	for _, r := range name {
		if r < 0x20 || r == 0x7f || r == '"' || r == ';' || r == '\\' {
			continue
		}
		cleaned.WriteRune(r)
	}
	result := strings.Trim(cleaned.String(), ". ")
	if result == "" {
		return "download"
	}
	return result
}

func DownloadOrganogramTemplate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := requireOrganogramReader(w, r); !ok {
			return
		}
		body, err := TemplateXLSX()
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="organogram-template.xlsx"`)
		w.Header().Set("Cache-Control", "private, no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

func decodePosition(w http.ResponseWriter, r *http.Request, body *PositionInput) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var payload struct {
		DepartmentID        string `json:"departmentId"`
		Title               string `json:"title"`
		PersonID            string `json:"personId"`
		ReportsToPositionID string `json:"reportsToPositionId"`
	}
	if err := decoder.Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest)
		return false
	}
	*body = PositionInput{DepartmentID: payload.DepartmentID, Title: payload.Title, PersonID: payload.PersonID, ReportsToPositionID: payload.ReportsToPositionID}
	return true
}

func writeDirectoryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound)
	case errors.Is(err, ErrDuplicate):
		writeError(w, http.StatusConflict)
	case errors.Is(err, ErrTooLarge):
		http.Error(w, "The file must be 5 MB or smaller.", http.StatusRequestEntityTooLarge)
	case errors.Is(err, ErrCycle):
		http.Error(w, "That reporting line would create a cycle.", http.StatusBadRequest)
	case errors.Is(err, ErrHasReports):
		http.Error(w, "Move the positions that report to this one before removing it.", http.StatusBadRequest)
	case errors.Is(err, ErrInvalid):
		http.Error(w, "The organogram request is not valid.", http.StatusBadRequest)
	default:
		writeError(w, http.StatusInternalServerError)
	}
}
