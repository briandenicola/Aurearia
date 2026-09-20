package database

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func requiredBackfillFixture(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	for _, sql := range []string{
		"CREATE TABLE api_keys (capabilities TEXT)",
		"INSERT INTO api_keys VALUES (NULL), (''), ('write')",
		"CREATE TABLE auction_lots (source TEXT, source_url TEXT, numis_bids_url TEXT)",
		"INSERT INTO auction_lots VALUES (NULL, '', 'https://example.test/lot')",
		"CREATE TABLE featured_coins (source_type TEXT)",
		"INSERT INTO featured_coins VALUES (NULL)",
		"CREATE TABLE users (coin_of_day_include_wishlist INTEGER)",
		"INSERT INTO users VALUES (NULL), (0)",
		"CREATE TABLE deep_identification_jobs (status TEXT, expires_at datetime)",
		"INSERT INTO deep_identification_jobs VALUES ('completed', NULL), ('partial', NULL), ('running', NULL)",
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestRequiredBackfillsLegacyAndIdempotent(t *testing.T) {
	db := requiredBackfillFixture(t)
	for range 2 {
		if err := backfillRequiredDefaults(db); err != nil {
			t.Fatal(err)
		}
		for _, check := range []struct {
			sql  string
			want int64
		}{
			{"SELECT COUNT(*) FROM api_keys WHERE capabilities='read'", 2},
			{"SELECT COUNT(*) FROM api_keys WHERE capabilities='write'", 1},
			{"SELECT COUNT(*) FROM auction_lots WHERE source='numisbids' AND source_url=numis_bids_url", 1},
			{"SELECT COUNT(*) FROM featured_coins WHERE source_type='owned'", 1},
			{"SELECT COUNT(*) FROM users WHERE coin_of_day_include_wishlist=1", 1},
			{"SELECT COUNT(*) FROM users WHERE coin_of_day_include_wishlist=0", 1},
			{"SELECT COUNT(*) FROM deep_identification_jobs WHERE status IN ('completed','partial') AND expires_at IS NOT NULL", 2},
			{"SELECT COUNT(*) FROM deep_identification_jobs WHERE status='running' AND expires_at IS NULL", 1},
		} {
			var count int64
			if err := db.Raw(check.sql).Scan(&count).Error; err != nil || count != check.want {
				t.Fatalf("%s: %d, %v", check.sql, count, err)
			}
		}
	}
}

func TestRequiredBackfillFailuresStopSequence(t *testing.T) {
	for index, name := range []string{"API key capabilities", "auction source", "auction source URL", "featured coin source", "coin of day wishlist preference", "completed identification retention"} {
		t.Run(name, func(t *testing.T) {
			db := requiredBackfillFixture(t)
			executed := 0
			if err := db.Callback().Raw().Before("gorm:raw").Register("inject", func(tx *gorm.DB) {
				executed++
				if executed == index+1 {
					tx.AddError(errors.New("injected backfill failure"))
				}
			}); err != nil {
				t.Fatal(err)
			}
			err := backfillRequiredDefaults(db)
			if err == nil || !strings.Contains(err.Error(), name) || executed != index+1 {
				t.Fatalf("err=%v executed=%d", err, executed)
			}
		})
	}
}

func TestRequiredBackfillFailureBlocksStartup(t *testing.T) {
	if path := os.Getenv("AUREARIA_BACKFILL_TEST_DB"); path != "" {
		Connect(path)
		fmt.Println("TEST_READY")
		return
	}
	dbPath := filepath.Join(t.TempDir(), "startup.db")
	Connect(dbPath)
	sqlDB, _ := DB.DB()
	t.Cleanup(func() { sqlDB.Close() })
	user := models.User{Username: "fixture", Email: "fixture@example.test"}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		"INSERT INTO api_keys (user_id, key_hash, key_prefix, name, capabilities) VALUES (1, 'fixture-hash', 'fixture', 'fixture', '')",
		"CREATE TRIGGER reject_backfill BEFORE UPDATE OF capabilities ON api_keys BEGIN SELECT RAISE(FAIL, 'injected startup failure'); END",
	} {
		if err := DB.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestRequiredBackfillFailureBlocksStartup$")
	cmd.Env = append(os.Environ(), "AUREARIA_BACKFILL_TEST_DB="+dbPath)
	output, err := cmd.CombinedOutput()
	if err == nil || strings.Contains(string(output), "TEST_READY") || !strings.Contains(string(output), "Failed required database backfill: API key capabilities") {
		t.Fatalf("startup did not fail at required backfill: %v\n%s", err, output)
	}
}
