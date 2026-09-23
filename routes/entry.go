// entry.go — HTTP routes over the catalog's Entries: the public read, the
// operator's read and writes, visibility, and the private parts.
package routes

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/aosanya/mwanachama-backend-shared/httpwire"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

// entryStatusTable maps the catalog's Entry sentinels to a status code.
//
// ErrEntryNotFound is 404 for every reason it can be returned — a missing
// slug, a pending Entry and a wrong key alike. Splitting them would hand
// back over HTTP exactly what the store refuses to tell a caller.
var entryStatusTable = map[error]int{
	catalog.ErrEntryNotFound:  http.StatusNotFound,
	catalog.ErrEntryExists:    http.StatusConflict,
	catalog.ErrSlugImmutable:  http.StatusConflict,
	catalog.ErrNotPrivate:     http.StatusConflict,
	catalog.ErrInvalidEntry:   http.StatusBadRequest,
	catalog.ErrInvalidPrivate: http.StatusBadRequest,
}

func writeEntryErr(w http.ResponseWriter, err error) {
	code := httpwire.StatusFor(err, entryStatusTable, http.StatusInternalServerError)
	if code == http.StatusInternalServerError {
		httpwire.WriteErr(w, code, "internal error")
		return
	}
	httpwire.WriteErr(w, code, err.Error())
}

// OpenEntry handles the storefront read: GET with an optional ?k= share
// key.
func OpenEntry(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := cm.OpenEntry(r.Context(), r.PathValue("slug"), r.URL.Query().Get("k"))
		if err != nil {
			writeEntryErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// ListEntries handles GET with ?visibility=, ?q=, ?limit= and any number of
// ?doc.<path>=<value> filters.
//
// The `doc.` prefix is this provisional package's stand-in for what the
// route spec will declare: the spec maps a domain's own query name onto a
// document path, so an agency catalog answers ?sector= and a library
// answers ?genre= over the same handler. Until the dispatcher exists, the
// caller spells the path itself and this package stays free of both words.
func ListEntries(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		limit, _ := strconv.Atoi(q.Get("limit"))

		var docs []catalog.DocFilter
		for key, vals := range q {
			path, ok := strings.CutPrefix(key, "doc.")
			if !ok || len(vals) == 0 {
				continue
			}
			docs = append(docs, catalog.DocFilter{Path: path, Value: vals[0]})
		}
		sort.Slice(docs, func(i, j int) bool { return docs[i].Path < docs[j].Path })

		out, err := cm.ListEntries(r.Context(), catalog.ListFilter{
			Visibility: catalog.Visibility(q.Get("visibility")),
			Docs:       docs,
			Search:     q.Get("q"),
			Limit:      limit,
		})
		if err != nil {
			writeEntryErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// GetEntry handles the operator's read, which ignores visibility.
func GetEntry(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := cm.GetEntry(r.Context(), r.PathValue("slug"))
		if err != nil {
			writeEntryErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// UpsertEntry handles PUT — create the Entry or replace its document.
func UpsertEntry(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in catalog.Entry
		if err := httpwire.ReadJSON(r, &in); err != nil {
			httpwire.WriteErr(w, http.StatusBadRequest, err.Error())
			return
		}
		in.Slug = r.PathValue("slug")
		out, err := cm.UpsertEntry(r.Context(), in)
		if err != nil {
			writeEntryErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// DeleteEntry handles DELETE.
func DeleteEntry(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := cm.DeleteEntry(r.Context(), r.PathValue("slug")); err != nil {
			writeEntryErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// SetVisibility handles POST {"visibility": "public"}.
func SetVisibility(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Visibility catalog.Visibility `json:"visibility"`
		}
		if err := httpwire.ReadJSON(r, &in); err != nil {
			httpwire.WriteErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := cm.SetVisibility(r.Context(), r.PathValue("slug"), in.Visibility)
		if err != nil {
			writeEntryErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// GetPrivate handles the operator's read of the private parts.
func GetPrivate(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := cm.GetPrivate(r.Context(), r.PathValue("slug"))
		if err != nil {
			writeEntryErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// PutPrivate handles a partial write of the private parts.
func PutPrivate(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in catalog.EntryPrivate
		if err := httpwire.ReadJSON(r, &in); err != nil {
			httpwire.WriteErr(w, http.StatusBadRequest, err.Error())
			return
		}
		in.Slug = r.PathValue("slug")
		out, err := cm.PutPrivate(r.Context(), in)
		if err != nil {
			writeEntryErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// IssueShareLink handles POST {"label": "..."}, returning the raw key once.
func IssueShareLink(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Label string `json:"label"`
			By    string `json:"by"`
		}
		if err := httpwire.ReadJSON(r, &in); err != nil {
			httpwire.WriteErr(w, http.StatusBadRequest, err.Error())
			return
		}
		link, raw, err := cm.IssueShareLink(r.Context(), r.PathValue("slug"), in.Label, in.By)
		if err != nil {
			writeEntryErr(w, err)
			return
		}
		// The raw key appears here and nowhere else, ever again.
		httpwire.WriteJSON(w, http.StatusCreated, struct {
			catalog.ShareLink
			Key string `json:"key"`
		}{ShareLink: link, Key: raw})
	}
}

// ListShareLinks handles GET, never carrying a key.
func ListShareLinks(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := cm.ListShareLinks(r.Context(), r.PathValue("slug"))
		if err != nil {
			writeEntryErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}

// RetireShareLink handles POST against one link.
func RetireShareLink(cm catalog.CatalogManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := cm.RetireShareLink(r.Context(), r.PathValue("linkID"))
		if err != nil {
			writeSuggestionErr(w, err)
			return
		}
		httpwire.WriteJSON(w, http.StatusOK, out)
	}
}
