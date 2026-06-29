package handlers

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// Swagger response types for documentation

type ErrorResponse struct {
	Error string `json:"error" example:"Something went wrong"`
}

type MessageResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

type CoinReferenceDTO struct {
	ID            uint   `json:"id" example:"42"`
	CoinID        uint   `json:"coinId" example:"1"`
	Catalog       string `json:"catalog" example:"RIC"`
	Volume        string `json:"volume,omitempty" example:"III"`
	Number        string `json:"number" example:"234"`
	InvoiceNumber string `json:"invoiceNumber,omitempty" example:"INV-2024-001"`
	URI           string `json:"uri,omitempty" example:"https://numismatics.org/ocre/id/ric.2.tr.234"`
	CreatedAt     string `json:"createdAt" example:"2025-01-15T12:00:00Z"`
	UpdatedAt     string `json:"updatedAt" example:"2025-01-15T12:00:00Z"`
}

type CoinReferenceUpsertRequest struct {
	Catalog       string `json:"catalog" example:"RIC"`
	Volume        string `json:"volume,omitempty" example:"III"`
	Number        string `json:"number" example:"234"`
	InvoiceNumber string `json:"invoiceNumber,omitempty" example:"INV-2024-001"`
	URI           string `json:"uri,omitempty" example:"https://numismatics.org/ocre/id/ric.2.tr.234"`
}

type MigrationResultDTO struct {
	Succeeded int `json:"succeeded" example:"12"`
	Skipped   int `json:"skipped" example:"45"`
	Failed    int `json:"failed" example:"3"`
}

