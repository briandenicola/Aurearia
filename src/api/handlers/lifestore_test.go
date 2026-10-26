package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/middleware"
	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

const lifestoreTestSecret = "lifestore-records-test-secret-0123456789"

type lifestoreFixture struct {
	router  *gin.Engine
	db      *gorm.DB
	owner   models.User
	other   models.User
	readKey string
	trajan  models.Coin
}

func setupLifestoreRecords(t *testing.T) *lifestoreFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.ApiKey{}, &models.StorageLocation{}, &models.MintLocation{}, &models.Coin{},
		&models.CoinImage{}, &models.CoinReference{}, &models.Tag{}, &models.CoinSet{}, &models.CoinSetMembership{},
		&models.CoinJournal{}, &models.CoinValueHistory{}); err != nil {
		t.Fatal(err)
	}
	f := &lifestoreFixture{db: db, readKey: "ak_lifestore_read"}
	f.owner = models.User{Username: "brian", Email: "brian@example.test", PasswordHash: "x", Role: models.RoleUser}
	f.other = models.User{Username: "guest", Email: "guest@example.test", PasswordHash: "x", Role: models.RoleUser}
	for _, u := range []*models.User{&f.owner, &f.other} {
		if err := db.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	key := models.ApiKey{UserID: f.owner.ID, KeyHash: services.HashAPIKey(f.readKey, lifestoreTestSecret), KeyPrefix: "ak_test", Name: "lifestore", Capabilities: "read"}
	if err := db.Create(&key).Error; err != nil {
		t.Fatal(err)
	}

	tray := models.StorageLocation{UserID: f.owner.ID, Name: "Tray A", Type: "standard"}
	db.Create(&tray)
	price, value := 120.0, 180.5
	bought := time.Date(2024, 5, 4, 0, 0, 0, 0, time.UTC)
	f.trajan = models.Coin{UserID: f.owner.ID, Name: "Denarius of Trajan", Category: models.CategoryRoman, Denomination: "Denarius",
		Ruler: "Trajan", PurchasePrice: &price, PurchaseDate: &bought, CurrentValue: &value, StorageLocationID: &tray.ID, Notes: "Dacian triumph reverse"}
	wishlist := models.Coin{UserID: f.owner.ID, Name: "Aureus of Hadrian", Category: models.CategoryRoman, IsWishlist: true}
	sold := models.Coin{UserID: f.owner.ID, Name: "Tetradrachm", Category: models.CategoryGreek, IsSold: true}
	theirs := models.Coin{UserID: f.other.ID, Name: "Someone else's coin", Category: models.CategoryOther}
	for _, c := range []*models.Coin{&f.trajan, &wishlist, &sold, &theirs} {
		if err := db.Create(c).Error; err != nil {
			t.Fatal(err)
		}
	}
	tag := models.Tag{UserID: f.owner.ID, Name: "favorites"}
	db.Create(&tag)
	if err := db.Model(&f.trajan).Association("Tags").Append(&tag); err != nil {
		t.Fatal(err)
	}
	set := models.CoinSet{UserID: f.owner.ID, Name: "Twelve Caesars"}
	db.Create(&set)
	db.Create(&models.CoinSetMembership{SetID: set.ID, CoinID: f.trajan.ID, AddedAt: time.Now()})
	db.Create(&models.CoinReference{CoinID: f.trajan.ID, Catalog: "RIC", Volume: "II", Number: "98"})
	db.Create(&models.CoinImage{CoinID: f.trajan.ID, FilePath: "obv.jpg", ImageType: models.ImageTypeObverse})
	db.Create(&models.CoinImage{CoinID: f.trajan.ID, FilePath: "card.jpg", ImageType: models.ImageTypeCard})
	db.Create(&models.CoinJournal{CoinID: f.trajan.ID, UserID: f.owner.ID, Entry: "Bought at the spring show", CreatedAt: bought})
	db.Create(&models.CoinValueHistory{CoinID: f.trajan.ID, UserID: f.owner.ID, Value: 150, Confidence: "medium", Source: "manual", RecordedAt: bought})
	db.Create(&models.CoinValueHistory{CoinID: f.trajan.ID, UserID: f.owner.ID, Value: 180.5, Confidence: "high", Source: "ai_scheduled", RecordedAt: bought.AddDate(1, 0, 0)})
	db.Create(&models.CoinJournal{CoinID: theirs.ID, UserID: f.other.ID, Entry: "not yours"})

	r := gin.New()
	protected := r.Group("/api")
	protected.Use(middleware.AuthRequired(lifestoreTestSecret, repository.NewApiKeyRepository(db)))
	protected.Use(middleware.RequireAPIKeyWriteForChanges())
	protected.GET("/lifestore/records", NewLifestoreHandler(services.NewLifestoreRecordsService(repository.NewCoinRepository(db))).ListRecords)
	f.router = r
	return f
}

