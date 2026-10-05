package routes_test

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestNoReadAnswerCarriesAShareKeyOrItsStoredHashUnderAnyName(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "alpha", catalog.VisibilityPrivate)

	first, _ := issueKey(t, h, "alpha")
	second, _ := issueKey(t, h, "alpha")
	if first == second {
		t.Fatalf("precondition: two issues answered the same key %q", first)
	}
	if rec := do(t, h, http.MethodGet, "/entries/alpha?k="+first, ""); rec.Code != http.StatusOK {
		t.Fatalf("precondition: the first key should open alpha, got %d %s", rec.Code, rec.Body.String())
	}

	reads := []string{"/entries/alpha/links", "/entries/alpha?k=" + second, "/entries", "/entries/alpha/private"}
	for _, key := range []string{first, second} {
		sum := sha256.Sum256([]byte(key))
		hash := hex.EncodeToString(sum[:])
		for _, path := range reads {
			body := strings.ToLower(do(t, h, http.MethodGet, path, "").Body.String())
			if strings.Contains(body, hash) {
				t.Errorf("GET %s carries the stored SHA-256 of a share key (%s…); a stored key hash must never reach a read answer under any field name", path, hash[:12])
			}
			if strings.Contains(body, strings.ToLower(key)) {
				t.Errorf("GET %s carries a raw share key (%s…); the raw key is returned once, at issue, and never again", path, key[:12])
			}
		}
	}
}
