package mwanachamacatalog_test

import (
	"errors"
	"reflect"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestTheRaiserAttachesFilesAndARepeatAttachesNothingNew(t *testing.T) {
	cm, ctx := newManager(t)
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Broken bench", RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := cm.AttachFiles(ctx, r.ID, "amina", []string{"file-1", "file-2"})
	if err != nil || len(got) != 2 {
		t.Fatalf("attached = %+v (%v)", got, err)
	}
	if got[0].AttachedBy != "amina" || got[0].AttachedAt == "" || got[0].ApprovedForPublishing() {
		t.Errorf("attachment = %+v", got[0])
	}
	again, err := cm.AttachFiles(ctx, r.ID, "amina", []string{"file-2", "file-3"})
	if err != nil || len(again) != 3 {
		t.Fatalf("after repeat = %+v (%v)", again, err)
	}
	if _, err := cm.AttachFiles(ctx, r.ID, "baraka", []string{"file-9"}); !errors.Is(err, catalog.ErrRaisedItemNotFound) {
		t.Errorf("someone else attaching = %v", err)
	}
	if _, err := cm.AttachFiles(ctx, "nobody", "amina", []string{"file-9"}); !errors.Is(err, catalog.ErrRaisedItemNotFound) {
		t.Errorf("unknown raised item = %v", err)
	}
	if _, err := cm.AttachFiles(ctx, r.ID, "amina", []string{""}); !errors.Is(err, catalog.ErrInvalidAttachment) {
		t.Errorf("empty file id = %v", err)
	}
	if _, err := cm.AttachFiles(ctx, r.ID, "amina", nil); !errors.Is(err, catalog.ErrInvalidAttachment) {
		t.Errorf("no file ids = %v", err)
	}
}

func TestOnlyAFileOnAPublishableItemIsApprovedForPublishing(t *testing.T) {
	cm, ctx := newManager(t)
	private, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "My own complaint", RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.AttachFiles(ctx, private.ID, "amina", []string{"file-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.ApproveFileForPublishing(ctx, private.ID, "file-1", "clerk-1"); !errors.Is(err, catalog.ErrRaisedItemNotPublishable) {
		t.Errorf("approving on an item that is not publishable = %v", err)
	}
	public, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Pothole", Publishable: true, RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.AttachFiles(ctx, public.ID, "amina", []string{"file-2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.ApproveFileForPublishing(ctx, public.ID, "file-missing", "clerk-1"); !errors.Is(err, catalog.ErrAttachmentNotFound) {
		t.Errorf("approving a file never attached = %v", err)
	}
	if _, err := cm.ApproveFileForPublishing(ctx, public.ID, "file-2", ""); !errors.Is(err, catalog.ErrInvalidAttachment) {
		t.Errorf("approving with nobody named = %v", err)
	}
	first, err := cm.ApproveFileForPublishing(ctx, public.ID, "file-2", "clerk-1")
	if err != nil || !first.ApprovedForPublishing() || first.ApprovedBy != "clerk-1" {
		t.Fatalf("approved = %+v (%v)", first, err)
	}
	second, err := cm.ApproveFileForPublishing(ctx, public.ID, "file-2", "clerk-2")
	if err != nil || second.ApprovedBy != "clerk-1" || second.ApprovedAt != first.ApprovedAt {
		t.Errorf("a second approval moved the first = %+v (%v)", second, err)
	}
}

func TestThePublicBoardShowsOnlyApprovedFilesWhileTheRaiserAndStaffSeeAll(t *testing.T) {
	cm, ctx := newManager(t)
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Potholes on the main road"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Pothole", Topic: "roads", Publishable: true, RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.GroupUnderBoardItem(ctx, r.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.AttachFiles(ctx, r.ID, "amina", []string{"file-1", "file-2"}); err != nil {
		t.Fatal(err)
	}
	shared, err := cm.OpenSharedBoardItem(ctx, b.ID)
	if err != nil || len(shared.Files) != 0 {
		t.Fatalf("before approval the public saw %+v (%v)", shared.Files, err)
	}
	if _, err := cm.ApproveFileForPublishing(ctx, r.ID, "file-2", "clerk-1"); err != nil {
		t.Fatal(err)
	}
	shared, err = cm.OpenSharedBoardItem(ctx, b.ID)
	if err != nil || !reflect.DeepEqual(shared.Files, []string{"file-2"}) {
		t.Fatalf("after approval the public saw %+v (%v)", shared.Files, err)
	}
	all, err := cm.ListRaisedItemFiles(ctx, r.ID)
	if err != nil || len(all) != 2 {
		t.Fatalf("raiser and staff see %+v (%v)", all, err)
	}
}

func TestWithdrawingAnApprovalTakesTheFileOffThePublicBoard(t *testing.T) {
	cm, ctx := newManager(t)
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Potholes on the main road"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Pothole", Topic: "roads", Publishable: true, RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.GroupUnderBoardItem(ctx, r.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.AttachFiles(ctx, r.ID, "amina", []string{"file-1", "file-2"}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"file-1", "file-2"} {
		if _, err := cm.ApproveFileForPublishing(ctx, r.ID, id, "clerk-1"); err != nil {
			t.Fatal(err)
		}
	}
	withdrawn, err := cm.WithdrawFileApproval(ctx, r.ID, "file-1")
	if err != nil || withdrawn.ApprovedForPublishing() || withdrawn.ApprovedBy != "" {
		t.Fatalf("withdrawn = %+v (%v)", withdrawn, err)
	}
	shared, err := cm.OpenSharedBoardItem(ctx, b.ID)
	if err != nil || !reflect.DeepEqual(shared.Files, []string{"file-2"}) {
		t.Fatalf("after withdrawal the public saw %+v (%v)", shared.Files, err)
	}
	all, err := cm.ListRaisedItemFiles(ctx, r.ID)
	if err != nil || len(all) != 2 {
		t.Fatalf("withdrawal detached the file: %+v (%v)", all, err)
	}
	if _, err := cm.WithdrawFileApproval(ctx, r.ID, "file-1"); err != nil {
		t.Errorf("withdrawing again = %v", err)
	}
	if _, err := cm.WithdrawFileApproval(ctx, r.ID, "file-missing"); !errors.Is(err, catalog.ErrAttachmentNotFound) {
		t.Errorf("withdrawing a file never attached = %v", err)
	}
	again, err := cm.ApproveFileForPublishing(ctx, r.ID, "file-1", "clerk-2")
	if err != nil || again.ApprovedBy != "clerk-2" {
		t.Errorf("approving after a withdrawal = %+v (%v)", again, err)
	}
}

func TestTheRaiserDetachesAFileOnlyUntilItIsApproved(t *testing.T) {
	cm, ctx := newManager(t)
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Pothole", Publishable: true, RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.AttachFiles(ctx, r.ID, "amina", []string{"file-1", "file-2"}); err != nil {
		t.Fatal(err)
	}
	if err := cm.DetachFile(ctx, r.ID, "baraka", "file-1"); !errors.Is(err, catalog.ErrRaisedItemNotFound) {
		t.Errorf("someone else detaching = %v", err)
	}
	if err := cm.DetachFile(ctx, r.ID, "", "file-1"); !errors.Is(err, catalog.ErrRaisedItemNotFound) {
		t.Errorf("nobody detaching = %v", err)
	}
	if err := cm.DetachFile(ctx, "nobody", "amina", "file-1"); !errors.Is(err, catalog.ErrRaisedItemNotFound) {
		t.Errorf("unknown raised item = %v", err)
	}
	if err := cm.DetachFile(ctx, r.ID, "amina", "file-missing"); !errors.Is(err, catalog.ErrAttachmentNotFound) {
		t.Errorf("detaching a file never attached = %v", err)
	}
	if err := cm.DetachFile(ctx, r.ID, "amina", "file-1"); err != nil {
		t.Fatalf("detaching = %v", err)
	}
	if _, err := cm.ApproveFileForPublishing(ctx, r.ID, "file-2", "clerk-1"); err != nil {
		t.Fatal(err)
	}
	if err := cm.DetachFile(ctx, r.ID, "amina", "file-2"); !errors.Is(err, catalog.ErrAttachmentApproved) {
		t.Errorf("detaching an approved file = %v", err)
	}
	all, err := cm.ListRaisedItemFiles(ctx, r.ID)
	if err != nil || len(all) != 1 || all[0].FileID != "file-2" {
		t.Fatalf("left attached = %+v (%v)", all, err)
	}
	if _, err := cm.WithdrawFileApproval(ctx, r.ID, "file-2"); err != nil {
		t.Fatal(err)
	}
	if err := cm.DetachFile(ctx, r.ID, "amina", "file-2"); err != nil {
		t.Errorf("detaching after the approval was withdrawn = %v", err)
	}
}
