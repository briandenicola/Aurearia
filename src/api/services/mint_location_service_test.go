package services

import (
	"errors"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

func newTestMintLocationService(t *testing.T) *MintLocationService {
	t.Helper()
	db := setupTestDB(t)
	return NewMintLocationService(repository.NewMintLocationRepository(db))
}

func TestMintLocationService_CreateValidatesCoordinatesAndName(t *testing.T) {
	svc := newTestMintLocationService(t)

	cases := []struct {
		name  string
		input MintLocationInput
		want  error
	}{
		{
			name:  "blank name",
			input: MintLocationInput{DisplayName: " ", Lat: 10, Lng: 10},
			want:  ErrMintLocationNameRequired,
		},
		{
			name:  "bad lat",
			input: MintLocationInput{DisplayName: "Rome", Lat: 91, Lng: 10},
			want:  ErrMintLocationLatInvalid,
		},
		{
			name:  "bad lng",
			input: MintLocationInput{DisplayName: "Rome", Lat: 10, Lng: -181},
			want:  ErrMintLocationLngInvalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.CreateGlobal(tc.input); !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}

func TestMintLocationService_CreateRejectsNormalizedDuplicate(t *testing.T) {
	svc := newTestMintLocationService(t)

	if _, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := svc.CreateGlobal(MintLocationInput{DisplayName: " rome! ", Lat: 41.9, Lng: 12.5}); !errors.Is(err, ErrMintLocationDuplicate) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestMintLocationService_CreateRejectsDisplayNameMatchingExistingAlias(t *testing.T) {
	svc := newTestMintLocationService(t)

	if _, err := svc.CreateGlobal(MintLocationInput{
		DisplayName: "Rome",
		Lat:         41.9,
		Lng:         12.5,
		Aliases:     []string{"Roma"},
	}); err != nil {
		t.Fatalf("create Rome failed: %v", err)
	}

	_, err := svc.CreateGlobal(MintLocationInput{DisplayName: " roma! ", Lat: 44.4, Lng: 11.3})
	if !errors.Is(err, ErrMintLocationDuplicate) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestMintLocationService_CreateRejectsAliasMatchingExistingDisplayName(t *testing.T) {
	svc := newTestMintLocationService(t)

	if _, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5}); err != nil {
		t.Fatalf("create Rome failed: %v", err)
	}

	_, err := svc.CreateGlobal(MintLocationInput{
		DisplayName: "Athens",
		Lat:         37.9,
		Lng:         23.7,
		Aliases:     []string{" ROME! "},
	})
	if !errors.Is(err, ErrMintLocationDuplicate) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestMintLocationService_NormalizesAliases(t *testing.T) {
	svc := newTestMintLocationService(t)

	created, err := svc.CreateGlobal(MintLocationInput{
		DisplayName: "Rome",
		Lat:         41.9,
		Lng:         12.5,
		Aliases:     []string{" Roma ", "Roma", "Rome", "Rome mint"},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	want := models.StringList{"Roma", "Rome mint"}
	if len(created.Aliases) != len(want) {
		t.Fatalf("expected aliases %v, got %v", want, created.Aliases)
	}
	for i := range want {
		if created.Aliases[i] != want[i] {
			t.Fatalf("expected aliases %v, got %v", want, created.Aliases)
		}
	}
}

func TestMintLocationService_RejectsBlankAlias(t *testing.T) {
	svc := newTestMintLocationService(t)

	_, err := svc.CreateGlobal(MintLocationInput{
		DisplayName: "Athens",
		Lat:         37.9,
		Lng:         23.7,
		Aliases:     []string{"Athenae", " "},
	})
	if !errors.Is(err, ErrMintLocationAliasInvalid) {
		t.Fatalf("expected blank alias error, got %v", err)
	}
}

func TestMintLocationService_UpdateDuplicateRejected(t *testing.T) {
	svc := newTestMintLocationService(t)

	if _, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5}); err != nil {
		t.Fatalf("create Rome failed: %v", err)
	}
	athens, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Athens", Lat: 37.9, Lng: 23.7})
	if err != nil {
		t.Fatalf("create Athens failed: %v", err)
	}

	_, err = svc.UpdateGlobal(athens.ID, MintLocationInput{DisplayName: "ROME", Lat: 37.9, Lng: 23.7})
	if !errors.Is(err, ErrMintLocationDuplicate) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestMintLocationService_UpdateRejectsLookupKeyCollisionWithAnotherLocation(t *testing.T) {
	t.Run("display name matches existing alias", func(t *testing.T) {
		svc := newTestMintLocationService(t)

		if _, err := svc.CreateGlobal(MintLocationInput{
			DisplayName: "Rome",
			Lat:         41.9,
			Lng:         12.5,
			Aliases:     []string{"Roma"},
		}); err != nil {
			t.Fatalf("create Rome failed: %v", err)
		}
		athens, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Athens", Lat: 37.9, Lng: 23.7})
		if err != nil {
			t.Fatalf("create Athens failed: %v", err)
		}

		_, err = svc.UpdateGlobal(athens.ID, MintLocationInput{DisplayName: "Roma", Lat: 37.9, Lng: 23.7})
		if !errors.Is(err, ErrMintLocationDuplicate) {
			t.Fatalf("expected duplicate error, got %v", err)
		}
	})

	t.Run("alias matches existing display name", func(t *testing.T) {
		svc := newTestMintLocationService(t)

		if _, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5}); err != nil {
			t.Fatalf("create Rome failed: %v", err)
		}
		athens, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Athens", Lat: 37.9, Lng: 23.7})
		if err != nil {
			t.Fatalf("create Athens failed: %v", err)
		}

		_, err = svc.UpdateGlobal(athens.ID, MintLocationInput{
			DisplayName: "Athens",
			Lat:         37.9,
			Lng:         23.7,
			Aliases:     []string{"Rome"},
		})
		if !errors.Is(err, ErrMintLocationDuplicate) {
			t.Fatalf("expected duplicate error, got %v", err)
		}
	})
}

// --- Private (self-service) CRUD tests ---

func TestMintLocationService_CreatePrivateSetsUserID(t *testing.T) {
	svc := newTestMintLocationService(t)
	const userID = uint(7)

	created, err := svc.CreatePrivate(userID, MintLocationInput{
		DisplayName: "My Custom Mint",
		Lat:         10,
		Lng:         20,
		Region:      "Test Region",
		Aliases:     []string{"Custom"},
	})
	if err != nil {
		t.Fatalf("CreatePrivate failed: %v", err)
	}
	if created.UserID == nil || *created.UserID != userID {
		t.Fatalf("expected UserID %d, got %v", userID, created.UserID)
	}
	if created.DisplayName != "My Custom Mint" || created.Region != "Test Region" {
		t.Fatalf("unexpected fields: %+v", created)
	}
}

func TestMintLocationService_CreatePrivateDuplicateRejectsNameCollidingWithGlobal(t *testing.T) {
	svc := newTestMintLocationService(t)

	if _, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5}); err != nil {
		t.Fatalf("global create failed: %v", err)
	}

	// Private mint with same normalized name as global should be rejected
	// because ListVisibleTo includes global entries.
	_, err := svc.CreatePrivate(5, MintLocationInput{DisplayName: "ROME", Lat: 41.9, Lng: 12.5})
	if !errors.Is(err, ErrMintLocationDuplicate) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestMintLocationService_CreatePrivateTwoUsersSameNameAllowed(t *testing.T) {
	svc := newTestMintLocationService(t)

	// Two different users may have private mints with the same name since
	// they live in separate visibility scopes.
	if _, err := svc.CreatePrivate(1, MintLocationInput{DisplayName: "Local Mint", Lat: 10, Lng: 10}); err != nil {
		t.Fatalf("user 1 create failed: %v", err)
	}
	if _, err := svc.CreatePrivate(2, MintLocationInput{DisplayName: "Local Mint", Lat: 10, Lng: 10}); err != nil {
		t.Fatalf("user 2 create failed: %v", err)
	}
}

func TestMintLocationService_UpdatePrivateOwnerScopingRejectsOtherUser(t *testing.T) {
	svc := newTestMintLocationService(t)

	created, err := svc.CreatePrivate(1, MintLocationInput{DisplayName: "User 1 Mint", Lat: 10, Lng: 10})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// User 2 must not be able to update user 1's private mint.
	_, err = svc.UpdatePrivate(created.ID, 2, MintLocationInput{DisplayName: "Hijacked", Lat: 1, Lng: 1})
	if !errors.Is(err, ErrMintLocationNotFound) {
		t.Fatalf("expected not-found, got %v", err)
	}
}

func TestMintLocationService_UpdatePrivateOwnerCanRenameOwnMint(t *testing.T) {
	svc := newTestMintLocationService(t)

	created, err := svc.CreatePrivate(3, MintLocationInput{DisplayName: "Old Name", Lat: 10, Lng: 10})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	updated, err := svc.UpdatePrivate(created.ID, 3, MintLocationInput{
		DisplayName: "New Name",
		Lat:         11,
		Lng:         11,
		Region:      "Pannonia",
	})
	if err != nil {
		t.Fatalf("UpdatePrivate failed: %v", err)
	}
	if updated.DisplayName != "New Name" || updated.Region != "Pannonia" {
		t.Fatalf("unexpected fields: %+v", updated)
	}
}

func TestMintLocationService_UpdatePrivateCannotMutateGlobalMint(t *testing.T) {
	svc := newTestMintLocationService(t)

	global, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5})
	if err != nil {
		t.Fatalf("global create failed: %v", err)
	}

	_, err = svc.UpdatePrivate(global.ID, 1, MintLocationInput{DisplayName: "Hijacked", Lat: 1, Lng: 1})
	if !errors.Is(err, ErrMintLocationNotFound) {
		t.Fatalf("expected not-found, got %v", err)
	}
}