func (f *lifestoreFixture) get(t *testing.T, query string) (int, services.LifestoreRecordsPage) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/lifestore/records"+query, nil)
	req.Header.Set("X-API-Key", f.readKey)
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	var page services.LifestoreRecordsPage
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
			t.Fatalf("decode: %v: %s", err, w.Body.String())
		}
	}
	return w.Code, page
}

func TestLifestoreRecords_ReadKeyListsOnlyTheOwnersCoinsWithWhatBelongsToThem(t *testing.T) {
	f := setupLifestoreRecords(t)
	code, page := f.get(t, "")
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if page.Schema != services.LifestoreRecordsSchema || page.Next != "" {
		t.Fatalf("page = %+v", page)
	}
	if len(page.Records) != 3 {
		t.Fatalf("records = %d, want the owner's 3 (collection, wishlist, sold)", len(page.Records))
	}
	for _, r := range page.Records {
		if r.Data.Name == "Someone else's coin" {
			t.Fatal("another user's coin is in the feed")
		}
	}
	c := page.Records[0]
	if c.Kind != "coin" || c.ID != itoaUint(f.trajan.ID) || c.Data.Ruler != "Trajan" || *c.Data.PurchasePrice != 120 || *c.Data.CurrentValue != 180.5 {
		t.Fatalf("first record = %+v", c)
	}
	d := c.Data
	if d.StorageLocation != "Tray A" || len(d.Tags) != 1 || d.Tags[0] != "favorites" || len(d.Sets) != 1 || d.Sets[0] != "Twelve Caesars" {
		t.Fatalf("location, tags, sets = %q %v %v", d.StorageLocation, d.Tags, d.Sets)
	}
	if len(d.References) != 1 || d.References[0].Catalog != "RIC" || d.References[0].Number != "98" {
		t.Fatalf("references = %+v", d.References)
	}
	if d.Images != 1 {
		t.Fatalf("images = %d; the private attribution card must not be counted", d.Images)
	}
	if len(d.Journal) != 1 || d.Journal[0].Entry != "Bought at the spring show" {
		t.Fatalf("journal = %+v", d.Journal)
	}
	if len(d.ValueHistory) != 2 || d.ValueHistory[0].Value != 150 || d.ValueHistory[1].Source != "ai_scheduled" {
		t.Fatalf("value history = %+v", d.ValueHistory)
	}
	if !page.Records[1].Data.IsWishlist || !page.Records[2].Data.IsSold {
		t.Fatalf("wishlist and sold flags = %+v %+v", page.Records[1].Data, page.Records[2].Data)
	}
	// Empty collections are lists, not null, so a reader hashes the same shape.
	w := page.Records[1].Data
	if w.Tags == nil || w.Sets == nil || w.References == nil || w.Journal == nil || w.ValueHistory == nil {
		t.Fatalf("empty children are null: %+v", w)
	}
}

func TestLifestoreRecords_PagesCoverEveryCoinOnce(t *testing.T) {
	f := setupLifestoreRecords(t)
	code, first := f.get(t, "?limit=2")
	if code != http.StatusOK || len(first.Records) != 2 || first.Next == "" {
		t.Fatalf("first page = %d %+v", code, first)
	}
	_, second := f.get(t, "?limit=2&page="+first.Next)
	if len(second.Records) != 1 || second.Next != "" {
		t.Fatalf("second page = %+v", second)
	}
	seen := map[string]bool{}
	for _, r := range append(first.Records, second.Records...) {
		if seen[r.ID] {
			t.Fatalf("coin %s listed twice", r.ID)
		}
		seen[r.ID] = true
	}
	if len(seen) != 3 {
		t.Fatalf("paged through %d coins, want 3", len(seen))
	}
}

func TestLifestoreRecords_RejectsAPageTokenItDidNotGiveOut(t *testing.T) {
	f := setupLifestoreRecords(t)
	for _, token := range []string{"x", "coin:", "coin:-1", "set:4", "coin:99999999999"} {
		if code, _ := f.get(t, "?page="+token); code != http.StatusBadRequest {
			t.Errorf("page=%q = %d, want 400", token, code)
		}
	}
}

func itoaUint(n uint) string {
	b, _ := json.Marshal(n)
	return string(b)
}