type AuthResponse struct {
	Token        string       `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	RefreshToken string       `json:"refreshToken" example:"rt_a1b2c3d4..."`
	User         AuthUserInfo `json:"user"`
}

type AuthUserInfo struct {
	ID       uint   `json:"id" example:"1"`
	Username string `json:"username" example:"admin"`
	Role     string `json:"role" example:"admin"`
}

type SetupResponse struct {
	NeedsSetup bool `json:"needsSetup" example:"false"`
}

type CoinListResponse struct {
	Coins []models.Coin `json:"coins"`
	Total int64         `json:"total" example:"42"`
	Page  int           `json:"page" example:"1"`
	Limit int           `json:"limit" example:"50"`
}

type NoteListResponse struct {
	Notes []models.Note `json:"notes"`
}

type WishlistSearchAlertCriteriaRequest struct {
	RulerOrIssuer    string   `json:"rulerOrIssuer" example:"Domitian"`
	CoinType         string   `json:"coinType" example:"Denarius"`
	DateFrom         *int     `json:"dateFrom" example:"81"`
	DateTo           *int     `json:"dateTo" example:"96"`
	Mint             string   `json:"mint" example:"Rome"`
	Material         string   `json:"material" example:"Silver"`
	GradeOrCondition string   `json:"gradeOrCondition" example:"VF or better"`
	PriceMin         *float64 `json:"priceMin" example:"0"`
	PriceMax         *float64 `json:"priceMax" example:"300"`
	Currency         string   `json:"currency" example:"USD"`
	DealerPreference string   `json:"dealerPreference" example:"VCoins or MA-Shops"`
	SourceFilters    []string `json:"sourceFilters" example:"vcoins.com,ma-shops.com"`
	Keywords         string   `json:"keywords" example:"RIC Minerva"`
	Notes            string   `json:"notes" example:"Prefer clear legends"`
}

type WishlistSearchAlertRequest struct {
	Name     string                             `json:"name" example:"Domitian denarius under $300"`
	Criteria WishlistSearchAlertCriteriaRequest `json:"criteria"`
	Cadence  string                             `json:"cadence" example:"manual"`
	IsActive bool                               `json:"isActive" example:"true"`
}

type WishlistSearchAlertListResponse struct {
	Alerts []models.WishlistSearchAlert `json:"alerts"`
	Total  int64                        `json:"total" example:"1"`
	Page   int                          `json:"page" example:"1"`
	Limit  int                          `json:"limit" example:"20"`
}

type WishlistSearchAlertRunRequest struct {
	MaxCandidates int `json:"maxCandidates" example:"20"`
}

type WishlistSearchAlertRunResponse struct {
	RunID           uint                    `json:"runId" example:"42"`
	AlertID         uint                    `json:"alertId" example:"1"`
	Status          models.AlertRunStatus   `json:"status" example:"completed"`
	StartedAt       string                  `json:"startedAt" example:"2026-06-29T17:00:00Z"`
	CompletedAt     string                  `json:"completedAt" example:"2026-06-29T17:00:10Z"`
	ResultCount     int                     `json:"resultCount" example:"8"`
	NewCount        int                     `json:"newCount" example:"5"`
	DuplicateCount  int                     `json:"duplicateCount" example:"3"`
	DismissedCount  int                     `json:"dismissedCount" example:"0"`
	PartialWarnings []string                `json:"partialWarnings"`
	RateLimitStatus string                  `json:"rateLimitStatus" example:"ok"`
	Candidates      []models.AlertCandidate `json:"candidates,omitempty"`
}

type WishlistSearchAlertRunListResponse struct {
	Runs  []models.AlertRun `json:"runs"`
	Total int64             `json:"total" example:"1"`
	Page  int               `json:"page" example:"1"`
	Limit int               `json:"limit" example:"20"`
}

type WishlistSearchAlertCandidateListResponse struct {
	Candidates []models.AlertCandidate `json:"candidates"`
	Total      int64                   `json:"total" example:"1"`
	Page       int                     `json:"page" example:"1"`
	Limit      int                     `json:"limit" example:"20"`
}

type WishlistSearchAlertDismissRequest struct {
	Reason string `json:"reason" example:"price_too_high"`
	Notes  string `json:"notes" example:"Too expensive with shipping"`
}

type WishlistSearchAlertConvertRequest struct {
	Coin                        models.Coin `json:"coin"`
	AcknowledgeDuplicateWarning bool        `json:"acknowledgeDuplicateWarning" example:"false"`
}

type WishlistSearchAlertConvertResponse struct {
	Coin      models.Coin           `json:"coin"`
	Candidate models.AlertCandidate `json:"candidate"`
	Warnings  []string              `json:"warnings"`
}

type WishlistSearchAlertCriteriaAdjustRequest struct {
	CandidateIDs []uint                             `json:"candidateIds"`
	Criteria     WishlistSearchAlertCriteriaRequest `json:"criteria"`
}

type StatsResponse struct {
	TotalCoins    int64           `json:"totalCoins" example:"25"`
	TotalWishlist int64           `json:"totalWishlist" example:"5"`
	ByCategory    []CategoryCount `json:"byCategory"`
	ByMaterial    []MaterialCount `json:"byMaterial"`
	Values        ValueSummary    `json:"values"`
}

type InvestmentBreakdownResponse struct {
	Dimension string                                  `json:"dimension" example:"purchase-month"`
	Segments  []repository.InvestmentBreakdownSegment `json:"segments"`
}

type CategoryCount struct {
	Category string `json:"category" example:"Roman"`
	Count    int64  `json:"count" example:"10"`
}

type MaterialCount struct {
	Material string `json:"material" example:"Silver"`
	Count    int64  `json:"count" example:"8"`
}

type ValueSummary struct {
	TotalPurchasePrice float64 `json:"totalPurchasePrice" example:"1250.00"`
	TotalCurrentValue  float64 `json:"totalCurrentValue" example:"2500.00"`
	AvgPurchasePrice   float64 `json:"avgPurchasePrice" example:"50.00"`
	AvgCurrentValue    float64 `json:"avgCurrentValue" example:"100.00"`
}

type AnalysisResponse struct {
	Analysis string      `json:"analysis" example:"This coin appears to be a Roman denarius..."`
	Side     string      `json:"side" example:"obverse"`
	Coin     models.Coin `json:"coin"`
}

type DeleteAnalysisResponse struct {
	Coin models.Coin `json:"coin"`
}

type ExtractTextResponse struct {
	Text string `json:"text" example:"CAESAR AVGVSTVS"`
}

type OllamaStatusResponse struct {
	Available bool   `json:"available" example:"true"`
	Model     string `json:"model" example:"llava"`
	URL       string `json:"url" example:"http://localhost:11434"`
	Message   string `json:"message" example:"Model llava is available"`
}

type AIStatusResponse struct {
	Available bool   `json:"available" example:"true"`
	Provider  string `json:"provider" example:"anthropic"`
	Model     string `json:"model" example:"claude-sonnet-4-20250514"`
	Message   string `json:"message" example:"Anthropic provider configured (model: claude-sonnet-4-20250514)"`
}

type UserInfoResponse struct {
	ID        uint   `json:"id" example:"1"`
	Username  string `json:"username" example:"admin"`
	Role      string `json:"role" example:"admin"`
	CreatedAt string `json:"createdAt" example:"2025-01-01T00:00:00Z"`
}

type PasswordChangedResponse struct {
	Message string `json:"message" example:"Password changed"`
}

type ImportResponse struct {
	Message  string `json:"message" example:"Import complete"`
	Imported int    `json:"imported" example:"10"`
}

type UserDTO struct {
	ID        uint            `json:"id" example:"1"`
	Username  string          `json:"username" example:"admin"`
	Role      models.UserRole `json:"role" example:"admin"`
	CreatedAt string          `json:"createdAt" example:"2025-01-01T00:00:00Z"`
}

type LogsResponse struct {
	Logs     []LogEntry `json:"logs"`
	Count    int        `json:"count" example:"100"`
	LogLevel string     `json:"logLevel" example:"info"`
}

type LogEntry struct {
	Timestamp string `json:"timestamp" example:"2025-01-01T12:00:00Z"`
	Level     string `json:"level" example:"info"`
	Message   string `json:"message" example:"Application starting"`
}

type ImageDeletedResponse struct {
	Message string `json:"message" example:"Image deleted"`
}

type CoinDeletedResponse struct {
	Message string `json:"message" example:"Coin deleted"`
}

type SettingsUpdateResponse struct {
	Message string `json:"message" example:"Settings updated"`
}

type IntakeConfidenceSummary struct {
	Overall         string   `json:"overall" example:"medium"`
	UncertainFields []string `json:"uncertainFields"`
}

type IntakeEvidenceItem struct {
	Type       string `json:"type" example:"vision"`
	Source     string `json:"source" example:"obverse"`
	Field      string `json:"field" example:"ruler"`
	Value      string `json:"value" example:"Trajan"`
	Confidence string `json:"confidence" example:"medium"`
	Notes      string `json:"notes,omitempty" example:"Legend partially visible"`
}

type IntakeDraftCreateResponse struct {
	DraftID           uint                    `json:"draftId" example:"42"`
	Status            string                  `json:"status" example:"drafted"`
	Coin              map[string]interface{}  `json:"coin"`
	ConfidenceSummary IntakeConfidenceSummary `json:"confidenceSummary"`
	Evidence          []IntakeEvidenceItem    `json:"evidence"`
	UnresolvedFields  []string                `json:"unresolvedFields"`
	ExpiresAt         string                  `json:"expiresAt" example:"2026-01-15T12:00:00Z"`
}

type IntakeDraftCommitRequest struct {
	DraftID   uint                   `json:"draftId" binding:"required" example:"42"`
	Confirm   bool                   `json:"confirm" binding:"required" example:"true"`
	Overrides map[string]interface{} `json:"overrides"`
}

type IntakeDraftCommitResponse struct {
	DraftID uint   `json:"draftId" example:"42"`
	Status  string `json:"status" example:"confirmed"`
	CoinID  uint   `json:"coinId" example:"314"`
}

type SettingInput struct {
	Key   string `json:"key" example:"OllamaURL"`
	Value string `json:"value" example:"http://localhost:11434"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required" example:"oldpass123"`
	NewPassword     string `json:"newPassword" binding:"required,min=6" example:"newpass456"`
}