func TestMintLocationService_DeletePrivateOwnerScopingRejectsOtherUser(t *testing.T) {
	svc := newTestMintLocationService(t)

	created, err := svc.CreatePrivate(1, MintLocationInput{DisplayName: "User 1 Mint", Lat: 10, Lng: 10})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// User 2 must not be able to delete user 1's private mint.
	if err := svc.DeletePrivate(created.ID, 2); !errors.Is(err, ErrMintLocationNotFound) {
		t.Fatalf("expected not-found, got %v", err)
	}
}

func TestMintLocationService_DeletePrivateCannotDeleteGlobalMint(t *testing.T) {
	svc := newTestMintLocationService(t)

	global, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5})
	if err != nil {
		t.Fatalf("global create failed: %v", err)
	}

	if err := svc.DeletePrivate(global.ID, 1); !errors.Is(err, ErrMintLocationNotFound) {
		t.Fatalf("expected not-found, got %v", err)
	}
}

func TestMintLocationService_ListReturnsGlobalAndUsersOwn(t *testing.T) {
	svc := newTestMintLocationService(t)

	if _, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5}); err != nil {
		t.Fatalf("global create failed: %v", err)
	}
	if _, err := svc.CreatePrivate(1, MintLocationInput{DisplayName: "User Mint", Lat: 10, Lng: 10}); err != nil {
		t.Fatalf("private create failed: %v", err)
	}
	if _, err := svc.CreatePrivate(2, MintLocationInput{DisplayName: "Other User Mint", Lat: 5, Lng: 5}); err != nil {
		t.Fatalf("other private create failed: %v", err)
	}

	all, err := svc.List(1)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	// User 1 should see global "Rome" + their own "User Mint" — not user 2's.
	if len(all) != 2 {
		t.Fatalf("expected 2 locations (global + user 1's own), got %d: %+v", len(all), all)
	}
	names := map[string]bool{}
	for _, loc := range all {
		names[loc.DisplayName] = true
	}
	if !names["Rome"] || !names["User Mint"] {
		t.Fatalf("unexpected names in list: %v", names)
	}
}

func TestMintLocationService_UpdateAllowsOwnExistingAliases(t *testing.T) {
	svc := newTestMintLocationService(t)

	created, err := svc.CreateGlobal(MintLocationInput{
		DisplayName: "Rome",
		Lat:         41.9,
		Lng:         12.5,
		Aliases:     []string{"Roma", "Rome mint"},
	})
	if err != nil {
		t.Fatalf("create Rome failed: %v", err)
	}

	updated, err := svc.UpdateGlobal(created.ID, MintLocationInput{
		DisplayName: "Rome",
		Lat:         41.91,
		Lng:         12.49,
		Aliases:     []string{"Roma", "Roma", "Rome", "Rome mint"},
	})
	if err != nil {
		t.Fatalf("update with own aliases failed: %v", err)
	}

	want := models.StringList{"Roma", "Rome mint"}
	if len(updated.Aliases) != len(want) {
		t.Fatalf("expected aliases %v, got %v", want, updated.Aliases)
	}
	for i := range want {
		if updated.Aliases[i] != want[i] {
			t.Fatalf("expected aliases %v, got %v", want, updated.Aliases)
		}
	}
}
