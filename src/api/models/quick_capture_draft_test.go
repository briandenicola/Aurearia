package models

import "testing"

func TestQuickCaptureDraftEnums(t *testing.T) {
	if !IsValidQuickCaptureDraftStatus(QuickCaptureDraftStatusActive) {
		t.Fatal("active status should be valid")
	}
	if IsValidQuickCaptureDraftStatus(QuickCaptureDraftStatus("archived")) {
		t.Fatal("unexpected status should be invalid")
	}
	if !IsValidDraftLifecycleEventType(DraftLifecycleEventImageAdded) {
		t.Fatal("image_added lifecycle event should be valid")
	}
	if IsValidDraftLifecycleEventType(DraftLifecycleEventType("raw filesystem path")) {
		t.Fatal("unexpected lifecycle event should be invalid")
	}
}
