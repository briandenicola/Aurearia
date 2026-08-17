package services

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"sync"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
)

// T105: TestDeepIdentificationService_ServiceTest_ConcurrentSameCoinDifferentImages
// closes the concurrency gap left by
// TestDeepIdentificationService_CreateJobFromIntake_DuplicateSubmitIsIdempotent
// (deep_identification_service_test.go:1110-ish), which only ever calls
// CreateJobFromIntake twice *sequentially* with byte-identical images. That
// proves nothing about what happens when two goroutines race into
// CreateJobFromIntake for the *same coin* with *different* image bytes -
// exactly the shape of bug this feature has repeatedly shipped (SQLITE_BUSY
// from a read->write lock upgrade inside a deferred transaction, per this
// feature's history).
//
// Decided semantics (derived from reading the current, post-884f709 code,
// not invented): CreateJobFromIntake computes InputFingerprint from the
// actual uploaded image bytes (ComputeInputFingerprint, keyed on
// sha256(obverse)/sha256(reverse) among other fields) *before* creating any
// row, then serializes the entire create-or-reuse decision under
// svc.intakeMu.Lock() (see CreateJobFromIntake's own comment: "Hold the
// intake write lock across row creation and artifact persistence so neither
// the wake signal nor the polling fallback can claim incomplete evidence").
// Two goroutines submitting different image bytes therefore never race the
// database directly; intakeMu forces one submission fully to finish before
// the other's StartJob dedupe check runs. Given that ordering, StartJob's
// contract from its own doc comment is:
//   - distinct fingerprints, both under MaxActivePerUser -> both admitted as
//     genuinely separate jobs;
//   - distinct fingerprints, but the user is already at MaxActivePerUser ->
//     the losing submission is refused with ErrDeepJobAtCapacity (surfaced
//     to the handler as 409 job_at_capacity), never handed an unrelated
//     job's identity. This is a deliberate, approved contract change
//     (2026-08-17): the prior behavior silently returned the user's other
//     active job with reused=true even though its fingerprint did not
//     match, which meant a second coin's submission could receive the
//     first coin's report presented as its own answer. The assertion below
//     was rewritten to require the 409/sentinel, not weakened to make a
//     build pass - a matching fingerprint at capacity still legitimately
//     dedupes with reused=true (see the third sub-test).
// Both halves of that contract are asserted below under real goroutines so
// a regression in either the fingerprint comparison or the MaxActivePerUser
// gate is caught by -race and by a plain failure, not just logically implied
// by sequential calls.
func TestDeepIdentificationService_ServiceTest_ConcurrentSameCoinDifferentImages(t *testing.T) {
	t.Run("distinct fingerprints under the per-user cap are both admitted as separate jobs", func(t *testing.T) {
		svc, db, _ := newDeepIdentificationServiceTestDeps(t)
		user := models.User{Username: "concurrent-diff-owner", Email: "concurrent-diff-owner@example.com", PasswordHash: "x"}
		if err := db.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
		coin := models.Coin{UserID: user.ID, Name: "Racing Denarius"}
		if err := db.Create(&coin).Error; err != nil {
			t.Fatal(err)
		}
		enableDeepIdentification(t, svc, map[string]string{
			SettingDeepIdentificationMaxActivePerUser: "5",
			SettingDeepIdentificationQueueDepth:       "10",
		})

		imgA := coloredPNGBytes(t, 10, 20, 30)
		imgB := coloredPNGBytes(t, 200, 210, 220)

		results := make([]struct {
			job    *models.DeepIdentificationJob
			reused bool
			err    error
		}, 2)

		var wg sync.WaitGroup
		start := make(chan struct{})
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			job, reused, err := svc.CreateJobFromIntake(CreateJobInput{
				UserID: user.ID, CoinID: &coin.ID,
				ObverseBytes: imgA, ObverseFilename: "a-o.png",
				ReverseBytes: imgA, ReverseFilename: "a-r.png",
			})
			results[0] = struct {
				job    *models.DeepIdentificationJob
				reused bool
				err    error
			}{job, reused, err}
		}()
		go func() {
			defer wg.Done()
			<-start
			job, reused, err := svc.CreateJobFromIntake(CreateJobInput{
				UserID: user.ID, CoinID: &coin.ID,
				ObverseBytes: imgB, ObverseFilename: "b-o.png",
				ReverseBytes: imgB, ReverseFilename: "b-r.png",
			})
			results[1] = struct {
				job    *models.DeepIdentificationJob
				reused bool
				err    error
			}{job, reused, err}
		}()
		close(start)
		wg.Wait()

		for i, r := range results {
			if r.err != nil {
				t.Fatalf("submission %d failed: %v", i, r.err)
			}
			if r.job == nil {
				t.Fatalf("submission %d returned a nil job", i)
			}
			if r.reused {
				t.Fatalf("submission %d unexpectedly reused an existing job; distinct image bytes must produce distinct fingerprints under the per-user cap", i)
			}
		}
		if results[0].job.ID == results[1].job.ID {
			t.Fatalf("expected two distinct job rows for two distinct image submissions, got the same id %d twice", results[0].job.ID)
		}
		if results[0].job.InputFingerprint == results[1].job.InputFingerprint {
			t.Fatalf("expected distinct fingerprints for distinct image bytes, both hashed to %q", results[0].job.InputFingerprint)
		}

		var jobCount int64
		if err := db.Model(&models.DeepIdentificationJob{}).Where("user_id = ?", user.ID).Count(&jobCount).Error; err != nil {
			t.Fatal(err)
		}
		if jobCount != 2 {
			t.Fatalf("expected exactly 2 job rows for two concurrently admitted distinct submissions, got %d", jobCount)
		}
	})

	t.Run("distinct fingerprints at the per-user cap: one is admitted, the other is refused with ErrDeepJobAtCapacity", func(t *testing.T) {
		svc, db, _ := newDeepIdentificationServiceTestDeps(t)
		user := models.User{Username: "concurrent-bound-owner", Email: "concurrent-bound-owner@example.com", PasswordHash: "x"}
		if err := db.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
		coin := models.Coin{UserID: user.ID, Name: "Bounded Denarius"}
		if err := db.Create(&coin).Error; err != nil {
			t.Fatal(err)
		}
		enableDeepIdentification(t, svc, map[string]string{
			SettingDeepIdentificationMaxActivePerUser: "1",
			SettingDeepIdentificationQueueDepth:       "10",
		})

		imgA := coloredPNGBytes(t, 10, 20, 30)
		imgB := coloredPNGBytes(t, 200, 210, 220)

		results := make([]struct {
			job    *models.DeepIdentificationJob
			reused bool
			err    error
		}, 2)

		var wg sync.WaitGroup
		start := make(chan struct{})
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			job, reused, err := svc.CreateJobFromIntake(CreateJobInput{
				UserID: user.ID, CoinID: &coin.ID,
				ObverseBytes: imgA, ObverseFilename: "a-o.png",
				ReverseBytes: imgA, ReverseFilename: "a-r.png",
			})
			results[0] = struct {
				job    *models.DeepIdentificationJob
				reused bool
				err    error
			}{job, reused, err}
		}()
		go func() {
			defer wg.Done()
			<-start
			job, reused, err := svc.CreateJobFromIntake(CreateJobInput{
				UserID: user.ID, CoinID: &coin.ID,
				ObverseBytes: imgB, ObverseFilename: "b-o.png",
				ReverseBytes: imgB, ReverseFilename: "b-r.png",
			})
			results[1] = struct {
				job    *models.DeepIdentificationJob
				reused bool
				err    error
			}{job, reused, err}
		}()
		close(start)
		wg.Wait()

		// Exactly one submission must have created the job; the other must
		// be refused outright with ErrDeepJobAtCapacity (409 job_at_capacity
		// at the handler), regardless of its own (different) fingerprint. It
		// must NOT be silently handed the winner's job with reused=true -
		// that was the wrong-job-returned defect this test now guards
		// against (approved breaking change, 2026-08-17).
		var admittedIdx, refusedIdx = -1, -1
		for i, r := range results {
			switch {
			case r.err == nil:
				admittedIdx = i
			case errors.Is(r.err, ErrDeepJobAtCapacity):
				refusedIdx = i
			default:
				t.Fatalf("submission %d returned unexpected error: %v", i, r.err)
			}
		}
		if admittedIdx == -1 {
			t.Fatalf("expected exactly one of the two racing submissions to be admitted, neither was: %+v", results)
		}
		if refusedIdx == -1 {
			t.Fatalf("expected the losing submission to be refused with ErrDeepJobAtCapacity, got: %+v", results)
		}
		admitted := results[admittedIdx]
		if admitted.job == nil {
			t.Fatalf("admitted submission %d returned a nil job", admittedIdx)
		}
		if admitted.reused {
			t.Fatalf("admitted submission %d must not be marked reused - it is the first job created for this user, not a duplicate", admittedIdx)
		}
		refused := results[refusedIdx]
		if refused.job != nil {
			t.Fatalf("refused submission %d must not return a job (must not surface an unrelated job's data), got job id %d", refusedIdx, refused.job.ID)
		}
		if refused.reused {
			t.Fatalf("refused submission %d must not be marked reused", refusedIdx)
		}

		var jobCount int64
		if err := db.Model(&models.DeepIdentificationJob{}).Where("user_id = ?", user.ID).Count(&jobCount).Error; err != nil {
			t.Fatal(err)
		}
		if jobCount != 1 {
			t.Fatalf("expected MaxActivePerUser=1 to bound two concurrent distinct-image submissions to exactly 1 job row, got %d", jobCount)
		}
	})

	t.Run("byte-identical concurrent submissions still dedupe to one job under a real race", func(t *testing.T) {
		svc, db, _ := newDeepIdentificationServiceTestDeps(t)
		user := models.User{Username: "concurrent-dup-owner", Email: "concurrent-dup-owner@example.com", PasswordHash: "x"}
		if err := db.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
		enableDeepIdentification(t, svc, map[string]string{
			SettingDeepIdentificationMaxActivePerUser: "5",
			SettingDeepIdentificationQueueDepth:       "10",
		})

		img := coloredPNGBytes(t, 55, 66, 77)
		in := CreateJobInput{
			UserID: user.ID,
			ObverseBytes: img, ObverseFilename: "o.png",
			ReverseBytes: append([]byte(nil), img...), ReverseFilename: "r.png",
		}

		results := make([]struct {
			job    *models.DeepIdentificationJob
			reused bool
			err    error
		}, 2)
		var wg sync.WaitGroup
		start := make(chan struct{})
		wg.Add(2)
		for i := 0; i < 2; i++ {
			i := i
			go func() {
				defer wg.Done()
				<-start
				job, reused, err := svc.CreateJobFromIntake(in)
				results[i] = struct {
					job    *models.DeepIdentificationJob
					reused bool
					err    error
				}{job, reused, err}
			}()
		}
		close(start)
		wg.Wait()

		for i, r := range results {
			if r.err != nil {
				t.Fatalf("submission %d failed: %v", i, r.err)
			}
		}
		if results[0].job.ID != results[1].job.ID {
			t.Fatalf("expected byte-identical concurrent submissions to dedupe to one job, got ids %d and %d", results[0].job.ID, results[1].job.ID)
		}
		if results[0].reused == results[1].reused {
			t.Fatalf("expected exactly one of the two identical racing submissions to report reused=true, got %v and %v", results[0].reused, results[1].reused)
		}

		var jobCount int64
		if err := db.Model(&models.DeepIdentificationJob{}).Where("user_id = ?", user.ID).Count(&jobCount).Error; err != nil {
			t.Fatal(err)
		}
		if jobCount != 1 {
			t.Fatalf("expected exactly one job row for two concurrent byte-identical submissions, got %d", jobCount)
		}
	})
}

// coloredPNGBytes returns a tiny, valid PNG whose pixel bytes are derived
// from (r,g,b) so two different colors reliably hash to two different
// sha256 sums (and therefore two different InputFingerprints) - unlike
// tinyPNGBytes, which always returns the same fixed pixel content.
func coloredPNGBytes(t *testing.T, r, g, b uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: r, G: g, B: b, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode colored png fixture: %v", err)
	}
	return buf.Bytes()
}
