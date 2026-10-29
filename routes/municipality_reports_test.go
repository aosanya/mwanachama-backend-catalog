package routes_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/routes"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

const municipalityReportsSpec = "municipality.reports.catalog.json"

var municipalityAnonymous = []string{
	"catalog.entry.open",
	"catalog.page_view.record",
	"catalog.suggestion.file",
	"catalog.entry.list_public",
}

func municipalityWard(t *testing.T) (operator http.Handler, anon http.Handler, cm catalog.CatalogManager) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	s, err := catalog.LoadSpec(filepath.Join("testdata", municipalityReportsSpec))
	if err != nil {
		t.Fatalf("load %s: %v", municipalityReportsSpec, err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cm, err = catalog.NewCatalogManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	opMux := http.NewServeMux()
	for _, r := range routes.Routes(cm) {
		opMux.Handle(r.Method+" "+r.Path, r.Handler)
	}
	anonMux := http.NewServeMux()
	for _, r := range routes.PublicRoutesFor(cm, routes.Mount{Anonymous: municipalityAnonymous}) {
		anonMux.Handle(r.Method+" "+r.Path, r.Handler)
	}
	return opMux, anonMux, cm
}

func wardReportBody(slug, ward, theme, channel string) string {
	proposal := fmt.Sprintf(
		`{"_report":{"ward":%q,"theme":%q,"channel":%q,"taken_by":"outreach-3","answer_due":"2026-10-15","place":"Ngong Road / Mbagathi"}}`,
		ward, theme, channel)
	return fmt.Sprintf(`{"slug":%q,"title":"Blocked drain at Ngong Road junction","why":"water standing after rain","proposal":%s}`,
		slug, strconv.Quote(proposal))
}

func boardItemBody(count int) string {
	doc := fmt.Sprintf(`{"_item":{"ward":"Kibera North","theme":"waste","state":"looking","raised_by_count":%d}}`, count)
	return fmt.Sprintf(`{"name":"Blocked drain at Ngong Road junction","summary":"Reported by residents of Kibera North","doc":%s}`,
		strconv.Quote(doc))
}

func fileArrival(t *testing.T, h http.Handler, slug, channel string) string {
	t.Helper()
	rec := do(t, h, "POST", "/suggestions", wardReportBody(slug, "Kibera North", "waste", channel))
	if rec.Code != http.StatusCreated {
		t.Fatalf("precondition: filing arrival %q must be 201, got %d %s", slug, rec.Code, rec.Body.String())
	}
	var v struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode arrival: %v", err)
	}
	if v.Status != "proposed" {
		t.Fatalf("precondition: a fresh arrival must be proposed, got %q", v.Status)
	}
	return v.ID
}

func TestMUNI1_TheReportsSpecTheTemplateShipsCanBeMounted(t *testing.T) {
	src := filepath.Join("testdata", municipalityReportsSpec)
	if _, err := catalog.LoadSpec(src); err != nil {
		t.Fatalf("MUNI1 open bug: the municipality template's reports module spec does not load, "+
			"so the mount the template describes cannot be created at all: %v", err)
	}
}

