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
			if _, err := svc.Create(tc.input); !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}

func TestMintLocationService_CreateRejectsNormalizedDuplicate(t *testing.T) {
	svc := newTestMintLocationService(t)

	if _, err := svc.Create(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := svc.Create(MintLocationInput{DisplayName: " rome! ", Lat: 41.9, Lng: 12.5}); !errors.Is(err, ErrMintLocationDuplicate) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestMintLocationService_CreateRejectsDisplayNameMatchingExistingAlias(t *testing.T) {
	svc := newTestMintLocationService(t)

	if _, err := svc.Create(MintLocationInput{
		DisplayName: "Rome",
		Lat:         41.9,
		Lng:         12.5,
		Aliases:     []string{"Roma"},
	}); err != nil {
		t.Fatalf("create Rome failed: %v", err)
	}

	_, err := svc.Create(MintLocationInput{DisplayName: " roma! ", Lat: 44.4, Lng: 11.3})
	if !errors.Is(err, ErrMintLocationDuplicate) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestMintLocationService_CreateRejectsAliasMatchingExistingDisplayName(t *testing.T) {
	svc := newTestMintLocationService(t)

	if _, err := svc.Create(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5}); err != nil {
		t.Fatalf("create Rome failed: %v", err)
	}

	_, err := svc.Create(MintLocationInput{
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

	created, err := svc.Create(MintLocationInput{
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

	_, err := svc.Create(MintLocationInput{
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

	if _, err := svc.Create(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5}); err != nil {
		t.Fatalf("create Rome failed: %v", err)
	}
	athens, err := svc.Create(MintLocationInput{DisplayName: "Athens", Lat: 37.9, Lng: 23.7})
	if err != nil {
		t.Fatalf("create Athens failed: %v", err)
	}

	_, err = svc.Update(athens.ID, MintLocationInput{DisplayName: "ROME", Lat: 37.9, Lng: 23.7})
	if !errors.Is(err, ErrMintLocationDuplicate) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestMintLocationService_UpdateRejectsLookupKeyCollisionWithAnotherLocation(t *testing.T) {
	t.Run("display name matches existing alias", func(t *testing.T) {
		svc := newTestMintLocationService(t)

		if _, err := svc.Create(MintLocationInput{
			DisplayName: "Rome",
			Lat:         41.9,
			Lng:         12.5,
			Aliases:     []string{"Roma"},
		}); err != nil {
			t.Fatalf("create Rome failed: %v", err)
		}
		athens, err := svc.Create(MintLocationInput{DisplayName: "Athens", Lat: 37.9, Lng: 23.7})
		if err != nil {
			t.Fatalf("create Athens failed: %v", err)
		}

		_, err = svc.Update(athens.ID, MintLocationInput{DisplayName: "Roma", Lat: 37.9, Lng: 23.7})
		if !errors.Is(err, ErrMintLocationDuplicate) {
			t.Fatalf("expected duplicate error, got %v", err)
		}
	})

	t.Run("alias matches existing display name", func(t *testing.T) {
		svc := newTestMintLocationService(t)

		if _, err := svc.Create(MintLocationInput{DisplayName: "Rome", Lat: 41.9, Lng: 12.5}); err != nil {
			t.Fatalf("create Rome failed: %v", err)
		}
		athens, err := svc.Create(MintLocationInput{DisplayName: "Athens", Lat: 37.9, Lng: 23.7})
		if err != nil {
			t.Fatalf("create Athens failed: %v", err)
		}

		_, err = svc.Update(athens.ID, MintLocationInput{
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

func TestMintLocationService_UpdateAllowsOwnExistingAliases(t *testing.T) {
	svc := newTestMintLocationService(t)

	created, err := svc.Create(MintLocationInput{
		DisplayName: "Rome",
		Lat:         41.9,
		Lng:         12.5,
		Aliases:     []string{"Roma", "Rome mint"},
	})
	if err != nil {
		t.Fatalf("create Rome failed: %v", err)
	}

	updated, err := svc.Update(created.ID, MintLocationInput{
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
