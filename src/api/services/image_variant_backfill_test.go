package services

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// makeLargeTestPNG is a noisy PNG well over the 512 KB inline-original limit,
// like a full-size phone photo saved as PNG.
func makeLargeTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 900, 700))
	seed := uint32(7)
	for y := 0; y < 700; y++ {
		for x := 0; x < 900; x++ {
			seed = seed*1664525 + 1013904223
			img.SetRGBA(x, y, color.RGBA{R: uint8(seed >> 24), G: uint8(seed >> 16), B: uint8(seed >> 8), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if buf.Len() <= maxInlineOriginalBytes {
		t.Fatalf("test PNG is %d bytes; it must be over %d to reproduce the bug", buf.Len(), maxInlineOriginalBytes)
	}
	return buf.Bytes()
}

func writeUpload(t *testing.T, uploadDir, relPath string, data []byte) {
	t.Helper()
	full := filepath.Join(uploadDir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func decodedBounds(t *testing.T, path string) image.Rectangle {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("variant %s missing: %v", path, err)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("variant %s is not an image: %v", path, err)
	}
	return img.Bounds()
}

func TestEnsureImageVariantsMakesMissingVariantsOnce(t *testing.T) {
	uploadDir := t.TempDir()
	writeUpload(t, uploadDir, "coin-1/100-obverse.png", makeLargeTestPNG(t))

	made, err := ensureImageVariants(uploadDir, "coin-1/100-obverse.png")
	if err != nil || !made {
		t.Fatalf("made=%v err=%v, want the variants made", made, err)
	}
	if b := decodedBounds(t, filepath.Join(uploadDir, "coin-1", "100-obverse_thumb.jpg")); b.Dx() > variantThumbMaxDim || b.Dy() > variantThumbMaxDim {
		t.Fatalf("thumb is %v, want within %d px", b, variantThumbMaxDim)
	}
	if b := decodedBounds(t, filepath.Join(uploadDir, "coin-1", "100-obverse_medium.jpg")); b.Dx() > variantMediumMaxDim || b.Dy() > variantMediumMaxDim {
		t.Fatalf("medium is %v, want within %d px", b, variantMediumMaxDim)
	}

	made, err = ensureImageVariants(uploadDir, "coin-1/100-obverse.png")
	if err != nil || made {
		t.Fatalf("second pass made=%v err=%v, want nothing to do", made, err)
	}
}

func TestEnsureImageVariantsMakesOnlyTheMissingOne(t *testing.T) {
	uploadDir := t.TempDir()
	writeUpload(t, uploadDir, "coin-1/1-obverse.png", makeLargeTestPNG(t))
	writeUpload(t, uploadDir, "coin-1/1-obverse_thumb.jpg", []byte("kept as it is"))

	if made, err := ensureImageVariants(uploadDir, "coin-1/1-obverse.png"); err != nil || !made {
		t.Fatalf("made=%v err=%v", made, err)
	}
	if got, _ := os.ReadFile(filepath.Join(uploadDir, "coin-1", "1-obverse_thumb.jpg")); string(got) != "kept as it is" {
		t.Fatalf("the existing thumb was rewritten")
	}
	decodedBounds(t, filepath.Join(uploadDir, "coin-1", "1-obverse_medium.jpg"))
}

func TestEnsureImageVariantsRefusesWhatItCannotResize(t *testing.T) {
	uploadDir := t.TempDir()
	writeUpload(t, uploadDir, "coin-1/broken.png", []byte("not an image"))
	outside := filepath.Join(filepath.Dir(uploadDir), "outside.png")
	if err := os.WriteFile(outside, makeLargeTestPNG(t), 0644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(outside) })

	for _, path := range []string{"coin-1/broken.png", "coin-1/missing.png", "../outside.png", "/etc/passwd", ""} {
		if made, err := ensureImageVariants(uploadDir, path); err == nil || made {
			t.Fatalf("%q: made=%v err=%v, want an error", path, made, err)
		}
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(uploadDir), "outside_thumb.jpg")); err == nil {
		t.Fatal("a variant was written outside the upload directory")
	}
}

func newImageVariantDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:image_variants_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Coin{}, &models.CoinImage{}, &models.AppSetting{}); err != nil {
		t.Fatal(err)
	}
	return db
}