func TestMUNI2_AResidentCanFileAReportAndReadTheBoard(t *testing.T) {
	op, anon, cm := municipalityWard(t)

	var reachable []string
	for _, r := range routes.PublicRoutesFor(cm, routes.Mount{Anonymous: municipalityAnonymous}) {
		reachable = append(reachable, r.Method+" "+r.Path)
	}

	filed := do(t, anon, "POST", "/suggestions", wardReportBody("anon-report", "Kibera North", "waste", "counter"))
	if filed.Code != http.StatusCreated {
		t.Fatalf("MUNI2 open bug: a resident cannot file a report — POST /suggestions answered %d %s. The template "+
			"declares report.written_by as \"anyone, through any channel\". The anonymous surface is %v",
			filed.Code, strings.TrimSpace(filed.Body.String()), reachable)
	}

	if rec := do(t, op, "PUT", "/entries/ngong-road-junction", boardItemBody(1)); rec.Code != http.StatusOK {
		t.Fatalf("precondition: opening the board item must be 200, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, op, "PUT", "/entries/names-a-person", boardItemBody(1)); rec.Code != http.StatusOK {
		t.Fatalf("precondition: opening the second board item must be 200, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, op, "POST", "/entries/ngong-road-junction/visibility", `{"visibility":"public"}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: publishing must be 200, got %d %s", rec.Code, rec.Body.String())
	}

	board := do(t, anon, "GET", "/public/entries", "")
	if board.Code != http.StatusOK {
		t.Fatalf("MUNI2 open bug: the public board cannot be read — GET /public/entries answered %d %s. The "+
			"template's visibility_use declares public as \"the board itself\"",
			board.Code, strings.TrimSpace(board.Body.String()))
	}
	var cards []struct {
		Slug       string `json:"slug"`
		Visibility string `json:"visibility"`
	}
	if err := json.Unmarshal(board.Body.Bytes(), &cards); err != nil {
		t.Fatalf("decode board: %v (%s)", err, board.Body.String())
	}
	if len(cards) != 1 || cards[0].Slug != "ngong-road-junction" {
		t.Errorf("MUNI2 open bug: the public board returned %d card(s) %+v, want only the published one. An "+
			"unmoderated item on the board is the moderation failing open", len(cards), cards)
	}
	for _, c := range cards {
		if c.Visibility != "public" {
			t.Errorf("MUNI2 open bug: the public board carried a %q item (%s). pending is \"an arrival nobody has "+
				"moderated yet\" and private \"never appears on the board\"", c.Visibility, c.Slug)
		}
	}

	widened := do(t, anon, "GET", "/public/entries?visibility=pending", "")
	if strings.Contains(widened.Body.String(), "names-a-person") {
		t.Errorf("MUNI2 open bug: an anonymous caller widened the public board to pending by asking — %s. "+
			"The public listing must force visibility and ignore a caller that asks for more",
			widened.Body.String())
	}
}

func TestMUNI3_TheBoardCountIsHowManyArrivalsAreAttached(t *testing.T) {
	op, _, _ := municipalityWard(t)

	var ids []string
	for i, channel := range []string{"counter", "phone", "councillor"} {
		ids = append(ids, fileArrival(t, op, fmt.Sprintf("junction-report-%d", i), channel))
	}
	if rec := do(t, op, "PUT", "/entries/ngong-road-junction", boardItemBody(len(ids))); rec.Code != http.StatusOK {
		t.Fatalf("precondition: opening the board item must be 200, got %d %s", rec.Code, rec.Body.String())
	}
	for _, id := range ids {
		rec := do(t, op, "POST", "/suggestions/"+id+"/matches", `{"slug":"ngong-road-junction","note":"same junction"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("precondition: attaching %s must be 201, got %d %s", id, rec.Code, rec.Body.String())
		}
	}

	listed := do(t, op, "GET", "/entries/ngong-road-junction/matches", "")
	if listed.Code != http.StatusOK {
		t.Fatalf("MUNI3 open bug: an entry cannot report the arrivals it was assembled from — "+
			"GET /entries/{slug}/matches answered %d %s. Without it the board's \"raised by N people\" is a number "+
			"an operator typed and nothing reconciles",
			listed.Code, strings.TrimSpace(listed.Body.String()))
	}
	var attached []struct {
		SuggestionID string `json:"suggestion_id"`
		Slug         string `json:"slug"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &attached); err != nil {
		t.Fatalf("decode matches: %v (%s)", err, listed.Body.String())
	}
	if len(attached) != len(ids) {
		t.Errorf("MUNI3 open bug: the board item reports %d arrivals attached, want %d. The count a council acts on "+
			"must be how many people are attached to the item, which is the module's stated reason for keeping every "+
			"arrival as its own row",
			len(attached), len(ids))
	}
	seen := map[string]bool{}
	for _, a := range attached {
		seen[a.SuggestionID] = true
	}
	for _, id := range ids {
		if !seen[id] {
			t.Errorf("MUNI3 open bug: arrival %s was attached but is not among the item's arrivals — a merge that "+
				"drops a raiser destroys the number the board is ranked on", id)
		}
	}
}

func TestMUNI4_AttachingAnArrivalDoesNotCloseIt(t *testing.T) {
	op, _, _ := municipalityWard(t)

	id := fileArrival(t, op, "junction-report-0", "counter")
	if rec := do(t, op, "PUT", "/entries/ngong-road-junction", boardItemBody(1)); rec.Code != http.StatusOK {
		t.Fatalf("precondition: opening the board item must be 200, got %d %s", rec.Code, rec.Body.String())
	}

	attached := do(t, op, "POST", "/suggestions/"+id+"/matches", `{"slug":"ngong-road-junction","note":"same junction"}`)
	if attached.Code != http.StatusCreated {
		t.Fatalf("MUNI4 open bug: a proposed arrival could not be attached to a board item — answered %d %s. "+
			"The template has the moderator assemble a board item from the arrivals attached to it, which is "+
			"attach-then-decide",
			attached.Code, strings.TrimSpace(attached.Body.String()))
	}

	read := do(t, op, "GET", "/suggestions/"+id, "")
	var v struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(read.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode arrival: %v", err)
	}
	if v.Status != "proposed" {
		t.Errorf("MUNI4 open bug: attaching an arrival to a board item left it %q, want \"proposed\". The ladder "+
			"reads \"built when it is done\" and the drain has not been touched — the moderator only grouped "+
			"reports of it. An arrival marked done drops off the overdue list RU-7 promises",
			v.Status)
	}

	if rec := do(t, op, "POST", "/suggestions/"+id+"/decide", `{"status":"approved"}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: approving must be 200, got %d %s", rec.Code, rec.Body.String())
	}
	done := do(t, op, "POST", "/suggestions/"+id+"/fulfil", `{"slug":"ngong-road-junction","note":"cleared"}`)
	if done.Code != http.StatusOK {
		t.Fatalf("precondition: fulfilling must be 200, got %d %s", done.Code, done.Body.String())
	}
	if err := json.Unmarshal(done.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode arrival: %v", err)
	}
	if v.Status != "built" {
		t.Errorf("MUNI4: fulfilling an attached arrival left it %q, want \"built\" — completing must still be "+
			"reachable once the work is actually done", v.Status)
	}
}

func TestMUNI5_AnArrivalRecordsTheChannelItCameOn(t *testing.T) {
	op, _, _ := municipalityWard(t)

	id := fileArrival(t, op, "junction-report-0", "market-stall")
	read := do(t, op, "GET", "/suggestions/"+id, "")
	if read.Code != http.StatusOK {
		t.Fatalf("precondition: reading the arrival must be 200, got %d %s", read.Code, read.Body.String())
	}
	var v struct {
		Source   string `json:"source"`
		Proposal string `json:"proposal"`
	}
	if err := json.Unmarshal(read.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode arrival: %v", err)
	}

	var proposal struct {
		Report struct {
			Channel string `json:"channel"`
			Ward    string `json:"ward"`
		} `json:"_report"`
	}
	if err := json.Unmarshal([]byte(v.Proposal), &proposal); err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	if proposal.Report.Channel != "market-stall" {
		t.Errorf("MUNI5 open bug: the channel an arrival came on did not survive the round trip — "+
			"proposal._report.channel read %q, want \"market-stall\". That path is what the template's own channel "+
			"index reads, and a board that only ever fills from one channel is hearing the same few people a "+
			"different way",
			proposal.Report.Channel)
	}
	if proposal.Report.Ward != "Kibera North" {
		t.Errorf("MUNI5 open bug: the ward did not survive the round trip — proposal._report.ward read %q", proposal.Report.Ward)
	}

	inSource := do(t, op, "POST", "/suggestions",
		`{"slug":"channel-in-source","title":"filed at the counter","source":"counter","proposal":`+
			strconv.Quote(`{"_report":{"ward":"Kibera North"}}`)+`}`)
	if inSource.Code == http.StatusCreated {
		t.Errorf("MUNI5: source accepted %q. source is the module's own enum of run/owner/stray and must not "+
			"become a second place a domain puts its channel", "counter")
	}
}

func TestMUNI6_ANoteSaysWhoAnsweredIt(t *testing.T) {
	op, _, _ := municipalityWard(t)

	id := fileArrival(t, op, "junction-report-0", "counter")
	answered := do(t, op, "POST", "/suggestions/"+id+"/comments",
		`{"author":"councillor-kibera-north","body":"The service company has been asked to clear it this week."}`)
	if answered.Code != http.StatusCreated {
		t.Errorf("MUNI6 open bug: a note cannot name who wrote it — author \"councillor-kibera-north\" answered "+
			"%d %s. suggestion_comment.author is a blueprint enum of run/admin, and a domain may not restate a "+
			"value set, so the template's report_note — declared to carry \"the councillor's answer, the moderator's "+
			"reason for a removal, the service company's update\" — cannot distinguish the three. A resident is told "+
			"something by \"admin\"",
			answered.Code, strings.TrimSpace(answered.Body.String()))
	}
}

func TestMUNI7_ThePrivateSideHoldsWhatTheTemplateSaysItHolds(t *testing.T) {
	op, anon, _ := municipalityWard(t)

	if rec := do(t, op, "PUT", "/entries/ngong-road-junction", boardItemBody(1)); rec.Code != http.StatusOK {
		t.Fatalf("precondition: opening the board item must be 200, got %d %s", rec.Code, rec.Body.String())
	}
	private := strconv.Quote(`{"partner_reference":"SC-9912","match_note":"grouped from three arrivals","identifying":"raiser is the councillors neighbour"}`)
	written := do(t, op, "PUT", "/entries/ngong-road-junction/private", `{"doc":`+private+`}`)
	if written.Code != http.StatusOK {
		t.Fatalf("MUNI7 open bug: the private side of a board item has nowhere to hold what the template declares "+
			"it holds — PUT /entries/{slug}/private answered %d %s. board_item_private is declared to carry \"how it "+
			"was matched, the operational reference the service company gave it, and any note that would identify "+
			"somebody\", and the with_partner index reads doc._item.partner_reference",
			written.Code, strings.TrimSpace(written.Body.String()))
	}

	read := do(t, op, "GET", "/entries/ngong-road-junction/private", "")
	if !strings.Contains(read.Body.String(), "SC-9912") {
		t.Errorf("MUNI7 open bug: the partner reference did not come back from the operator's private read: %d %s",
			read.Code, read.Body.String())
	}

	if rec := do(t, op, "POST", "/entries/ngong-road-junction/visibility", `{"visibility":"public"}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: publishing must be 200, got %d %s", rec.Code, rec.Body.String())
	}
	board := do(t, anon, "GET", "/entries/ngong-road-junction", "")
	if strings.Contains(board.Body.String(), "SC-9912") || strings.Contains(board.Body.String(), "neighbour") {
		t.Errorf("MUNI7 open bug: the private side reached the public board — an anonymous open of a public board "+
			"item returned %s. The template's visibility_use puts anything naming a person on the private side, "+
			"which never appears on the board",
			board.Body.String())
	}
}

func TestMUNI8_APendingBoardItemIsNotOnThePublicBoard(t *testing.T) {
	op, anon, _ := municipalityWard(t)

	if rec := do(t, op, "PUT", "/entries/names-a-person", boardItemBody(1)); rec.Code != http.StatusOK {
		t.Fatalf("precondition: opening the board item must be 200, got %d %s", rec.Code, rec.Body.String())
	}
	opened := do(t, anon, "GET", "/entries/names-a-person", "")
	if opened.Code != http.StatusNotFound {
		t.Errorf("MUNI8 open bug: an anonymous reader opened an unmoderated board item: %d %s. The template's "+
			"visibility_use declares pending as \"an arrival nobody has moderated yet\" and private as what \"never "+
			"appears on the board\"",
			opened.Code, strings.TrimSpace(opened.Body.String()))
	}
}

func TestMUNI9_AQuarterlySilenceSweepAndTheBoardClockRecord(t *testing.T) {
	op, _, _ := municipalityWard(t)

	clock := do(t, op, "PUT", "/settings/board_clock",
		`{"value":"{\"acknowledge_days\":5,\"answer_days\":10,\"quarter\":\"2026-Q4\"}"}`)
	if clock.Code != http.StatusOK {
		t.Errorf("MUNI9: the board clock would not store: %d %s", clock.Code, strings.TrimSpace(clock.Body.String()))
	}
	sweep := do(t, op, "POST", "/runs",
		`{"run_ref":"silence-2026-Q4","templates_filed":3,"summary":"Kibera North 3 arrivals; Kibera South 0; Laini Saba 0"}`)
	if sweep.Code != http.StatusCreated {
		t.Errorf("MUNI9: the quarterly silence sweep would not record: %d %s", sweep.Code, strings.TrimSpace(sweep.Body.String()))
	}
	listed := do(t, op, "GET", "/runs", "")
	if !strings.Contains(listed.Body.String(), "Laini Saba 0") {
		t.Errorf("MUNI9: a recorded sweep did not come back from GET /runs: %d %s", listed.Code, listed.Body.String())
	}
}

func TestMUNI10_AKeyOpensOneBoardItemForARaiserWithNoAccount(t *testing.T) {
	op, anon, _ := municipalityWard(t)

	if rec := do(t, op, "PUT", "/entries/ngong-road-junction", boardItemBody(1)); rec.Code != http.StatusOK {
		t.Fatalf("precondition: opening the board item must be 200, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, op, "POST", "/entries/ngong-road-junction/visibility", `{"visibility":"private"}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: making it private must be 200, got %d %s", rec.Code, rec.Body.String())
	}
	issued := do(t, op, "POST", "/entries/ngong-road-junction/links", `{"label":"the raiser who gave no contact"}`)
	if issued.Code != http.StatusCreated {
		t.Fatalf("precondition: issuing a key must be 201, got %d %s", issued.Code, issued.Body.String())
	}
	var v struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(issued.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode key: %v", err)
	}
	if v.Key == "" {
		t.Fatalf("precondition: the raw key is handed over once, got none in %s", issued.Body.String())
	}

	withKey := do(t, anon, "GET", "/entries/ngong-road-junction?k="+v.Key, "")
	if withKey.Code != http.StatusOK {
		t.Errorf("MUNI10 open bug: the key issued with the acknowledgement did not open the board item for a "+
			"reader with no account: %d %s. The template's O-11 no-account promise rests on this",
			withKey.Code, strings.TrimSpace(withKey.Body.String()))
	}
	withWrongKey := do(t, anon, "GET", "/entries/ngong-road-junction?k=not-the-key", "")
	if withWrongKey.Code != http.StatusNotFound {
		t.Errorf("MUNI10 open bug: a wrong key answered %d, not 404 — a private board item must answer a bad key "+
			"exactly as a missing one", withWrongKey.Code)
	}
}
