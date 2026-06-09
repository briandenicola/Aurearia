package services

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

func TestCatalogRegistryService_CreateAcceptsCustomEra(t *testing.T) {
	db := setupTestDB(t)
	svc := NewCatalogRegistryService(repository.NewCatalogRegistryRepository(db))

	created, err := svc.Create(models.CatalogRegistry{
		Catalog:     "PROV",
		DisplayName: "Provincial References",
		Era:         models.Era("provincial"),
	})
	if err != nil {
		t.Fatalf("Create should accept custom era: %v", err)
	}
	if created.Era != models.Era("provincial") {
		t.Fatalf("expected custom era to be preserved, got %q", created.Era)
	}
}
