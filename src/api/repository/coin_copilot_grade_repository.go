package repository

import (
	"errors"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrCopilotGradeNotFound reports a grade ID that does not exist.
var ErrCopilotGradeNotFound = errors.New("copilot grade not found")

// CoinCopilotGradeRepository stores rubric grades for completed Copilot runs.
type CoinCopilotGradeRepository struct {
	db *gorm.DB
}

func NewCoinCopilotGradeRepository(db *gorm.DB) *CoinCopilotGradeRepository {
	return &CoinCopilotGradeRepository{db: db}
}

// CopilotGradeRow is a grade joined with the question and answer it judged.
type CopilotGradeRow struct {
	models.CoinCopilotRunGrade
	Question    string     `json:"question"`
	Answer      string     `json:"answer"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// CopilotGradeListFilter selects and orders grades for admin review.
type CopilotGradeListFilter struct {
	Page  int
	Limit int
	// Sort is "worst" (lowest quality first) or "recent" (newest first).
	Sort string
	// Disputed limits the list to disputed (true) or undisputed (false) grades.
	Disputed *bool
}

// CopilotGradeDay is one day of the quality trend.
type CopilotGradeDay struct {
	Day        string  `json:"day"`
	AvgQuality float64 `json:"avgQuality"`
	Graded     int64   `json:"graded"`
}

// CopilotGradeSummary aggregates undisputed grades over a window.
type CopilotGradeSummary struct {
	Days       int               `json:"days"`
	Graded     int64             `json:"graded"`
	Disputed   int64             `json:"disputed"`
	AvgQuality float64           `json:"avgQuality"`
	Trend      []CopilotGradeDay `json:"trend"`
	// FailRates is the share of undisputed grades failing each criterion;
	// citesSources counts only grades where sources applied.
	FailRates map[string]float64 `json:"failRates"`
}

// ListUngradedRuns returns completed runs that still have a checkpoint and no
// grade, newest first.
func (r *CoinCopilotGradeRepository) ListUngradedRuns(limit int) ([]models.CoinCopilotRun, error) {
	var runs []models.CoinCopilotRun
	err := r.db.
		Where("status = ? AND final_answer <> '' AND checkpoints_pruned_at IS NULL", models.CopilotRunCompleted).
		Where("id NOT IN (?)", r.db.Model(&models.CoinCopilotRunGrade{}).Select("run_id")).
		Order("completed_at DESC").
		Limit(limit).
		Find(&runs).Error
	return runs, err
}

// Save stores a grade, replacing any earlier grade for the same run.
func (r *CoinCopilotGradeRepository) Save(grade *models.CoinCopilotRunGrade) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "run_id"}},
		UpdateAll: true,
	}).Create(grade).Error
}

func (r *CoinCopilotGradeRepository) joined() *gorm.DB {
	return r.db.Table("coin_copilot_run_grades AS g").
		Select("g.*, r.goal AS question, r.final_answer AS answer, r.completed_at AS completed_at").
		Joins("JOIN coin_copilot_runs r ON r.id = g.run_id")
}

// List returns one page of grades with their question and answer.
func (r *CoinCopilotGradeRepository) List(filter CopilotGradeListFilter) ([]CopilotGradeRow, int64, error) {
	page := max(filter.Page, 1)
	limit := min(max(filter.Limit, 1), 100)
	query := r.joined()
	if filter.Disputed != nil {
		query = query.Where("g.disputed = ?", *filter.Disputed)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "g.quality ASC, g.graded_at DESC"
	if filter.Sort == "recent" {
		order = "g.graded_at DESC"
	}
	var rows []CopilotGradeRow
	err := query.Order(order).Offset((page - 1) * limit).Limit(limit).Scan(&rows).Error
	return rows, total, err
}

// Get returns one grade with its question and answer.
func (r *CoinCopilotGradeRepository) Get(id uint) (*CopilotGradeRow, error) {
	var row CopilotGradeRow
	result := r.joined().Where("g.id = ?", id).Limit(1).Scan(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrCopilotGradeNotFound
	}
	return &row, nil
}

// SetDispute marks or clears an admin's disagreement with a grade.
func (r *CoinCopilotGradeRepository) SetDispute(id uint, disputed bool, note string, now time.Time) error {
	updates := map[string]interface{}{"disputed": disputed, "dispute_note": note, "disputed_at": nil}
	if disputed {
		updates["disputed_at"] = now
	} else {
		updates["dispute_note"] = ""
	}
	result := r.db.Model(&models.CoinCopilotRunGrade{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCopilotGradeNotFound
	}
	return nil
}

// Summary aggregates undisputed grades from the last `days` days.
func (r *CoinCopilotGradeRepository) Summary(days int, now time.Time) (*CopilotGradeSummary, error) {
	since := now.AddDate(0, 0, -days)
	summary := &CopilotGradeSummary{Days: days, Trend: []CopilotGradeDay{}, FailRates: map[string]float64{}}
	window := r.db.Model(&models.CoinCopilotRunGrade{}).Where("graded_at >= ?", since)
	if err := window.Session(&gorm.Session{}).Where("disputed = ?", true).Count(&summary.Disputed).Error; err != nil {
		return nil, err
	}
	var grades []models.CoinCopilotRunGrade
	if err := window.Session(&gorm.Session{}).Where("disputed = ?", false).Order("graded_at ASC").Find(&grades).Error; err != nil {
		return nil, err
	}
	summary.Graded = int64(len(grades))
	if len(grades) == 0 {
		return summary, nil
	}
	failures := map[string]int{}
	sourcesApplicable := 0
	total := 0.0
	byDay := map[string]*CopilotGradeDay{}
	for _, grade := range grades {
		total += grade.Quality
		day := grade.GradedAt.UTC().Format("2006-01-02")
		entry, ok := byDay[day]
		if !ok {
			entry = &CopilotGradeDay{Day: day}
			byDay[day] = entry
			summary.Trend = append(summary.Trend, CopilotGradeDay{Day: day})
		}
		entry.AvgQuality += grade.Quality
		entry.Graded++
		for name, passed := range map[string]bool{
			"answersDirectly":  grade.AnswersDirectly,
			"grounded":         grade.Grounded,
			"accurate":         grade.Accurate,
			"noOfferMenu":      grade.NoOfferMenu,
			"noFalseInability": grade.NoFalseInability,
		} {
			if !passed {
				failures[name]++
			}
		}
		if grade.CitesSources != "not_applicable" {
			sourcesApplicable++
			if grade.CitesSources != "yes" {
				failures["citesSources"]++
			}
		}
	}
	for i, day := range summary.Trend {
		entry := byDay[day.Day]
		summary.Trend[i] = CopilotGradeDay{Day: day.Day, Graded: entry.Graded, AvgQuality: entry.AvgQuality / float64(entry.Graded)}
	}
	summary.AvgQuality = total / float64(len(grades))
	for _, name := range []string{"answersDirectly", "grounded", "accurate", "noOfferMenu", "noFalseInability"} {
		summary.FailRates[name] = float64(failures[name]) / float64(len(grades))
	}
	if sourcesApplicable > 0 {
		summary.FailRates["citesSources"] = float64(failures["citesSources"]) / float64(sourcesApplicable)
	}
	return summary, nil
}