// The bug: a large photo uploaded before variants existed never reached the
// MCP coin viewer. After one pass of the scheduler it does.
func TestImageVariantPassMakesLegacyPhotosShowInTheCoinViewer(t *testing.T) {
	db := newImageVariantDB(t)
	uploadDir := t.TempDir()
	coin := models.Coin{UserID: 10, Name: "Faustina"}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatal(err)
	}
	photos := []models.CoinImage{
		{CoinID: coin.ID, FilePath: fmt.Sprintf("coin-%d/1-obverse.png", coin.ID), ImageType: models.ImageTypeObverse, IsPrimary: true},
		{CoinID: coin.ID, FilePath: fmt.Sprintf("coin-%d/2-reverse.png", coin.ID), ImageType: models.ImageTypeReverse},
		{CoinID: coin.ID, FilePath: fmt.Sprintf("coin-%d/3-other.png", coin.ID), ImageType: models.ImageTypeOther},
		{CoinID: coin.ID, FilePath: fmt.Sprintf("coin-%d/4-card.png", coin.ID), ImageType: models.ImageTypeCard},
	}
	writeUpload(t, uploadDir, photos[0].FilePath, makeLargeTestPNG(t))
	writeUpload(t, uploadDir, photos[1].FilePath, makeLargeTestPNG(t))
	writeUpload(t, uploadDir, photos[2].FilePath, []byte("corrupt"))
	writeUpload(t, uploadDir, photos[3].FilePath, makeLargeTestPNG(t))
	if err := db.Create(&photos).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.NewImageRepository(db)
	imageSvc := NewImageService(repo, uploadDir)
	limits := InlineCoinImageLimits{Size: VariantSizeThumb, PerCoin: 1, MaxTotalBytes: mcpMaxInlineImageBytes}

	before, _, err := imageSvc.LoadOwnedCoinImages(10, []uint{coin.ID}, limits)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 0 {
		t.Fatalf("before the pass the viewer got %d images; the large original should be skipped", len(before))
	}

	scheduler := NewImageVariantScheduler(repo, nil, uploadDir, NewSettingsService(repository.NewSettingsRepository(db)), NewLogger(100))
	cycle := scheduler.RunCycle()
	if cycle.Checked != 3 || cycle.Made != 2 || cycle.Failed != 1 || len(cycle.Failures) != 1 ||
		!strings.Contains(cycle.Failures[0], "3-other.png") {
		t.Fatalf("cycle=%+v, want 3 photos checked (the card left alone), 2 made, the corrupt one named", cycle)
	}
	if _, err := os.Stat(filepath.Join(uploadDir, fmt.Sprintf("coin-%d", coin.ID), "4-card_thumb.jpg")); err == nil {
		t.Fatal("an attribution card was given variants")
	}

	after, _, err := imageSvc.LoadOwnedCoinImages(10, []uint{coin.ID}, limits)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 || after[0].MIMEType != "image/jpeg" || after[0].ImageID != photos[0].ID {
		t.Fatalf("after the pass the viewer got %+v, want the primary photo's thumb", after)
	}

	again := scheduler.RunCycle()
	if again.Made != 0 || again.Checked != 3 {
		t.Fatalf("second pass=%+v, want nothing left to make", again)
	}
}

type fakePhotoLister struct {
	photos []models.CoinImage
	calls  int
}

func (f *fakePhotoLister) ListPhotosAfter(afterID uint, limit int) ([]models.CoinImage, error) {
	f.calls++
	out := []models.CoinImage{}
	for _, p := range f.photos {
		if p.ID > afterID && len(out) < limit {
			out = append(out, p)
		}
	}
	return out, nil
}

func TestImageVariantPassWalksEveryBatchAndCapsTheFailuresNamed(t *testing.T) {
	lister := &fakePhotoLister{}
	for i := 1; i <= imageVariantBatchSize*2+50; i++ {
		lister.photos = append(lister.photos, models.CoinImage{ID: uint(i), CoinID: 1, FilePath: fmt.Sprintf("coin-1/%d.png", i)})
	}
	db := newImageVariantDB(t)
	scheduler := NewImageVariantScheduler(lister, nil, t.TempDir(), NewSettingsService(repository.NewSettingsRepository(db)), NewLogger(100))
	cycle := scheduler.RunCycle()
	if cycle.Checked != len(lister.photos) || cycle.Failed != len(lister.photos) || len(cycle.Failures) != imageVariantFailuresLogged {
		t.Fatalf("cycle=%+v, want every photo checked and %d failures named", cycle, imageVariantFailuresLogged)
	}
	if lister.calls != 3 {
		t.Fatalf("listed %d times, want 3 batches", lister.calls)
	}
	if status := scheduler.GetStatus(); status.Name != "Image Variants" || !status.Enabled || status.IsRunning {
		t.Fatalf("status=%+v", status)
	}
}

