package routes_test

import (
	"net/http"
	"net/url"
	"testing"
)

func TestCAT7_MalformedDocumentPathAnswersBadRequest(t *testing.T) {
	operator, anon, _ := municipalityWard(t)
	malformed := "docs=" + url.QueryEscape(`{"path":"a'; drop table x;--","value":"v"}`)

	for name, h := range map[string]http.Handler{"operator": operator, "anonymous": anon} {
		path := "/entries?"
		if name == "anonymous" {
			path = "/public/entries?"
		}
		rec := do(t, h, http.MethodGet, path+malformed, "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s GET %s answered %d %s; want 400", name, path, rec.Code, rec.Body.String())
		}
	}
}
