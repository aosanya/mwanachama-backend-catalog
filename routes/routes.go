package routes

import (
	"net/http"

	"github.com/aosanya/mwanachama-backend-shared/httpwire"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

// Route is httpwire.Route — one address this package answers, relative to
// wherever the mounting process prefixes it.
type Route = httpwire.Route

// PublicRoutes is everything an anonymous storefront reader may reach: one
// Entry by slug with an optional ?k= key, the public listing, and a page
// view. Nothing here exposes a pending Entry, the private parts, or a
// share key.
func PublicRoutes(cm catalog.CatalogManager) []Route {
	return []Route{
		{Method: http.MethodGet, Path: "/entries/{slug}", Handler: OpenEntry(cm)},
		{Method: http.MethodPost, Path: "/page-views", Handler: RecordPageView(cm)},
	}
}

// EntryRoutes is the operator's Entry surface: the read that ignores
// visibility, the listing with its filters, upsert, delete and visibility.
//
// Mounted without a capability gate, this publishes every pending agency.
func EntryRoutes(cm catalog.CatalogManager) []Route {
	base := "/entries"
	return []Route{
		{Method: http.MethodGet, Path: base, Handler: ListEntries(cm)},
		{Method: http.MethodGet, Path: base + "/{slug}/admin", Handler: GetEntry(cm)},
		{Method: http.MethodPut, Path: base + "/{slug}", Handler: UpsertEntry(cm)},
		{Method: http.MethodDelete, Path: base + "/{slug}", Handler: DeleteEntry(cm)},
		{Method: http.MethodPost, Path: base + "/{slug}/visibility", Handler: SetVisibility(cm)},
		{Method: http.MethodGet, Path: base + "/{slug}/private", Handler: GetPrivate(cm)},
		{Method: http.MethodPut, Path: base + "/{slug}/private", Handler: PutPrivate(cm)},
		{Method: http.MethodGet, Path: base + "/{slug}/links", Handler: ListShareLinks(cm)},
		{Method: http.MethodPost, Path: base + "/{slug}/links", Handler: IssueShareLink(cm)},
		{Method: http.MethodPost, Path: "/links/{linkID}/retire", Handler: RetireShareLink(cm)},
	}
}

// SuggestionRoutes is the mining backlog.
func SuggestionRoutes(cm catalog.CatalogManager) []Route {
	base := "/suggestions"
	return []Route{
		{Method: http.MethodGet, Path: base, Handler: ListSuggestions(cm)},
		{Method: http.MethodPost, Path: base, Handler: FileSuggestion(cm)},
		{Method: http.MethodGet, Path: base + "/{suggestionID}", Handler: GetSuggestion(cm)},
		{Method: http.MethodPost, Path: base + "/{suggestionID}/decide", Handler: DecideSuggestion(cm)},
		{Method: http.MethodPost, Path: base + "/{suggestionID}/attempt", Handler: RecordSuggestionAttempt(cm)},
		{Method: http.MethodPost, Path: base + "/{suggestionID}/fulfil", Handler: FulfilSuggestion(cm)},
		{Method: http.MethodGet, Path: base + "/{suggestionID}/comments", Handler: ListSuggestionComments(cm)},
		{Method: http.MethodPost, Path: base + "/{suggestionID}/comments", Handler: CommentOnSuggestion(cm)},
		{Method: http.MethodGet, Path: base + "/{suggestionID}/matches", Handler: ListSuggestionMatches(cm)},
	}
}

// OperationsRoutes is the run reports, the settings and the page-view
// totals.
func OperationsRoutes(cm catalog.CatalogManager) []Route {
	return []Route{
		{Method: http.MethodGet, Path: "/runs", Handler: ListRuns(cm)},
		{Method: http.MethodPost, Path: "/runs", Handler: RecordRun(cm)},
		{Method: http.MethodGet, Path: "/settings/{key}", Handler: GetSetting(cm)},
		{Method: http.MethodPut, Path: "/settings/{key}", Handler: PutSetting(cm)},
		{Method: http.MethodGet, Path: "/page-views", Handler: ListPageViews(cm)},
	}
}

// Routes is every address this package answers. A process that wants one
// gate over the lot uses this; one that wants the anonymous reader gated
// differently from the operator mounts PublicRoutes separately from the
// other three.
func Routes(cm catalog.CatalogManager) []Route {
	out := PublicRoutes(cm)
	out = append(out, EntryRoutes(cm)...)
	out = append(out, SuggestionRoutes(cm)...)
	out = append(out, OperationsRoutes(cm)...)
	return out
}