type ResetPasswordRequest struct {
	NewPassword string `json:"newPassword" binding:"required,min=6" example:"newpass456"`
}

type UpdateUserRoleRequest struct {
	Role models.UserRole `json:"role" binding:"required" example:"admin"`
}

// Coin Lookup types
type NGCDataSwagger struct {
	CertNumber     string `json:"certNumber" example:"823160-093"`
	NormalizedCert string `json:"normalizedCert" example:"823160-093"`
	LookupURL      string `json:"lookupURL" example:"https://www.ngccoin.com/certlookup/823160-093/"`
	Grade          string `json:"grade,omitempty" example:"Ch AU"`
	Description    string `json:"description,omitempty" example:"Roman Empire, Trajan Decius"`
}

type LookupExtractedDataSwagger struct {
	NGC         *NGCDataSwagger `json:"ngc,omitempty"`
	LabelText   string          `json:"labelText,omitempty" example:"NGC Ch AU 5/5 4/5"`
	CoinFields  map[string]any  `json:"coinFields,omitempty"`
	Confidence  string          `json:"confidence" example:"medium"`
	RawAnalysis string          `json:"rawAnalysis" example:"Vision analysis text..."`
}

type NumistaCandidateSwagger struct {
	ID        string `json:"id" example:"12345"`
	Title     string `json:"title" example:"Denarius - Trajan (98-117)"`
	Issuer    string `json:"issuer" example:"Roman Empire"`
	Year      string `json:"year" example:"101-102"`
	Thumbnail string `json:"thumbnail,omitempty" example:"https://en.numista.com/..."`
	URL       string `json:"url" example:"https://en.numista.com/catalogue/pieces12345.html"`
}

type CandidateReferenceSwagger struct {
	Catalog string `json:"catalog" example:"NGC"`
	Volume  string `json:"volume,omitempty" example:""`
	Number  string `json:"number" example:"823160-093"`
	URI     string `json:"uri,omitempty" example:"https://www.ngccoin.com/certlookup/823160-093/"`
}

type CoinLookupSwaggerResponse struct {
	ExtractedData       LookupExtractedDataSwagger  `json:"extractedData"`
	NumistaCandidates   []NumistaCandidateSwagger   `json:"numistaCandidates"`
	PrefilledDraft      map[string]any              `json:"prefilledDraft,omitempty"`
	CandidateReferences []CandidateReferenceSwagger `json:"candidateReferences,omitempty"`
}
