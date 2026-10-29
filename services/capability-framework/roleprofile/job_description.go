package roleprofile

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/capability-framework/db"
)

const MaxJobDescriptionBytes = 5 << 20

type JobDescriptionFile struct {
	ID        string `json:"id"`
	FileName  string `json:"fileName"`
	MediaType string `json:"mediaType"`
	ByteSize  int    `json:"byteSize"`
	Detail    string `json:"detail"`
	Text      string `json:"text,omitempty"`
}

type JobDescriptionStore interface {
	SaveJobDescription(ctx context.Context, organisationID, versionID, actorID, fileName string, body []byte) (JobDescriptionFile, error)
	ReadJobDescription(ctx context.Context, organisationID, versionID, fileID string) (JobDescriptionFile, []byte, error)
}

type storedJobFile struct {
	JobDescriptionFile
	OrganisationID string
	VersionID      string
	Body           []byte
	CreatedAt      time.Time
}

func classifyJobDescription(name string, body []byte) (string, string, string, error) {
	if len(body) == 0 || len(body) > MaxJobDescriptionBytes {
		return "", "", "", ErrInvalid
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".txt":
		if bytes.IndexByte(body, 0) >= 0 || !utf8.Valid(body) {
			return "", "", "", ErrInvalid
		}
		return "text/plain", "The file text is shown for you to review. It is not saved with the role until you save the role information, and it was not read by a model.", jobText(body), nil
	case ".pdf":
		if !bytes.HasPrefix(body, []byte("%PDF")) {
			return "", "", "", ErrInvalid
		}
		return "application/pdf", "The file is stored with this role. Text was not extracted from the PDF.", "", nil
	case ".docx":
		if !bytes.HasPrefix(body, []byte("PK")) {
			return "", "", "", ErrInvalid
		}
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "The file is stored with this role. Text was not extracted from the Word document.", "", nil
	case ".doc":
		if !bytes.HasPrefix(body, []byte{0xD0, 0xCF, 0x11, 0xE0}) {
			return "", "", "", ErrInvalid
		}
		return "application/msword", "The file is stored with this role. Text was not extracted from the Word document.", "", nil
	default:
		return "", "", "", ErrInvalid
	}
}

func jobText(body []byte) string {
	runes := []rune(string(body))
	if len(runes) > 20000 {
		runes = runes[:20000]
	}
	return string(runes)
}

