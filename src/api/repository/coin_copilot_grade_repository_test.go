package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

func seedGradableRun(t *testing.T, db *gorm.DB, id string, status models.CopilotRunStatus, answer string, completedAt time.Time) models.CoinCopilotRun {
	t.Helper()
	thread := models.CoinCopilotThread{ID: "cct_" + id, UserID: 7, Title: id}
	run := models.CoinCopilotRun{
		ID: id, ThreadID: thread.ID, UserID: 7, Status: status, Goal: "question " + id, FinalAnswer: answer,
		StartIdempotencyKeyHash: id, StartRequestFingerprint: "fingerprint", ExecutionID: "cce_" + id,
		MaxIterations: 8, MaxToolCalls: 12, MaxConcurrentTools: 1, HardTimeoutSeconds: 120,
		MaxPersistedToolResultBytes: 32768, CompletedAt: &completedAt,
	}
	if err := db.Create(&thread).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	return run
}

func gradeFor(run models.CoinCopilotRun, quality float64, gradedAt time.Time) *models.CoinCopilotRunGrade {
	return &models.CoinCopilotRunGrade{
		RunID: run.ID, ThreadID: run.ThreadID, UserID: run.UserID, Quality: quality,
		AnswersDirectly: quality >= 0.5, Grounded: true, Accurate: true, CitesSources: "not_applicable",
		NoOfferMenu: quality == 1, NoFalseInability: true, Explanation: "explanation", JudgeModel: "claude-sonnet-5-5",
		GradedAt: gradedAt,
	}
}

