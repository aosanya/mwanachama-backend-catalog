package routes_test

import (
	"slices"
	"testing"

	"github.com/aosanya/mwanachama-backend-catalog/routes"
)

func TestOpeningAFollowedRaisedItemIsAnonymousAndIssuingAKeyIsNot(t *testing.T) {
	if !slices.Contains(routes.AnonymousActions, "catalog.raised_item.open_followed") {
		t.Error("open_followed is not anonymous")
	}
	if slices.Contains(routes.AnonymousActions, "catalog.raised_item.issue_follow_key") {
		t.Error("issue_follow_key is anonymous")
	}
}