func (m *Memory) SaveJobDescription(_ context.Context, organisationID, versionID, actorID, fileName string, body []byte) (JobDescriptionFile, error) {
	media, detail, text, err := classifyJobDescription(fileName, body)
	if err != nil || actorID == "" {
		return JobDescriptionFile{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	version, ok := m.version(organisationID, versionID)
	if !ok {
		return JobDescriptionFile{}, ErrNotFound
	}
	if version.Status != StatusDraft {
		return JobDescriptionFile{}, ErrImmutable
	}
	id, err := newID()
	if err != nil {
		return JobDescriptionFile{}, err
	}
	record := storedJobFile{
		JobDescriptionFile: JobDescriptionFile{ID: id, FileName: filepath.Base(fileName), MediaType: media, ByteSize: len(body), Detail: detail},
		OrganisationID:     organisationID,
		VersionID:          versionID,
		Body:               append([]byte{}, body...),
		CreatedAt:          clock(),
	}
	m.jobFiles = append(m.jobFiles, record)
	record.Text = text
	return record.JobDescriptionFile, nil
}

func (m *Memory) ReadJobDescription(_ context.Context, organisationID, versionID, fileID string) (JobDescriptionFile, []byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, file := range m.jobFiles {
		if file.OrganisationID == organisationID && file.VersionID == versionID && file.ID == fileID {
			return file.JobDescriptionFile, append([]byte{}, file.Body...), nil
		}
	}
	return JobDescriptionFile{}, nil, ErrNotFound
}

func (s SQLStore) SaveJobDescription(ctx context.Context, organisationID, versionID, actorID, fileName string, body []byte) (JobDescriptionFile, error) {
	media, detail, text, err := classifyJobDescription(fileName, body)
	if err != nil || actorID == "" {
		return JobDescriptionFile{}, ErrInvalid
	}
	var saved JobDescriptionFile
	err = s.inTxWith(ctx, func(q *db.Queries, tx *sql.Tx) error {
		row, err := q.GetRoleProfileVersion(ctx, db.GetRoleProfileVersionParams{OrganisationID: organisationID, ID: versionID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if row.Status != StatusDraft {
			return ErrImmutable
		}
		id, err := newID()
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO role_profile_job_descriptions (
				id, organisation_id, profile_version_id, file_name, media_type, byte_size, body, detail, created_by_person_id, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, id, organisationID, versionID, filepath.Base(fileName), media, len(body), body, detail, actorID, clock()); err != nil {
			return err
		}
		saved = JobDescriptionFile{ID: id, FileName: filepath.Base(fileName), MediaType: media, ByteSize: len(body), Detail: detail, Text: text}
		return nil
	})
	return saved, err
}

func (s SQLStore) ReadJobDescription(ctx context.Context, organisationID, versionID, fileID string) (JobDescriptionFile, []byte, error) {
	var file JobDescriptionFile
	var body []byte
	err := s.database.QueryRowContext(ctx, `
		SELECT id, file_name, media_type, byte_size, body, detail
		FROM role_profile_job_descriptions
		WHERE organisation_id = $1 AND profile_version_id = $2 AND id = $3
	`, organisationID, versionID, fileID).Scan(&file.ID, &file.FileName, &file.MediaType, &file.ByteSize, &body, &file.Detail)
	if errors.Is(err, sql.ErrNoRows) {
		return JobDescriptionFile{}, nil, ErrNotFound
	}
	return file, body, err
}

func uploadJobDescription(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireMaintainer(w, r)
		if !ok {
			return
		}
		files, ok := store.(JobDescriptionStore)
		if !ok {
			writeCode(w, http.StatusNotFound, "not_found")
			return
		}
		body, name, ok := jobFile(w, r)
		if !ok {
			return
		}
		saved, err := files.SaveJobDescription(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID, name, body)
		if err != nil {
			if errors.Is(err, ErrInvalid) {
				writeDetail(w, http.StatusBadRequest, "invalid", "Upload a TXT, PDF, DOC, or DOCX file of 5 MB or less whose contents match the file name.")
				return
			}
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, saved)
	}
}

func downloadJobDescription(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireMaintainer(w, r)
		if !ok {
			return
		}
		files, ok := store.(JobDescriptionStore)
		if !ok {
			writeCode(w, http.StatusNotFound, "not_found")
			return
		}
		file, body, err := files.ReadJobDescription(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), chi.URLParam(r, "fileId"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		w.Header().Set("Content-Type", file.MediaType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+safeAttachmentName(file.FileName)+"\"")
		w.Header().Set("Cache-Control", "private, no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

func safeAttachmentName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
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

func jobFile(w http.ResponseWriter, r *http.Request) ([]byte, string, bool) {
	if err := r.ParseMultipartForm(MaxJobDescriptionBytes); err != nil {
		writeDetail(w, http.StatusBadRequest, "invalid", "Upload a TXT, PDF, DOC, or DOCX file of 5 MB or less whose contents match the file name.")
		return nil, "", false
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeDetail(w, http.StatusBadRequest, "invalid", "Choose a job description file.")
		return nil, "", false
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, MaxJobDescriptionBytes+1))
	if err != nil || len(body) > MaxJobDescriptionBytes {
		writeDetail(w, http.StatusRequestEntityTooLarge, "invalid", "The file must be 5 MB or smaller.")
		return nil, "", false
	}
	return body, header.Filename, true
}

func writeDetail(w http.ResponseWriter, status int, code, detail string) {
	writeJSON(w, status, map[string]string{"error": code, "detail": detail})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