func TestCopilotGradeRepositoryListsOnlyCompletedUngradedRunsWithCheckpoints(t *testing.T) {
	db, _ := newCopilotRepositoryTestDB(t)
	repo := NewCoinCopilotGradeRepository(db)
	now := time.Now().UTC()
	older := seedGradableRun(t, db, "ccr_older", models.CopilotRunCompleted, "answer", now.Add(-2*time.Hour))
	newer := seedGradableRun(t, db, "ccr_newer", models.CopilotRunCompleted, "answer", now.Add(-time.Hour))
	graded := seedGradableRun(t, db, "ccr_graded", models.CopilotRunCompleted, "answer", now)
	seedGradableRun(t, db, "ccr_failed", models.CopilotRunFailed, "", now)
	pruned := seedGradableRun(t, db, "ccr_pruned", models.CopilotRunCompleted, "answer", now)
	if err := db.Model(&pruned).Update("checkpoints_pruned_at", now).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(gradeFor(graded, 1, now)); err != nil {
		t.Fatal(err)
	}

	runs, err := repo.ListUngradedRuns(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 || runs[0].ID != newer.ID || runs[1].ID != older.ID {
		t.Fatalf("ungraded runs = %#v", runs)
	}
	if capped, _ := repo.ListUngradedRuns(1); len(capped) != 1 {
		t.Fatalf("cap ignored: %d runs", len(capped))
	}
}

func TestCopilotGradeRepositorySaveReplacesTheRunsGrade(t *testing.T) {
	db, _ := newCopilotRepositoryTestDB(t)
	repo := NewCoinCopilotGradeRepository(db)
	run := seedGradableRun(t, db, "ccr_once", models.CopilotRunCompleted, "answer", time.Now())
	if err := repo.Save(gradeFor(run, 0.2, time.Now())); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(gradeFor(run, 0.9, time.Now())); err != nil {
		t.Fatal(err)
	}
	var grades []models.CoinCopilotRunGrade
	if err := db.Find(&grades).Error; err != nil {
		t.Fatal(err)
	}
	if len(grades) != 1 || grades[0].Quality != 0.9 {
		t.Fatalf("grades = %#v", grades)
	}
}

func TestCopilotGradeRepositoryListsWorstFirstWithQuestionAndAnswer(t *testing.T) {
	db, _ := newCopilotRepositoryTestDB(t)
	repo := NewCoinCopilotGradeRepository(db)
	now := time.Now().UTC()
	for _, item := range []struct {
		id      string
		quality float64
	}{{"ccr_good", 1}, {"ccr_bad", 0.2}, {"ccr_mid", 0.6}} {
		run := seedGradableRun(t, db, item.id, models.CopilotRunCompleted, "answer "+item.id, now)
		if err := repo.Save(gradeFor(run, item.quality, now)); err != nil {
			t.Fatal(err)
		}
	}

	rows, total, err := repo.List(CopilotGradeListFilter{Page: 1, Limit: 2, Sort: "worst"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(rows) != 2 || rows[0].RunID != "ccr_bad" || rows[1].RunID != "ccr_mid" {
		t.Fatalf("total=%d rows=%#v", total, rows)
	}
	if rows[0].Question != "question ccr_bad" || rows[0].Answer != "answer ccr_bad" {
		t.Fatalf("joined row = %#v", rows[0])
	}

	if err := repo.SetDispute(rows[0].ID, true, "judge missed the citation", now); err != nil {
		t.Fatal(err)
	}
	disputed := true
	rows, total, err = repo.List(CopilotGradeListFilter{Page: 1, Limit: 10, Disputed: &disputed})
	if err != nil || total != 1 || rows[0].DisputeNote != "judge missed the citation" || rows[0].DisputedAt == nil {
		t.Fatalf("disputed rows=%#v total=%d err=%v", rows, total, err)
	}
	if err := repo.SetDispute(rows[0].ID, false, "ignored", now); err != nil {
		t.Fatal(err)
	}
	cleared, err := repo.Get(rows[0].ID)
	if err != nil || cleared.Disputed || cleared.DisputeNote != "" || cleared.DisputedAt != nil {
		t.Fatalf("cleared = %#v err=%v", cleared, err)
	}
	if err := repo.SetDispute(999, true, "", now); !errors.Is(err, ErrCopilotGradeNotFound) {
		t.Fatalf("missing grade error = %v", err)
	}
	if _, err := repo.Get(999); !errors.Is(err, ErrCopilotGradeNotFound) {
		t.Fatalf("missing grade get error = %v", err)
	}
}

func TestCopilotGradeRepositorySummaryExcludesDisputedAndOldGrades(t *testing.T) {
	db, _ := newCopilotRepositoryTestDB(t)
	repo := NewCoinCopilotGradeRepository(db)
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	day1, day2 := now.AddDate(0, 0, -2), now.AddDate(0, 0, -1)
	cited := seedGradableRun(t, db, "ccr_cited", models.CopilotRunCompleted, "a", day1)
	grade := gradeFor(cited, 1, day1)
	grade.CitesSources = "yes"
	if err := repo.Save(grade); err != nil {
		t.Fatal(err)
	}
	uncited := seedGradableRun(t, db, "ccr_uncited", models.CopilotRunCompleted, "a", day2)
	grade = gradeFor(uncited, 0.4, day2)
	grade.CitesSources = "no"
	if err := repo.Save(grade); err != nil {
		t.Fatal(err)
	}
	disputedRun := seedGradableRun(t, db, "ccr_disputed", models.CopilotRunCompleted, "a", day2)
	disputed := gradeFor(disputedRun, 0, day2)
	disputed.Disputed = true
	if err := repo.Save(disputed); err != nil {
		t.Fatal(err)
	}
	oldRun := seedGradableRun(t, db, "ccr_old", models.CopilotRunCompleted, "a", now.AddDate(0, 0, -60))
	if err := repo.Save(gradeFor(oldRun, 0, now.AddDate(0, 0, -60))); err != nil {
		t.Fatal(err)
	}

	summary, err := repo.Summary(30, now)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Graded != 2 || summary.Disputed != 1 || summary.AvgQuality != 0.7 {
		t.Fatalf("summary = %#v", summary)
	}
	if len(summary.Trend) != 2 || summary.Trend[0].Day != "2026-10-19" || summary.Trend[1].AvgQuality != 0.4 {
		t.Fatalf("trend = %#v", summary.Trend)
	}
	if summary.FailRates["answersDirectly"] != 0.5 || summary.FailRates["noOfferMenu"] != 0.5 || summary.FailRates["citesSources"] != 0.5 {
		t.Fatalf("fail rates = %#v", summary.FailRates)
	}
}

func TestDeleteThreadRemovesItsGrades(t *testing.T) {
	db, copilotRepo := newCopilotRepositoryTestDB(t)
	repo := NewCoinCopilotGradeRepository(db)
	run := seedGradableRun(t, db, "ccr_delete", models.CopilotRunCompleted, "answer", time.Now())
	if err := repo.Save(gradeFor(run, 1, time.Now())); err != nil {
		t.Fatal(err)
	}
	if err := copilotRepo.DeleteThread(run.ThreadID, run.UserID); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&models.CoinCopilotRunGrade{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("grades left after delete: %d err=%v", count, err)
	}
}
