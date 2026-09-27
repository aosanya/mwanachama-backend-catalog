package routes

import (
	"fmt"
	"sync"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/httpwire"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

type Route = httpwire.Route

var operations = sync.OnceValues(func() (*dispatch.Spec, error) {
	return dispatch.Parse(catalog.Operations())
})

var sentinels = map[string]error{
	"ErrEntryNotFound":       catalog.ErrEntryNotFound,
	"ErrSuggestionNotFound":  catalog.ErrSuggestionNotFound,
	"ErrShareLinkNotFound":   catalog.ErrShareLinkNotFound,
	"ErrSettingNotFound":     catalog.ErrSettingNotFound,
	"ErrEntryExists":         catalog.ErrEntryExists,
	"ErrSlugImmutable":       catalog.ErrSlugImmutable,
	"ErrNotPrivate":          catalog.ErrNotPrivate,
	"ErrSuggestionDuplicate": catalog.ErrSuggestionDuplicate,
	"ErrSuggestionStatus":    catalog.ErrSuggestionStatus,
	"ErrInvalidEntry":        catalog.ErrInvalidEntry,
	"ErrInvalidPrivate":      catalog.ErrInvalidPrivate,
	"ErrInvalidSuggestion":   catalog.ErrInvalidSuggestion,
	"ErrAttemptNeedsComment": catalog.ErrAttemptNeedsComment,
	"ErrInvalidComment":      catalog.ErrInvalidComment,
	"ErrInvalidRun":          catalog.ErrInvalidRun,
	"ErrInvalidSetting":      catalog.ErrInvalidSetting,
	"ErrInvalidPageView":     catalog.ErrInvalidPageView,
}

// AnonymousActions is every operation a storefront reader may reach without
// presenting a caller. It is the allowlist Routes is split on, never the
// list of what is protected: an operation added to the spec and not named
// here arrives gated.
var AnonymousActions = []string{
	"catalog.entry.open",
	"catalog.page_view.record",
}

func Build(cm catalog.CatalogManager) ([]Route, error) { return BuildWith(cm, nil) }

func BuildWith(cm catalog.CatalogManager, authorize dispatch.Authorizer) ([]Route, error) {
	s, err := operations()
	if err != nil {
		return nil, err
	}
	return dispatch.Dispatch(s, dispatch.Deps{Manager: cm, Errors: sentinels, Authorize: authorize})
}

func Routes(cm catalog.CatalogManager) []Route { return RoutesWith(cm, nil) }

func RoutesWith(cm catalog.CatalogManager, authorize dispatch.Authorizer) []Route {
	out, err := BuildWith(cm, authorize)
	if err != nil {
		panic(fmt.Sprintf("catalog routes: %v", err))
	}
	return out
}

func Split(cm catalog.CatalogManager) dispatch.Split { return SplitWith(cm, nil) }

func SplitWith(cm catalog.CatalogManager, authorize dispatch.Authorizer) dispatch.Split {
	public := dispatch.Anonymous(Routes(cm), AnonymousActions...)
	gated := dispatch.Anonymous(RoutesWith(cm, authorize), AnonymousActions...)
	return dispatch.Split{Anonymous: public.Anonymous, Gated: gated.Gated}
}

func PublicRoutes(cm catalog.CatalogManager) []Route {
	return Split(cm).Anonymous
}

func OperatorRoutes(cm catalog.CatalogManager) []Route { return OperatorRoutesWith(cm, nil) }

func OperatorRoutesWith(cm catalog.CatalogManager, authorize dispatch.Authorizer) []Route {
	return SplitWith(cm, authorize).Gated
}
