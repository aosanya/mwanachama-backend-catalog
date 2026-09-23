package mwanachamacatalog_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

// domainWords are words that mean something in one domain and nothing in
// another. This module is expected to run a library and a course catalog as
// well as an agency storefront, so none of them may appear in an identifier
// or in a stored enum value here.
//
// They are allowed in comments, where they earn their place by illustrating
// what a neutral name stands for — "an agency's sector, a library's genre".
// "title" is deliberately absent: a suggestion has a title the way a form
// does, and the word belongs to no domain in particular. The list is for
// words that carry a domain with them, not for every word a domain uses.
var domainWords = []string{
	"agency", "agencies", "archetype", "sector", "mined",
	"isbn", "genre", "patron", "librar", "shelfmark",
}

// Identifiers that must never carry a domain word. This is the guard that
// stops the rule in doc.go becoming decoration: Entry.Sector and
// PageKind's "agency" value both existed once, and both would fail here.
func TestNoDomainWordsInIdentifiers(t *testing.T) {
	for _, dir := range []string{"models", "gormstore", "spec"} {
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", dir, err)
		}
		seen := map[string]bool{}
		for _, pkg := range pkgs {
			for path, file := range pkg.Files {
				ast.Inspect(file, func(n ast.Node) bool {
					id, ok := n.(*ast.Ident)
					if !ok {
						return true
					}
					w, bad := carriesDomainWord(id.Name)
					if !bad {
						return true
					}
					key := filepath.Base(path) + ":" + id.Name
					if seen[key] {
						return true
					}
					seen[key] = true
					t.Errorf("%s: identifier %q carries the domain word %q — it belongs in the document, not in this module (see doc.go)",
						filepath.Base(path), id.Name, w)
					return true
				})
			}
		}
	}
}

// A stored enum value outlives a rename, so a domain word in one is worse
// than a domain word in a field name.
func TestNoDomainWordsInStoredVocabularies(t *testing.T) {
	for _, v := range []string{
		string(catalogVisibilityPending), string(catalogVisibilityPublic), string(catalogVisibilityPrivate),
		string(linkActive), string(linkRetired),
		string(pageIndex), string(pageEntry),
		string(sugProposed), string(sugApproved), string(sugRejected), string(sugBuilt), string(sugStalled),
		string(srcRun), string(srcOwner), string(srcStray),
		string(authorRun), string(authorAdmin),
	} {
		if w, bad := carriesDomainWord(v); bad {
			t.Errorf("stored value %q carries the domain word %q — a value outlives a rename", v, w)
		}
	}
}

// Short aliases so the vocabulary test reads as a list of stored values.
const (
	catalogVisibilityPending = catalog.VisibilityPending
	catalogVisibilityPublic  = catalog.VisibilityPublic
	catalogVisibilityPrivate = catalog.VisibilityPrivate
	linkActive               = catalog.LinkStatusActive
	linkRetired              = catalog.LinkStatusRetired
	pageIndex                = catalog.PageKindIndex
	pageEntry                = catalog.PageKindEntry
	sugProposed              = catalog.SuggestionStatusProposed
	sugApproved              = catalog.SuggestionStatusApproved
	sugRejected              = catalog.SuggestionStatusRejected
	sugBuilt                 = catalog.SuggestionStatusBuilt
	sugStalled               = catalog.SuggestionStatusStalled
	srcRun                   = catalog.SuggestionSourceRun
	srcOwner                 = catalog.SuggestionSourceOwner
	srcStray                 = catalog.SuggestionSourceStray
	authorRun                = catalog.CommentAuthorRun
	authorAdmin              = catalog.CommentAuthorAdmin
)

func carriesDomainWord(s string) (string, bool) {
	low := strings.ToLower(s)
	for _, w := range domainWords {
		if strings.Contains(low, w) {
			return w, true
		}
	}
	return "", false
}
