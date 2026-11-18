package mwanachamacatalog_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

const testSenderSecret = "zoldhalom-test-secret"

type testClock struct{ at time.Time }

func (c *testClock) now() string { return c.at.UTC().Format(time.RFC3339) }

func (c *testClock) pass(days int) { c.at = c.at.AddDate(0, 0, days) }

func senderManager(t *testing.T) (catalog.CatalogManager, *testClock) {
	t.Helper()
	cm := wardManager(t)
	clock := &testClock{at: time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)}
	cm.UseClock(clock.now)
	cm.UseSenderSecret([]byte(testSenderSecret))
	return cm, clock
}

func wish(title string, who string) catalog.RaisedItem {
	return catalog.RaisedItem{Title: title, RaisedBy: who, Anonymous: true}
}

func TestASecondWishFromTheSamePersonInThePeriodIsRefused(t *testing.T) {
	cm, _ := senderManager(t)
	ctx := context.Background()
	if _, err := cm.RaiseItem(ctx, wish("Árok a Kossuth utcán", "r1")); err != nil {
		t.Fatal(err)
	}
	_, err := cm.RaiseItem(ctx, wish("Törött lámpa", "r1"))
	if !errors.Is(err, catalog.ErrWishAlreadySentThisPeriod) {
		t.Fatalf("a second wish in the period gave %v, want ErrWishAlreadySentThisPeriod", err)
	}
	if _, err := cm.RaiseItem(ctx, wish("Padok a téren", "r2")); err != nil {
		t.Fatalf("another person's wish in the same period gave %v, want it accepted", err)
	}
}

func TestTheSamePersonCanSendAgainWhenThePeriodRolls(t *testing.T) {
	cm, clock := senderManager(t)
	ctx := context.Background()
	if _, err := cm.RaiseItem(ctx, wish("Árok a Kossuth utcán", "r1")); err != nil {
		t.Fatal(err)
	}
	clock.pass(31)
	if _, err := cm.RaiseItem(ctx, wish("Törött lámpa", "r1")); err != nil {
		t.Fatalf("a wish after the period rolled gave %v, want it accepted", err)
	}
}

func TestShorteningThePeriodLetsThePersonSendAgain(t *testing.T) {
	cm, clock := senderManager(t)
	ctx := context.Background()
	if _, err := cm.RaiseItem(ctx, wish("Árok a Kossuth utcán", "r1")); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.PutSetting(ctx, catalog.SenderPeriodSetting, "7"); err != nil {
		t.Fatal(err)
	}
	clock.pass(8)
	if _, err := cm.RaiseItem(ctx, wish("Törött lámpa", "r1")); err != nil {
		t.Fatalf("a wish after the admin shortened the period gave %v, want it accepted", err)
	}
}

func TestAPeriodThatIsNotADayCountIsRefusedAsASetting(t *testing.T) {
	cm, _ := senderManager(t)
	ctx := context.Background()
	if _, err := cm.PutSetting(ctx, catalog.SenderPeriodSetting, "0"); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.RaiseItem(ctx, wish("Árok", "r1")); !errors.Is(err, catalog.ErrInvalidSetting) {
		t.Fatalf("a period of 0 days gave %v, want ErrInvalidSetting", err)
	}
}

func TestTheCouncilReadsNoNameAndCannotLinkTwoWishesOfOnePerson(t *testing.T) {
	cm, clock := senderManager(t)
	ctx := context.Background()
	first, err := cm.RaiseItem(ctx, wish("Árok a Kossuth utcán", "r1"))
	if err != nil {
		t.Fatal(err)
	}
	clock.pass(31)
	second, err := cm.RaiseItem(ctx, wish("Törött lámpa", "r1"))
	if err != nil {
		t.Fatal(err)
	}
	if first.SenderHash == "" || second.SenderHash == "" || first.SenderHash == second.SenderHash {
		t.Fatalf("stored fingerprints %q and %q, want two different ones", first.SenderHash, second.SenderHash)
	}
	for _, item := range []catalog.RaisedItem{first, second} {
		seen := item.AsSeenByOthers()
		if seen.RaisedBy != "" || seen.SenderHash != "" || seen.SenderPeriod != "" {
			t.Fatalf("the council sees %+v, want no name, fingerprint or period", seen)
		}
	}
}

func TestNoReadReturnsTheSecretOrTheFingerprint(t *testing.T) {
	cm, _ := senderManager(t)
	ctx := context.Background()
	item, err := cm.RaiseItem(ctx, wish("Árok", "r1"))
	if err != nil {
		t.Fatal(err)
	}
	mine, err := cm.ListRaisedItems(ctx, "r1")
	if err != nil || len(mine) != 1 {
		t.Fatalf("read %d items, %v, want 1", len(mine), err)
	}
	for _, v := range []any{item, mine} {
		body, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(body), testSenderSecret) || strings.Contains(string(body), item.SenderHash) {
			t.Fatalf("a read returned the secret or the fingerprint: %s", body)
		}
	}
}

func TestMeTooCountsOncePerPersonPerWish(t *testing.T) {
	cm, _ := senderManager(t)
	ctx := context.Background()
	for _, holder := range []string{"r2", "r2", "r3"} {
		if _, err := cm.Endorse(ctx, "wish-1", holder); err != nil {
			t.Fatal(err)
		}
	}
	n, err := cm.CountEndorsements(ctx, "wish-1")
	if err != nil || n != 2 {
		t.Fatalf("Me too counted %d, %v, want 2", n, err)
	}
}

func TestWithoutASecretNoFingerprintIsKeptAndNoLimitApplies(t *testing.T) {
	cm := wardManager(t)
	ctx := context.Background()
	first, err := cm.RaiseItem(ctx, wish("Árok", "r1"))
	if err != nil || first.SenderHash != "" {
		t.Fatalf("first wish gave %q, %v, want no fingerprint", first.SenderHash, err)
	}
	if _, err := cm.RaiseItem(ctx, wish("Lámpa", "r1")); err != nil {
		t.Fatalf("second wish without a secret gave %v, want it accepted", err)
	}
}

func TestStaffFilingIsNotLimitedByThePeriod(t *testing.T) {
	cm, _ := senderManager(t)
	ctx := context.Background()
	if _, err := cm.RaiseItem(ctx, wish("Árok", "r1")); err != nil {
		t.Fatal(err)
	}
	for _, title := range []string{"Lámpa", "Pad"} {
		filed, err := cm.FileRaisedItem(ctx, catalog.RaisedItem{Title: title, RaisedBy: "r1", FiledBy: "clerk1", Channel: "phone"})
		if err != nil {
			t.Fatalf("staff filing for r1 gave %v, want it accepted", err)
		}
		if filed.SenderHash != "" {
			t.Fatalf("staff filing kept fingerprint %q, want none", filed.SenderHash)
		}
	}
}
