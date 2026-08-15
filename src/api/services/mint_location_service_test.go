package services

import (
	"context"
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

// fakeNomismaClient is a test double for NomismaClient. If failIfCalled is
// set, Search fails the test immediately - used to prove a private mint's
// Nomisma methods never reach the client at all (User Story 4).
type fakeNomismaClient struct {
	t            *testing.T
	failIfCalled bool
	candidates   []NomismaCandidate
	kind         NomismaErrorKind
	err          error
	calls        int
}

func (f *fakeNomismaClient) Search(ctx context.Context, query string, limit int) ([]NomismaCandidate, NomismaErrorKind, error) {
	f.calls++
	if f.failIfCalled {
		f.t.Fatalf("NomismaClient.Search must never be invoked for this test, got query %q", query)
	}
	return f.candidates, f.kind, f.err
}

func newTestMintLocationServiceWithNomisma(t *testing.T, client NomismaClient) *MintLocationService {
	t.Helper()
	db := setupTestDB(t)
	return NewMintLocationService(repository.NewMintLocationRepository(db)).
		WithNomisma(client, NewNomismaCache())
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

// --- Nomisma authority linking (343-nomisma-mint-authority-linking) ---

func TestMintLocationService_SearchNomisma_OK(t *testing.T) {
	client := &fakeNomismaClient{t: t, candidates: []NomismaCandidate{{URI: "http://nomisma.org/id/roma", Label: "Roma", Score: 100, Match: true}}}
	svc := newTestMintLocationServiceWithNomisma(t, client)
	mint, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	outcome, err := svc.SearchNomisma(mint.ID, "Roma")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outcome.Status != NomismaSearchOK || len(outcome.Candidates) != 1 {
		t.Fatalf("unexpected outcome: %+v", outcome)
	}
}

func TestMintLocationService_SearchNomisma_NoMatch(t *testing.T) {
	client := &fakeNomismaClient{t: t, kind: NomismaErrorNoMatch}
	svc := newTestMintLocationServiceWithNomisma(t, client)
	mint, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	outcome, err := svc.SearchNomisma(mint.ID, "zzzzgibberish")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outcome.Status != NomismaSearchNoMatch || len(outcome.Candidates) != 0 {
		t.Fatalf("unexpected outcome: %+v", outcome)
	}
}

func TestMintLocationService_SearchNomisma_Unavailable(t *testing.T) {
	client := &fakeNomismaClient{t: t, kind: NomismaErrorUnavailable, err: errors.New("boom")}
	svc := newTestMintLocationServiceWithNomisma(t, client)
	mint, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	outcome, err := svc.SearchNomisma(mint.ID, "Roma")
	if err != nil {
		t.Fatalf("expected no hard error for an upstream failure, got %v", err)
	}
	if outcome.Status != NomismaSearchUnavailable || len(outcome.Candidates) != 0 {
		t.Fatalf("unexpected outcome: %+v", outcome)
	}
}

func TestMintLocationService_SearchNomisma_InvalidQuery(t *testing.T) {
	client := &fakeNomismaClient{t: t, failIfCalled: true}
	svc := newTestMintLocationServiceWithNomisma(t, client)
	mint, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if _, err := svc.SearchNomisma(mint.ID, "   "); !errors.Is(err, ErrMintLocationNomismaQueryInvalid) {
		t.Fatalf("expected invalid query error, got %v", err)
	}
}

func TestMintLocationService_SearchNomisma_PrivateMintNotFound(t *testing.T) {
	client := &fakeNomismaClient{t: t, failIfCalled: true}
	svc := newTestMintLocationServiceWithNomisma(t, client)
	mint, err := svc.CreatePrivate(1, MintLocationInput{DisplayName: "My Mint", Lat: 1, Lng: 1})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if _, err := svc.SearchNomisma(mint.ID, "Roma"); !errors.Is(err, ErrMintLocationNotFound) {
		t.Fatalf("expected not found for a private mint, got %v", err)
	}
}

func TestMintLocationService_LinkNomismaGlobal_HappyPath(t *testing.T) {
	svc := newTestMintLocationServiceWithNomisma(t, &fakeNomismaClient{t: t})
	mint, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	updated, err := svc.LinkNomismaGlobal(mint.ID, "http://nomisma.org/id/roma", "Roma")
	if err != nil {
		t.Fatalf("link failed: %v", err)
	}
	if updated.NomismaURI == nil || *updated.NomismaURI != "http://nomisma.org/id/roma" {
		t.Fatalf("expected NomismaURI set, got %+v", updated)
	}
	if updated.NomismaLabel != "Roma" {
		t.Fatalf("expected NomismaLabel set, got %+v", updated)
	}
	if updated.NomismaLinkedAt == nil {
		t.Fatalf("expected NomismaLinkedAt set, got %+v", updated)
	}
}

func TestMintLocationService_LinkNomismaGlobal_InvalidURIHost(t *testing.T) {
	svc := newTestMintLocationServiceWithNomisma(t, &fakeNomismaClient{t: t})
	mint, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	cases := []string{"http://evil.example.com/id/roma", "not-a-url", "ftp://nomisma.org/id/roma", ""}
	for _, uri := range cases {
		if _, err := svc.LinkNomismaGlobal(mint.ID, uri, "Roma"); !errors.Is(err, ErrMintLocationNomismaURIInvalid) {
			t.Fatalf("expected URI invalid error for %q, got %v", uri, err)
		}
	}
}

func TestMintLocationService_LinkNomismaGlobal_BlankOrTooLongLabel(t *testing.T) {
	svc := newTestMintLocationServiceWithNomisma(t, &fakeNomismaClient{t: t})
	mint, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if _, err := svc.LinkNomismaGlobal(mint.ID, "http://nomisma.org/id/roma", "  "); !errors.Is(err, ErrMintLocationNomismaLabelInvalid) {
		t.Fatalf("expected label invalid error for blank label, got %v", err)
	}

	tooLong := make([]byte, 257)
	for i := range tooLong {
		tooLong[i] = 'a'
	}
	if _, err := svc.LinkNomismaGlobal(mint.ID, "http://nomisma.org/id/roma", string(tooLong)); !errors.Is(err, ErrMintLocationNomismaLabelInvalid) {
		t.Fatalf("expected label invalid error for too-long label, got %v", err)
	}
}

func TestMintLocationService_LinkNomismaGlobal_PrivateMintNotFound(t *testing.T) {
	svc := newTestMintLocationServiceWithNomisma(t, &fakeNomismaClient{t: t})
	mint, err := svc.CreatePrivate(1, MintLocationInput{DisplayName: "My Mint", Lat: 1, Lng: 1})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if _, err := svc.LinkNomismaGlobal(mint.ID, "http://nomisma.org/id/roma", "Roma"); !errors.Is(err, ErrMintLocationNotFound) {
		t.Fatalf("expected not found for a private mint, got %v", err)
	}
}

func TestMintLocationService_LinkNomismaGlobal_ReplacesExistingLinkWithoutAlteringOtherFields(t *testing.T) {
	svc := newTestMintLocationServiceWithNomisma(t, &fakeNomismaClient{t: t})
	mint, err := svc.CreateGlobal(MintLocationInput{
		DisplayName: "Rome", Lat: 41.9, Lng: 12.5, Region: "Italy", Aliases: []string{"Roma"},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := svc.LinkNomismaGlobal(mint.ID, "http://nomisma.org/id/roma", "Roma"); err != nil {
		t.Fatalf("first link failed: %v", err)
	}

	updated, err := svc.LinkNomismaGlobal(mint.ID, "http://nomisma.org/id/roma_alt", "Roma (alt)")
	if err != nil {
		t.Fatalf("replace link failed: %v", err)
	}
	if updated.NomismaURI == nil || *updated.NomismaURI != "http://nomisma.org/id/roma_alt" || updated.NomismaLabel != "Roma (alt)" {
		t.Fatalf("expected replaced link fields, got %+v", updated)
	}
	if updated.DisplayName != "Rome" || updated.Lat != 41.9 || updated.Lng != 12.5 || updated.Region != "Italy" {
		t.Fatalf("expected display fields unchanged after replacing a link, got %+v", updated)
	}
	if len(updated.Aliases) != 1 || updated.Aliases[0] != "Roma" {
		t.Fatalf("expected aliases unchanged after replacing a link, got %+v", updated.Aliases)
	}
}

func TestMintLocationService_UnlinkNomismaGlobal_ClearsExistingLink(t *testing.T) {
	svc := newTestMintLocationServiceWithNomisma(t, &fakeNomismaClient{t: t})
	mint, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := svc.LinkNomismaGlobal(mint.ID, "http://nomisma.org/id/roma", "Roma"); err != nil {
		t.Fatalf("link failed: %v", err)
	}

	updated, err := svc.UnlinkNomismaGlobal(mint.ID)
	if err != nil {
		t.Fatalf("unlink failed: %v", err)
	}
	if updated.NomismaURI != nil || updated.NomismaLabel != "" || updated.NomismaLinkedAt != nil {
		t.Fatalf("expected Nomisma fields cleared, got %+v", updated)
	}
}

func TestMintLocationService_UnlinkNomismaGlobal_IdempotentNoOp(t *testing.T) {
	svc := newTestMintLocationServiceWithNomisma(t, &fakeNomismaClient{t: t})
	mint, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if _, err := svc.UnlinkNomismaGlobal(mint.ID); err != nil {
		t.Fatalf("expected a no-op success on an already-unlinked mint, got %v", err)
	}
	if _, err := svc.UnlinkNomismaGlobal(mint.ID); err != nil {
		t.Fatalf("expected idempotent double-unlink success, got %v", err)
	}
}

func TestMintLocationService_UnlinkNomismaGlobal_PrivateMintNotFound(t *testing.T) {
	svc := newTestMintLocationServiceWithNomisma(t, &fakeNomismaClient{t: t})
	mint, err := svc.CreatePrivate(1, MintLocationInput{DisplayName: "My Mint", Lat: 1, Lng: 1})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if _, err := svc.UnlinkNomismaGlobal(mint.ID); !errors.Is(err, ErrMintLocationNotFound) {
		t.Fatalf("expected not found for a private mint, got %v", err)
	}
}

func TestMintLocationService_UnlinkNomismaGlobal_NeverCallsNomismaClient(t *testing.T) {
	svc := newTestMintLocationServiceWithNomisma(t, &fakeNomismaClient{t: t, failIfCalled: true})
	mint, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := svc.UnlinkNomismaGlobal(mint.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestMintLocationService_MintCoinCRUDUnaffectedByNomismaOutage proves
// UpdateGlobal (general mint/coin CRUD) keeps working even when the
// NomismaClient errors on every call - the two code paths are structurally
// independent (FR-015, SC-004).
func TestMintLocationService_MintCoinCRUDUnaffectedByNomismaOutage(t *testing.T) {
	client := &fakeNomismaClient{t: t, kind: NomismaErrorUnavailable, err: errors.New("nomisma is down")}
	svc := newTestMintLocationServiceWithNomisma(t, client)
	mint, err := svc.CreateGlobal(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if _, err := svc.SearchNomisma(mint.ID, "Roma"); err != nil {
		t.Fatalf("expected SearchNomisma to degrade gracefully, got %v", err)
	}

	updated, err := svc.UpdateGlobal(mint.ID, MintLocationInput{DisplayName: "Roma", Lat: 41.91, Lng: 12.51})
	if err != nil {
		t.Fatalf("expected UpdateGlobal to succeed despite Nomisma outage, got %v", err)
	}
	if updated.DisplayName != "Roma" {
		t.Fatalf("expected UpdateGlobal to apply, got %+v", updated)
	}
}
