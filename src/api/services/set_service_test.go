package services

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupSetServiceTest(t *testing.T) *SetService {
	t.Helper()
	service, _ := setupSetServiceTestWithDB(t)
	return service
}

func setupSetServiceTestWithDB(t *testing.T) (*SetService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Tag{},
		&models.CoinTag{},
		&models.Coin{},
		&models.CoinSet{},
		&models.CoinSetMembership{},
		&models.CoinSetTarget{},
		&models.SmartCriteriaTemplate{},
	); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return NewSetService(repository.NewSetRepository(db), repository.NewTagRepository(db)), db
}

func TestSetService_CreateSet_NormalizesLegacySetTypes(t *testing.T) {
	service := setupSetServiceTest(t)

	openSet, err := service.CreateSet(1, map[string]interface{}{
		"name":    "Legacy Open",
		"setType": "open",
	})
	if err != nil {
		t.Fatalf("create open set failed: %v", err)
	}
	if openSet.SetType != models.CoinSetTypeStandard {
		t.Fatalf("expected open to normalize to %q, got %q", models.CoinSetTypeStandard, openSet.SetType)
	}

	definedSet, err := service.CreateSet(1, map[string]interface{}{
		"name":    "Legacy Defined",
		"setType": "defined",
	})
	if err != nil {
		t.Fatalf("create defined set failed: %v", err)
	}
	if definedSet.SetType != models.CoinSetTypeGoal {
		t.Fatalf("expected defined to normalize to %q, got %q", models.CoinSetTypeGoal, definedSet.SetType)
	}
}

func TestSetService_CreateSet_RejectsDynamicAsSetType(t *testing.T) {
	service := setupSetServiceTest(t)
	_, err := service.CreateSet(1, map[string]interface{}{
		"name":    "Invalid Dynamic Type",
		"setType": "dynamic",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid set type") {
		t.Fatalf("expected invalid set type error, got %v", err)
	}
}

func TestSetService_CreateSet_GeneratesDefaultPaletteColor(t *testing.T) {
	service := setupSetServiceTest(t)

	set, err := service.CreateSet(1, map[string]interface{}{
		"name":    "Palette Default",
		"setType": "standard",
	})
	if err != nil {
		t.Fatalf("create set failed: %v", err)
	}

	expected := defaultSetColor(1, "Palette Default", 0)
	if set.Color != expected {
		t.Fatalf("expected generated color %q, got %q", expected, set.Color)
	}
	if set.Color == "#6b7280" {
		t.Fatalf("expected palette color instead of legacy gray default")
	}
}

func TestSetService_CreateSet_DisablesDirectAgenticCreation(t *testing.T) {
	service := setupSetServiceTest(t)

	_, err := service.CreateSet(1, map[string]interface{}{
		"name":         "Invalid Dynamic Goal",
		"setType":      "goal",
		"creationMode": "dynamic",
	})
	if err == nil || !strings.Contains(err.Error(), "only valid for agentic sets") {
		t.Fatalf("expected agentic mode validation error, got %v", err)
	}

	_, err = service.CreateSet(1, map[string]interface{}{
		"name":          "US Silver Quarters",
		"setType":       "agentic",
		"agenticPrompt": "All US Silver Quarters from 1940s to 1960s",
	})
	if err == nil || !strings.Contains(err.Error(), "proposal review workflow") {
		t.Fatalf("expected direct agentic creation disabled error, got %v", err)
	}

	count, countErr := service.repo.CountByUser(1)
	if countErr != nil {
		t.Fatalf("count sets: %v", countErr)
	}
	if count != 0 {
		t.Fatalf("expected no set to be created, got %d", count)
	}
}

func TestSetService_AddCoinToSet_AgenticRequiresTargetID(t *testing.T) {
	service, db := setupSetServiceTestWithDB(t)

	set := models.CoinSet{UserID: 1, Name: "Agentic Slot Set", SetType: models.CoinSetTypeAgentic, CreationMode: models.CoinSetCreationModeDynamic}
	if err := service.repo.Create(&set); err != nil {
		t.Fatalf("create set: %v", err)
	}
	coin := models.Coin{Name: "Test Coin", UserID: 1}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("create coin: %v", err)
	}

	err := service.AddCoinToSet(coin.ID, set.ID, 1, "")
	if err == nil || !strings.Contains(err.Error(), "targetId is required") {
		t.Fatalf("expected agentic targetId validation error, got %v", err)
	}
}

