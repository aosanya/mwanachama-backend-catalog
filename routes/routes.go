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
	"ErrEntryNotFound":          catalog.ErrEntryNotFound,
	"ErrSuggestionNotFound":     catalog.ErrSuggestionNotFound,
	"ErrShareLinkNotFound":      catalog.ErrShareLinkNotFound,
	"ErrSettingNotFound":        catalog.ErrSettingNotFound,
	"ErrEntryExists":            catalog.ErrEntryExists,
	"ErrSlugImmutable":          catalog.ErrSlugImmutable,
	"ErrNotPrivate":             catalog.ErrNotPrivate,
	"ErrSuggestionDuplicate":    catalog.ErrSuggestionDuplicate,
	"ErrSuggestionStatus":       catalog.ErrSuggestionStatus,
	"ErrInvalidEntry":           catalog.ErrInvalidEntry,
	"ErrEntryKind":              catalog.ErrEntryKind,
	"ErrInvalidPrivate":         catalog.ErrInvalidPrivate,
	"ErrInvalidSuggestion":      catalog.ErrInvalidSuggestion,
	"ErrAttemptNeedsComment":    catalog.ErrAttemptNeedsComment,
	"ErrInvalidComment":         catalog.ErrInvalidComment,
	"ErrInvalidRun":             catalog.ErrInvalidRun,
	"ErrInvalidSetting":         catalog.ErrInvalidSetting,
	"ErrInvalidPageView":        catalog.ErrInvalidPageView,
	"ErrInvalidRecord":          catalog.ErrInvalidRecord,
	"ErrRecordsNotKept":         catalog.ErrRecordsNotKept,
	"ErrRecordHolderSpoke":      catalog.ErrRecordHolderSpoke,
	"ErrInvalidRaisedItem":      catalog.ErrInvalidRaisedItem,
	"ErrRaisedItemsNotTaken":    catalog.ErrRaisedItemsNotTaken,
	"ErrInvalidServiceTerms":    catalog.ErrInvalidServiceTerms,
	"ErrBoardItemNotFound":      catalog.ErrBoardItemNotFound,
	"ErrBoardItemClosed":        catalog.ErrBoardItemClosed,
	"ErrInvalidBoardItem":       catalog.ErrInvalidBoardItem,
	"ErrBoardNotKept":           catalog.ErrBoardNotKept,
	"ErrRaisedItemNotFound":     catalog.ErrRaisedItemNotFound,
	"ErrInvalidReply":           catalog.ErrInvalidReply,
	"ErrRepliesNotSent":         catalog.ErrRepliesNotSent,
	"ErrInvalidScopeFigure":     catalog.ErrInvalidScopeFigure,
	"ErrFiguresNotKept":         catalog.ErrFiguresNotKept,
	"ErrScopeFigureNotFound":    catalog.ErrScopeFigureNotFound,
	"ErrInvalidEndorsement":     catalog.ErrInvalidEndorsement,
	"ErrEndorsementsNotCounted": catalog.ErrEndorsementsNotCounted,

	"ErrInvalidRemoval":           catalog.ErrInvalidRemoval,
	"ErrRemovalsNotMade":          catalog.ErrRemovalsNotMade,
	"ErrRaisedItemAlreadyRemoved": catalog.ErrRaisedItemAlreadyRemoved,
}

// AnonymousActions is every operation a storefront reader may reach without
// presenting a caller. It is the allowlist Routes is split on, never the
// list of what is protected: an operation added to the spec and not named
// here arrives gated.
var AnonymousActions = []string{
	"catalog.entry.open",
	"catalog.page_view.record",
	"catalog.board_item.open_shared",
	"catalog.raised_item.open_followed",
}

type Mount struct {
	Authorize dispatch.Authorizer
	Caller    dispatch.Caller
	Anonymous []string
}

func (m Mount) anonymousActions() []string {
	if m.Anonymous == nil {
		return AnonymousActions
	}
	return m.Anonymous
}

func Build(cm catalog.CatalogManager) ([]Route, error) { return BuildWith(cm, nil) }

func BuildWith(cm catalog.CatalogManager, authorize dispatch.Authorizer) ([]Route, error) {
	return BuildFor(cm, Mount{Authorize: authorize})
}

func BuildFor(cm catalog.CatalogManager, m Mount) ([]Route, error) {
	s, err := operations()
	if err != nil {
		return nil, err
	}
	return dispatch.Dispatch(s, dispatch.Deps{
		Manager: cm, Errors: sentinels, Authorize: m.Authorize, Caller: m.Caller,
	})
}

func Routes(cm catalog.CatalogManager) []Route { return RoutesWith(cm, nil) }

func RoutesWith(cm catalog.CatalogManager, authorize dispatch.Authorizer) []Route {
	return RoutesFor(cm, Mount{Authorize: authorize})
}

func RoutesFor(cm catalog.CatalogManager, m Mount) []Route {
	out, err := BuildFor(cm, m)
	if err != nil {
		panic(fmt.Sprintf("catalog routes: %v", err))
	}
	return out
}

func Split(cm catalog.CatalogManager) dispatch.Split { return SplitWith(cm, nil) }

func SplitWith(cm catalog.CatalogManager, authorize dispatch.Authorizer) dispatch.Split {
	return SplitFor(cm, Mount{Authorize: authorize})
}

func SplitFor(cm catalog.CatalogManager, m Mount) dispatch.Split {
	actions := m.anonymousActions()
	ungated := Routes(cm)
	public := dispatch.Anonymous(ungated, actions...)
	if missing := public.Unmatched(ungated, actions); len(missing) > 0 {
		panic(fmt.Sprintf("catalog routes: mount opens unknown action(s) to anonymous callers: %v", missing))
	}
	gated := dispatch.Anonymous(RoutesFor(cm, m), actions...)
	return dispatch.Split{Anonymous: public.Anonymous, Gated: gated.Gated}
}

func PublicRoutes(cm catalog.CatalogManager) []Route {
	return Split(cm).Anonymous
}

func PublicRoutesFor(cm catalog.CatalogManager, m Mount) []Route {
	return SplitFor(cm, m).Anonymous
}

func OperatorRoutes(cm catalog.CatalogManager) []Route { return OperatorRoutesWith(cm, nil) }

func OperatorRoutesWith(cm catalog.CatalogManager, authorize dispatch.Authorizer) []Route {
	return OperatorRoutesFor(cm, Mount{Authorize: authorize})
}

func OperatorRoutesFor(cm catalog.CatalogManager, m Mount) []Route {
	return SplitFor(cm, m).Gated
}
