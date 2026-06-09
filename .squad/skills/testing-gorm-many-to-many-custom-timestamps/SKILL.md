---
name: "testing-gorm-many-to-many-custom-timestamps"
description: "How to test GORM many-to-many relationships with join tables that have custom timestamp fields (beyond CreatedAt/UpdatedAt)"
domain: "testing"
confidence: "high"
source: "earned"
---

## Context

GORM's automatic association handling **does not populate custom timestamp fields** in join tables during association replacement. When a many-to-many join table has fields like `AddedAt` (for CoinSetMembership) marked as NOT NULL, naive GORM updates will violate the constraint.

This skill covers:
- Testing that Update operations preserve existing join table rows with custom timestamps
- Verifying that `Omit()` correctly prevents association replacement
- Setting up test databases to include join table models

## Patterns

### 1. Include Join Table Models in Test Setup

```go
func setupTestDB(t *testing.T) *gorm.DB {
    db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    db.AutoMigrate(
        &models.Coin{}, &models.CoinSet{},
        &models.CoinSetMembership{},  // ← Must include join table
    )
    return db
}
```

### 2. Test Pattern: Verify Timestamp Preservation

```go
func TestUpdate_PreservesCustomTimestamps(t *testing.T) {
    // 1. Create entity with association via dedicated method (populates timestamp)
    setRepo.AddCoinToSet(coinID, setID, userID, notes)
    
    // 2. Capture original timestamp
    var membership models.CoinSetMembership
    db.Where("coin_id = ? AND set_id = ?", coinID, setID).First(&membership)
    originalAddedAt := membership.AddedAt
    
    // 3. Update main entity
    coinRepo.Update(coin, &models.Coin{Name: "Updated"})
    
    // 4. Assert: timestamp unchanged
    db.Where("coin_id = ? AND set_id = ?", coinID, setID).First(&membership)
    if !membership.AddedAt.Equal(originalAddedAt) {
        t.Error("AddedAt changed; Omit() failed")
    }
}
```

### 3. Test Pattern: Verify Omit() Effectiveness

```go
func TestUpdate_IgnoresAssociationField(t *testing.T) {
    // 1. Create coin with membership to set1
    setRepo.AddCoinToSet(coinID, set1.ID, userID, "")
    
    // 2. Attempt update with set2 in payload (should be ignored)
    updates := &models.Coin{
        Name: "Updated",
        Sets: []CoinSet{*set2},  // ← Should be ignored by Omit("Sets")
    }
    coinRepo.Update(coin, updates)
    
    // 3. Assert: still only set1, set2 not added
    var count int64
    db.Model(&models.CoinSetMembership{}).Where("coin_id = ?", coinID).Count(&count)
    if count != 1 {
        t.Error("membership count changed; Omit() failed")
    }
    
    var membership models.CoinSetMembership
    if err := db.Where("coin_id = ? AND set_id = ?", coinID, set1.ID).First(&membership).Error; err != nil {
        t.Error("original membership disappeared")
    }
}
```

## Examples

See `src/api/repository/coin_repository_test.go`:
- `TestCoinRepository_Update_PreservesSets` (lines 333-374)
- `TestCoinRepository_Update_WithSetsField` (lines 376-430)

Join table model:
- `CoinSetMembership` (models/set.go:38-43) — has `AddedAt` field

Repository implementations:
- `CoinRepository.Update` (coin_repository.go:337-344) — uses `Omit("Tags", "Sets")`
- `SetRepository.AddCoinToSet` (set_repository.go:82-108) — populates `AddedAt` via `time.Now()`

## Anti-Patterns

### ❌ Missing Join Table in Test Setup
```go
// BAD: AutoMigrate only includes entity models
db.AutoMigrate(&models.Coin{}, &models.CoinSet{})
// JOIN table coin_set_memberships is missing → tests fail silently
```

### ❌ Not Capturing Original Timestamp
```go
// BAD: Can't prove preservation without baseline
coinRepo.Update(coin, updates)
var membership models.CoinSetMembership
db.First(&membership)
if membership.AddedAt.IsZero() {  // ← Only catches NULL, not changes
    t.Error("timestamp zero")
}
```

### ❌ Naive Association Replacement in Production Code
```go
// BAD: GORM replaces memberships without custom fields
coin.Sets = []CoinSet{newSet}
db.Model(&coin).Updates(coin)  // ← Deletes old, inserts with NULL AddedAt

// GOOD: Omit associations from update path
db.Model(&coin).Omit("Tags", "Sets").Updates(coin)
```

### ❌ Testing Only Happy Path
```go
// BAD: Only tests that membership exists
if count := countMemberships(coinID); count == 0 {
    t.Error("membership missing")
}

// GOOD: Test that AddedAt is non-zero AND unchanged
if membership.AddedAt.IsZero() {
    t.Error("AddedAt zero")
}
if !membership.AddedAt.Equal(originalAddedAt) {
    t.Error("AddedAt changed")
}
```

## Notes

- This pattern applies to **any** join table with custom fields requiring manual population
- If GORM adds auto-population support for custom join fields in future versions, these tests will prove migration safety
- Regression tests must verify BOTH non-zero timestamps AND preservation across updates