// pinnedAtFromMap safely reads a *time.Time out of a map[string]interface{}
// response. A nil PinnedAt is stored as a typed-nil *time.Time inside the
// interface, so a bare `!= nil` comparison on the interface always reports
// true; the value must be type-asserted first.
func pinnedAtFromMap(t *testing.T, data map[string]interface{}) *time.Time {
	t.Helper()
	raw, ok := data["pinnedAt"]
	if !ok {
		t.Fatalf("expected pinnedAt key to be present")
	}
	ptr, ok := raw.(*time.Time)
	if !ok {
		t.Fatalf("expected pinnedAt to be *time.Time, got %T", raw)
	}
	return ptr
}

func TestSetService_UpdateSet_PinSetsUTCTimestamp(t *testing.T) {
	service := setupSetServiceTest(t)
	set, err := service.CreateSet(1, map[string]interface{}{"name": "Twelve Caesars", "setType": "standard"})
	if err != nil {
		t.Fatalf("create set: %v", err)
	}

	updated, err := service.UpdateSet(set.ID, 1, map[string]interface{}{"pinned": true})
	if err != nil {
		t.Fatalf("pin set: %v", err)
	}
	if updated.PinnedAt == nil {
		t.Fatalf("expected pinnedAt to be set")
	}
	if updated.PinnedAt.Location() != time.UTC {
		t.Fatalf("expected pinnedAt to be UTC, got %v", updated.PinnedAt.Location())
	}
}

func TestSetService_UpdateSet_UnpinNullsTimestamp(t *testing.T) {
	service := setupSetServiceTest(t)
	set, err := service.CreateSet(1, map[string]interface{}{"name": "Twelve Caesars", "setType": "standard"})
	if err != nil {
		t.Fatalf("create set: %v", err)
	}
	if _, err := service.UpdateSet(set.ID, 1, map[string]interface{}{"pinned": true}); err != nil {
		t.Fatalf("pin set: %v", err)
	}

	updated, err := service.UpdateSet(set.ID, 1, map[string]interface{}{"pinned": false})
	if err != nil {
		t.Fatalf("unpin set: %v", err)
	}
	if updated.PinnedAt != nil {
		t.Fatalf("expected pinnedAt to be nil after unpin, got %v", updated.PinnedAt)
	}
}

func TestSetService_UpdateSet_RepinPreservesOriginalTimestamp(t *testing.T) {
	service := setupSetServiceTest(t)
	set, err := service.CreateSet(1, map[string]interface{}{"name": "Twelve Caesars", "setType": "standard"})
	if err != nil {
		t.Fatalf("create set: %v", err)
	}
	first, err := service.UpdateSet(set.ID, 1, map[string]interface{}{"pinned": true})
	if err != nil {
		t.Fatalf("pin set: %v", err)
	}
	firstPinnedAt := *first.PinnedAt

	time.Sleep(2 * time.Millisecond)
	second, err := service.UpdateSet(set.ID, 1, map[string]interface{}{"pinned": true})
	if err != nil {
		t.Fatalf("re-pin set: %v", err)
	}
	if second.PinnedAt == nil || !second.PinnedAt.Equal(firstPinnedAt) {
		t.Fatalf("expected pinnedAt to be preserved on double-pin, first=%v second=%v", firstPinnedAt, second.PinnedAt)
	}
}

func TestSetService_UpdateSet_PinCap_RejectsSixthPin(t *testing.T) {
	service := setupSetServiceTest(t)
	var setIDs []uint
	for i := 0; i < 6; i++ {
		set, err := service.CreateSet(1, map[string]interface{}{"name": fmt.Sprintf("Set %d", i), "setType": "standard"})
		if err != nil {
			t.Fatalf("create set %d: %v", i, err)
		}
		setIDs = append(setIDs, set.ID)
	}

	for i := 0; i < 5; i++ {
		if _, err := service.UpdateSet(setIDs[i], 1, map[string]interface{}{"pinned": true}); err != nil {
			t.Fatalf("pin set %d: %v", i, err)
		}
	}

	_, err := service.UpdateSet(setIDs[5], 1, map[string]interface{}{"pinned": true})
	if !errors.Is(err, ErrPinLimitReached) {
		t.Fatalf("expected ErrPinLimitReached, got %v", err)
	}
	if err.Error() != "you can pin up to 5 sets" {
		t.Fatalf("expected exact cap message, got %q", err.Error())
	}

	sixth, err := service.repo.GetByID(setIDs[5], 1)
	if err != nil {
		t.Fatalf("get sixth set: %v", err)
	}
	if sixth.PinnedAt != nil {
		t.Fatalf("expected sixth set to remain unpinned after cap rejection")
	}
}

