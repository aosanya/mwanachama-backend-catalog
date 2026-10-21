package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestAHolderEndorsesAnItemOnceHoweverOftenTheyAsk(t *testing.T) {
	cm, ctx := newManager(t)
	for _, holder := range []string{"amina", "amina", "baraka"} {
		if _, err := cm.Endorse(ctx, "item-1", holder); err != nil {
			t.Fatal(err)
		}
	}
	n, err := cm.Endorse(ctx, "item-1", "amina")
	if err != nil || n != 2 {
		t.Fatalf("endorsements = %d (%v), want 2", n, err)
	}
	other, err := cm.CountEndorsements(ctx, "item-2")
	if err != nil || other != 0 {
		t.Fatalf("an unendorsed item counts %d (%v), want 0", other, err)
	}
}

func TestAnEndorsementNamesBothItsItemAndItsHolder(t *testing.T) {
	cm, ctx := newManager(t)
	if _, err := cm.Endorse(ctx, "", "amina"); !errors.Is(err, catalog.ErrInvalidEndorsement) {
		t.Fatalf("no item: %v", err)
	}
	if _, err := cm.Endorse(ctx, "item-1", ""); !errors.Is(err, catalog.ErrInvalidEndorsement) {
		t.Fatalf("no holder: %v", err)
	}
}
