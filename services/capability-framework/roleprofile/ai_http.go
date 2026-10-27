package roleprofile

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func startClarification(workflow *Workflow) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		session, err := workflow.Clarify(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID)
		writeResult(w, http.StatusOK, session, err)
	}
}

func readClarification(workflow *Workflow) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		session, err := workflow.Session(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID)
		writeResult(w, http.StatusOK, session, err)
	}
}

func answerClarification(workflow *Workflow) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		var body struct {
			Answers []struct {
				QuestionID string `json:"questionId"`
				Value      string `json:"value"`
			} `json:"answers"`
		}
		if !decode(w, r, &body) {
			return
		}
		if len(body.Answers) == 0 {
			writeCode(w, http.StatusBadRequest, "invalid")
			return
		}
		answers := make([]AnswerInput, 0, len(body.Answers))
		for _, answer := range body.Answers {
			answers = append(answers, AnswerInput{QuestionID: answer.QuestionID, Value: answer.Value})
		}
		session, err := workflow.Answer(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID, answers)
		writeResult(w, http.StatusOK, session, err)
	}
}

func startGeneration(workflow *Workflow) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		var body struct {
			ContextVersion int    `json:"contextVersion"`
			IdempotencyKey string `json:"idempotencyKey"`
		}
		if !decode(w, r, &body) {
			return
		}
		job, err := workflow.Enqueue(r.Context(), claims.OrganisationID, chi.URLParam(r, "versionId"), claims.PersonID, body.IdempotencyKey, body.ContextVersion)
		status := http.StatusAccepted
		if err == nil && job.Status != JobQueued && job.Status != JobRunning {
			status = http.StatusOK
		}
		writeResult(w, status, job, err)
	}
}

func readGeneration(workflow *Workflow) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		job, err := workflow.GetJob(r.Context(), claims.OrganisationID, chi.URLParam(r, "jobId"))
		writeResult(w, http.StatusOK, job, err)
	}
}

func retryGeneration(workflow *Workflow) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		job, err := workflow.Retry(r.Context(), claims.OrganisationID, chi.URLParam(r, "jobId"), claims.PersonID)
		writeResult(w, http.StatusAccepted, job, err)
	}
}

func cancelGeneration(workflow *Workflow) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := requireAdmin(w, r)
		if !ok {
			return
		}
		job, err := workflow.Cancel(r.Context(), claims.OrganisationID, chi.URLParam(r, "jobId"))
		writeResult(w, http.StatusOK, job, err)
	}
}
