package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCAT27_CommentAuthorNamesWhoWroteIt(t *testing.T) {
	cm, ctx := newManager(t)
	s := file(t, cm, ctx, "cat27-probe")

	for _, author := range []catalog.CommentAuthor{catalog.CommentAuthorRun, catalog.CommentAuthorAdmin, "councillor-kibera-north"} {
		c, err := cm.CommentOnSuggestion(ctx, s.ID, author, "a note")
		if err != nil {
			t.Fatalf("author %q refused: %v", author, err)
		}
		if c.Author != author {
			t.Errorf("author %q stored as %q", author, c.Author)
		}
	}

	read, err := cm.ListSuggestionComments(ctx, s.ID)
	if err != nil || len(read) != 3 {
		t.Fatalf("ListSuggestionComments = %d rows, %v; want 3", len(read), err)
	}
}

func TestCAT27_CommentAuthorMustBeASlug(t *testing.T) {
	cm, ctx := newManager(t)
	s := file(t, cm, ctx, "cat27-shape")

	for _, author := range []catalog.CommentAuthor{"Councillor Kibera", "x/../y"} {
		if _, err := cm.CommentOnSuggestion(ctx, s.ID, author, "a note"); !errors.Is(err, catalog.ErrInvalidComment) {
			t.Errorf("author %q answered %v, want ErrInvalidComment", author, err)
		}
	}
}
