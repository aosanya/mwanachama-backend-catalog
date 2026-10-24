package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"slices"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/routes"
	"github.com/aosanya/mwanachama-backend-shared/geo"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func raisedItemsMount(t *testing.T) (gated, anon http.Handler, cm catalog.CatalogManager) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	s, err := catalog.LoadSpec(filepath.Join("..", "spec", "examples", "agency.catalog.json"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cm, err = catalog.NewCatalogManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	gatedMux, anonMux := http.NewServeMux(), http.NewServeMux()
	for _, r := range routes.Routes(cm) {
		gatedMux.Handle(r.Method+" "+r.Path, r.Handler)
	}
	for _, r := range routes.PublicRoutes(cm) {
		anonMux.Handle(r.Method+" "+r.Path, r.Handler)
	}
	return gatedMux, anonMux, cm
}

func raiseAt(t *testing.T, cm catalog.CatalogManager, title, topic string, publishable bool, lon, lat float64) {
	t.Helper()
	p := geo.NewPoint(lon, lat)
	r, err := cm.RaiseItem(context.Background(), catalog.RaisedItem{
		Title: title, RaisedBy: "amina", Topic: topic, Publishable: publishable,
		Pin: &p, Address: "12 Riverside Road",
	})
	if err != nil {
		t.Fatalf("raise %s: %v", title, err)
	}
	if !publishable {
		return
	}
	b, err := cm.PutOnBoard(context.Background(), catalog.BoardItem{Title: title})
	if err != nil {
		t.Fatalf("board %s: %v", title, err)
	}
	if err := cm.GroupUnderBoardItem(context.Background(), r.ID, b.ID); err != nil {
		t.Fatalf("group %s: %v", title, err)
	}
}

func nearbyQuery(topic, at, within string) string {
	q := url.Values{}
	q.Set("topic", topic)
	q.Set("at", at)
	q.Set("within", within)
	return "/raised-items/nearby?" + q.Encode()
}

func TestNearbyRaisedItemsAnswersThroughTheDeclaredTableNearestFirstAndRounded(t *testing.T) {
	gated, _, cm := raisedItemsMount(t)
	raiseAt(t, cm, "Far lamp out", "streetlights", true, 36.8200, -1.2860)
	raiseAt(t, cm, "Near lamp out", "streetlights", true, 36.81712, -1.28634)
	raiseAt(t, cm, "Unpublishable lamp", "streetlights", false, 36.8171, -1.2863)
	raiseAt(t, cm, "Pothole", "roads", true, 36.8171, -1.2863)
	raiseAt(t, cm, "Lamp across town", "streetlights", true, 36.9000, -1.2000)

	rec := httptest.NewRecorder()
	gated.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, nearbyQuery("streetlights", "[36.8170,-1.2863]", "1000"), nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var got []catalog.RaisedItem
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode %s: %v", rec.Body, err)
	}
	var titles []string
	for _, g := range got {
		titles = append(titles, g.Title)
		if g.Address != "" {
			t.Errorf("%s: address %q reached a nearby read", g.Title, g.Address)
		}
		if g.Pin == nil || g.Pin.Point != geo.RoundForPublic(g.Pin.Point) {
			t.Errorf("%s: pin %v is not the rounded one", g.Title, g.Pin)
		}
	}
	if !slices.Equal(titles, []string{"Near lamp out", "Far lamp out"}) {
		t.Fatalf("got %v, want the two publishable streetlights nearest first", titles)
	}
}

func TestNearbyRaisedItemsRefusesAMissingTopicOrDistance(t *testing.T) {
	gated, _, _ := raisedItemsMount(t)
	for name, path := range map[string]string{
		"no topic":         nearbyQuery("", "[36.8,-1.28]", "500"),
		"zero distance":    nearbyQuery("streetlights", "[36.8,-1.28]", "0"),
		"point not a pair": nearbyQuery("streetlights", "36.8", "500"),
	} {
		rec := httptest.NewRecorder()
		gated.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400: %s", name, rec.Code, rec.Body)
		}
	}
}

func TestNearbyRaisedItemsIsNeverAnonymous(t *testing.T) {
	if slices.Contains(routes.AnonymousActions, "catalog.raised_item.list_nearby") {
		t.Fatal("the nearby read is on the anonymous allowlist")
	}
	_, anon, cm := raisedItemsMount(t)
	raiseAt(t, cm, "Near lamp out", "streetlights", true, 36.8171, -1.2863)
	rec := httptest.NewRecorder()
	anon.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, nearbyQuery("streetlights", "[36.8170,-1.2863]", "1000"), nil))
	if rec.Code == http.StatusOK {
		t.Fatalf("an anonymous reader reached the nearby read: %s", rec.Body)
	}
}