func TestSetService_UpdateSet_PinCapRecoversAfterUnpin(t *testing.T) {
	service := setupSetServiceTest(t)
	var setIDs []uint
	for i := 0; i < 6; i++ {
		set, err := service.CreateSet(1, map[string]interface{}{"name": fmt.Sprintf("Set %d", i), "setType": "standard"})
		if err != nil {
			t.Fatalf("create set %d: %v", i, err)
		}
		setIDs = append(setIDs, set.ID)
	}
	for i := 0; i < 5; i++ {
		if _, err := service.UpdateSet(setIDs[i], 1, map[string]interface{}{"pinned": true}); err != nil {
			t.Fatalf("pin set %d: %v", i, err)
		}
	}
	if _, err := service.UpdateSet(setIDs[5], 1, map[string]interface{}{"pinned": true}); !errors.Is(err, ErrPinLimitReached) {
		t.Fatalf("expected cap rejection before unpin, got %v", err)
	}

	if _, err := service.UpdateSet(setIDs[0], 1, map[string]interface{}{"pinned": false}); err != nil {
		t.Fatalf("unpin set 0: %v", err)
	}

	sixth, err := service.UpdateSet(setIDs[5], 1, map[string]interface{}{"pinned": true})
	if err != nil {
		t.Fatalf("expected pin to succeed after cap recovery, got %v", err)
	}
	if sixth.PinnedAt == nil {
		t.Fatalf("expected sixth set to be pinned after recovery")
	}
}

func TestSetService_UpdateSet_PinForeignSet_NotFound(t *testing.T) {
	service := setupSetServiceTest(t)
	set, err := service.CreateSet(1, map[string]interface{}{"name": "Owner's Set", "setType": "standard"})
	if err != nil {
		t.Fatalf("create set: %v", err)
	}

	_, err = service.UpdateSet(set.ID, 2, map[string]interface{}{"pinned": true})
	if err == nil || !repository.IsRecordNotFound(err) {
		t.Fatalf("expected not found error for foreign set pin, got %v", err)
	}
}

func TestSetService_ListSets_IncludesPinnedFields(t *testing.T) {
	service := setupSetServiceTest(t)
	set, err := service.CreateSet(1, map[string]interface{}{"name": "Twelve Caesars", "setType": "standard"})
	if err != nil {
		t.Fatalf("create set: %v", err)
	}
	if _, err := service.UpdateSet(set.ID, 1, map[string]interface{}{"pinned": true}); err != nil {
		t.Fatalf("pin set: %v", err)
	}

	sets, err := service.ListSets(1)
	if err != nil {
		t.Fatalf("list sets: %v", err)
	}
	if len(sets) != 1 {
		t.Fatalf("expected 1 set, got %d", len(sets))
	}
	if pinned, ok := sets[0]["pinned"].(bool); !ok || !pinned {
		t.Fatalf("expected pinned=true in list summary, got %v", sets[0]["pinned"])
	}
	if ptr := pinnedAtFromMap(t, sets[0]); ptr == nil {
		t.Fatalf("expected non-nil pinnedAt in list summary")
	}
}

func TestSetService_GetSetDetail_IncludesPinnedFields(t *testing.T) {
	service := setupSetServiceTest(t)
	set, err := service.CreateSet(1, map[string]interface{}{"name": "Twelve Caesars", "setType": "standard"})
	if err != nil {
		t.Fatalf("create set: %v", err)
	}

	detail, err := service.GetSetDetail(set.ID, 1)
	if err != nil {
		t.Fatalf("get set detail: %v", err)
	}
	if pinned, ok := detail["pinned"].(bool); !ok || pinned {
		t.Fatalf("expected pinned=false before pinning, got %v", detail["pinned"])
	}
	if ptr := pinnedAtFromMap(t, detail); ptr != nil {
		t.Fatalf("expected nil pinnedAt before pinning, got %v", ptr)
	}

	if _, err := service.UpdateSet(set.ID, 1, map[string]interface{}{"pinned": true}); err != nil {
		t.Fatalf("pin set: %v", err)
	}
	detail, err = service.GetSetDetail(set.ID, 1)
	if err != nil {
		t.Fatalf("get set detail after pin: %v", err)
	}
	if pinned, ok := detail["pinned"].(bool); !ok || !pinned {
		t.Fatalf("expected pinned=true after pinning, got %v", detail["pinned"])
	}
	if ptr := pinnedAtFromMap(t, detail); ptr == nil {
		t.Fatalf("expected non-nil pinnedAt after pinning")
	}
}
