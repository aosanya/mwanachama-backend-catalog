package mwanachamacatalog_test

import (
	"encoding/hex"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestShareKeyCarriesAtLeast128BitsThatVaryInEveryPosition(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)

	const issued = 400
	const minBytes = 16
	keys := make([]string, 0, issued)
	seen := map[string]bool{}
	for i := 0; i < issued; i++ {
		_, raw, err := cm.IssueShareLink(ctx, "private-one", "", "owner")
		if err != nil {
			t.Fatalf("precondition: issue %d failed: %v", i, err)
		}
		if _, err := cm.OpenEntry(ctx, "private-one", raw); err != nil {
			t.Fatalf("precondition: key %d did not open its own entry: %v", i, err)
		}
		decoded, err := hex.DecodeString(raw)
		if err != nil {
			t.Fatalf("share key %q is not hex: %v", raw, err)
		}
		if len(decoded) < minBytes {
			t.Fatalf("share key %q carries %d bytes; want at least %d so a private entry's key cannot be guessed by enumeration", raw, len(decoded), minBytes)
		}
		if seen[raw] {
			t.Fatalf("share key %q was issued twice in %d issues", raw, issued)
		}
		seen[raw] = true
		keys = append(keys, raw)
	}

	for pos := 0; pos < len(keys[0]); pos++ {
		values := map[byte]bool{}
		for _, k := range keys {
			if pos < len(k) {
				values[k[pos]] = true
			}
		}
		if len(values) < 8 {
			t.Errorf("position %d of %d share keys took only %d distinct hex digits; a key drawn from a counter, a clock or a short seed leaves positions fixed, and every position should vary", pos, issued, len(values))
		}
	}
}