// Quick Capture saves only originals; a promoted draft's photos get variants.
func TestQuickCapturePromotionMakesVariantsForTheCoinsPhotos(t *testing.T) {
	uploadDir := t.TempDir()
	svc, _ := newQuickCaptureServiceAndDBForTest(t, uploadDir)
	draft, err := svc.CreateDraft(CreateQuickCaptureDraftInput{
		UserID: 1, WorkingTitle: "Captured", Era: string(models.EraAncient),
		Images: []QuickCaptureImageUpload{{Filename: "obverse.png", Data: makeLargeTestPNG(t), ImageType: "obverse", IsPrimary: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(draft.Images) != 1 {
		t.Fatalf("draft images=%d", len(draft.Images))
	}
	rel := draft.Images[0].FilePath
	thumb, medium := variantNames(filepath.FromSlash(rel))
	if _, err := os.Stat(filepath.Join(uploadDir, thumb)); err == nil {
		t.Fatal("a draft photo already had a thumb; the test would not show the fix")
	}

	if _, err := svc.PromoteDraft(1, draft.ID, PromoteDraftInput{Confirm: true, Target: QuickCapturePromotionTargetCollection}); err != nil {
		t.Fatal(err)
	}
	decodedBounds(t, filepath.Join(uploadDir, thumb))
	decodedBounds(t, filepath.Join(uploadDir, medium))
}

func newSchedulerWithSettings(t *testing.T, lister imageVariantPhotoLister, uploadDir string, settings map[string]string) (*ImageVariantScheduler, *SettingsService) {
	t.Helper()
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(newImageVariantDB(t)))
	for k, v := range settings {
		if err := settingsSvc.SetSetting(k, v); err != nil {
			t.Fatal(err)
		}
	}
	return NewImageVariantScheduler(lister, nil, uploadDir, settingsSvc, NewLogger(100)), settingsSvc
}

func TestImageVariantCheckTurnedOffDoesNothing(t *testing.T) {
	lister := &fakePhotoLister{photos: []models.CoinImage{{ID: 1, FilePath: "coin-1/1.png"}}}
	scheduler, _ := newSchedulerWithSettings(t, lister, t.TempDir(), map[string]string{SettingImageVariantCheckEnabled: "false"})
	if cycle := scheduler.RunCycle(); !cycle.Skipped || cycle.Checked != 0 || lister.calls != 0 {
		t.Fatalf("cycle=%+v calls=%d, want the pass skipped", cycle, lister.calls)
	}
	if err := scheduler.RunNow(); err != nil || lister.calls != 0 {
		t.Fatalf("RunNow ran a disabled check: err=%v calls=%d", err, lister.calls)
	}
	if scheduler.GetStatus().Enabled {
		t.Fatal("status says enabled")
	}
}

func TestImageVariantSchedulerStopsBetweenPhotosAndStopsTwiceSafely(t *testing.T) {
	lister := &fakePhotoLister{photos: []models.CoinImage{{ID: 1, FilePath: "coin-1/1.png"}, {ID: 2, FilePath: "coin-1/2.png"}}}
	scheduler, _ := newSchedulerWithSettings(t, lister, t.TempDir(), nil)
	scheduler.Stop()
	scheduler.Stop()
	if cycle := scheduler.RunCycle(); !cycle.Stopped || cycle.Checked != 0 {
		t.Fatalf("cycle=%+v, want it to stop before the first photo", cycle)
	}
	done := make(chan struct{})
	go func() {
		scheduler.Start()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after Stop")
	}
}

func TestImageVariantPassesDoNotOverlap(t *testing.T) {
	uploadDir := t.TempDir()
	lister := &fakePhotoLister{}
	for i := 1; i <= 3; i++ {
		rel := fmt.Sprintf("coin-1/%d.png", i)
		writeUpload(t, uploadDir, rel, makeLargeTestPNG(t))
		lister.photos = append(lister.photos, models.CoinImage{ID: uint(i), CoinID: 1, FilePath: rel})
	}
	scheduler, _ := newSchedulerWithSettings(t, lister, uploadDir, nil)
	results := make(chan ImageVariantCycle, 2)
	for i := 0; i < 2; i++ {
		go func() { results <- scheduler.RunCycle() }()
	}
	a, b := <-results, <-results
	if a.Made+b.Made != 3 || a.Checked != 3 || b.Checked != 3 {
		t.Fatalf("passes=%+v %+v, want each photo resized once across both", a, b)
	}
}

func TestImageVariantCheckRunsAtTheConfiguredTime(t *testing.T) {
	scheduler, settingsSvc := newSchedulerWithSettings(t, &fakePhotoLister{}, t.TempDir(), map[string]string{SettingScheduleTimezone: "UTC"})
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	if got := scheduler.nextRun(now); !got.Equal(time.Date(2026, 11, 19, 3, 17, 0, 0, time.UTC)) {
		t.Fatalf("default next run %v, want 03:17 tomorrow", got)
	}
	if err := settingsSvc.SetSetting(SettingImageVariantCheckStartTime, "13:05"); err != nil {
		t.Fatal(err)
	}
	if got := scheduler.nextRun(now); !got.Equal(time.Date(2026, 11, 18, 13, 5, 0, 0, time.UTC)) {
		t.Fatalf("configured next run %v, want 13:05 today", got)
	}
	if err := settingsSvc.SetSetting(SettingImageVariantCheckStartTime, "25:99"); err != nil {
		t.Fatal(err)
	}
	if got := scheduler.nextRun(now); !got.Equal(time.Date(2026, 11, 19, 3, 17, 0, 0, time.UTC)) {
		t.Fatalf("invalid start time gave %v, want the 03:17 default", got)
	}
}
