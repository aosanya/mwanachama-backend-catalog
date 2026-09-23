// suggestion.go — HTTP routes over the mining backlog, the run reports,
// the settings and the page-view counters.
package routes

import (
	"net/http"
	"strconv"

	"github.com/aosanya/mwanachama-backend-shared/httpwire"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

var suggestionStatusTable = map[error]int{
	catalog.ErrSuggestionNotFound:  http.StatusNotFound,
	catalog.ErrEntryNotFound:       http.StatusNotFound,
	catalog.ErrShareLinkNotFound:   http.StatusNotFound,
	catalog.ErrSettingNotFound:     http.StatusNotFound,
	catalog.ErrSuggestionDuplicate: http.StatusConflict,
	catalog.ErrSuggestionStatus:    http.StatusConflict,
	catalog.ErrInvalidSuggestion:   http.StatusBadRequest,
	catalog.ErrAttemptNeedsComment: http.StatusBadRequest,
	catalog.ErrInvalidComment:      http.StatusBadRequest,
	catalog.ErrInvalidRun:          http.StatusBadRequest,
	catalog.ErrInvalidSetting:      http.StatusBadRequest,
	catalog.ErrInvalidPageView:     http.StatusBadRequest,
}

func writeSuggestionErr(w http.ResponseWriter, err error) {
	code := httpwire.StatusFor(err, suggestionStatusTable, http.StatusInternalServerError)
	if code == http.StatusInternalServerError {
		httpwire.WriteErr(w, code, "internal error")
		return
	}
	httpwire.WriteErr(w, code, err.Error())
}

// FileSuggestion handles POST of a new proposal.
func FileSuggestion(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in catalog.Suggestion
		if err := httpwire.ReadJSON(r, &in); err != nil {
			httpwire.WriteErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := cm.FileSuggestion(r.Context(), in)
		if err != nil {
			writeSuggestionErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusCreated, out)
	}
}

// ListSuggestions handles GET with repeated ?status= values.
func ListSuggestions(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var statuses []catalog.SuggestionStatus
		for _, s := range r.URL.Query()["status"] {
			statuses = append(statuses, catalog.SuggestionStatus(s))
		}
		out, err := cm.ListSuggestions(r.Context(), statuses...)
		if err != nil {
			writeSuggestionErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// GetSuggestion handles GET of one proposal.
func GetSuggestion(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := cm.GetSuggestion(r.Context(), r.PathValue("suggestionID"))
		if err != nil {
			writeSuggestionErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// DecideSuggestion handles POST {"status": "approved", "priority": 5}.
func DecideSuggestion(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Status   catalog.SuggestionStatus `json:"status"`
			Priority int                      `json:"priority"`
		}
		if err := httpwire.ReadJSON(r, &in); err != nil {
			httpwire.WriteErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := cm.DecideSuggestion(r.Context(), r.PathValue("suggestionID"), in.Status, in.Priority)
		if err != nil {
			writeSuggestionErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// RecordSuggestionAttempt handles POST {"run_id": "...", "comment": "..."}.
func RecordSuggestionAttempt(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			RunID   string `json:"run_id"`
			Comment string `json:"comment"`
		}
		if err := httpwire.ReadJSON(r, &in); err != nil {
			httpwire.WriteErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := cm.RecordSuggestionAttempt(r.Context(), r.PathValue("suggestionID"), in.RunID, in.Comment)
		if err != nil {
			writeSuggestionErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// FulfilSuggestion handles POST {"slug": "...", "run_id": "...", "note": "..."}.
func FulfilSuggestion(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Slug  string `json:"slug"`
			RunID string `json:"run_id"`
			Note  string `json:"note"`
		}
		if err := httpwire.ReadJSON(r, &in); err != nil {
			httpwire.WriteErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := cm.FulfilSuggestion(r.Context(), r.PathValue("suggestionID"), in.Slug, in.RunID, in.Note)
		if err != nil {
			writeSuggestionErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// ListSuggestionComments handles GET of one proposal's notes.
func ListSuggestionComments(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := cm.ListSuggestionComments(r.Context(), r.PathValue("suggestionID"))
		if err != nil {
			writeSuggestionErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// CommentOnSuggestion handles POST {"author": "admin", "body": "..."}.
func CommentOnSuggestion(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Author catalog.CommentAuthor `json:"author"`
			Body   string                `json:"body"`
		}
		if err := httpwire.ReadJSON(r, &in); err != nil {
			httpwire.WriteErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := cm.CommentOnSuggestion(r.Context(), r.PathValue("suggestionID"), in.Author, in.Body)
		if err != nil {
			writeSuggestionErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusCreated, out)
	}
}

// ListSuggestionMatches handles GET of the Entries answering a proposal.
func ListSuggestionMatches(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := cm.ListSuggestionMatches(r.Context(), r.PathValue("suggestionID"))
		if err != nil {
			writeSuggestionErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// RecordRun handles POST of one run report.
func RecordRun(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in catalog.MiningRun
		if err := httpwire.ReadJSON(r, &in); err != nil {
			httpwire.WriteErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := cm.RecordRun(r.Context(), in)
		if err != nil {
			writeSuggestionErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusCreated, out)
	}
}

// ListRuns handles GET with ?limit=.
func ListRuns(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		out, err := cm.ListRuns(r.Context(), limit)
		if err != nil {
			writeSuggestionErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// GetSetting handles GET of one knob.
func GetSetting(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := cm.GetSetting(r.Context(), r.PathValue("key"))
		if err != nil {
			writeSuggestionErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// PutSetting handles PUT {"value": "..."}.
func PutSetting(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Value string `json:"value"`
		}
		if err := httpwire.ReadJSON(r, &in); err != nil {
			httpwire.WriteErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := cm.PutSetting(r.Context(), r.PathValue("key"), in.Value)
		if err != nil {
			writeSuggestionErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// RecordPageView handles POST {"page": "agency", "slug": "..."}.
func RecordPageView(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Page catalog.PageKind `json:"page"`
			Slug string           `json:"slug"`
		}
		if err := httpwire.ReadJSON(r, &in); err != nil {
			httpwire.WriteErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := cm.RecordPageView(r.Context(), in.Page, in.Slug); err != nil {
			writeSuggestionErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ListPageViews handles GET of every counted page.
func ListPageViews(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := cm.ListPageViews(r.Context())
		if err != nil {
			writeSuggestionErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}
