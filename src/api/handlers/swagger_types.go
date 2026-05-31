package handlers

import "github.com/briandenicola/ancient-coins-api/models"

// Swagger response types for documentation

type ErrorResponse struct {
	Error string `json:"error" example:"Something went wrong"`
}

type MessageResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

type CoinReferenceDTO struct {
	ID        uint   `json:"id" example:"42"`
	CoinID    uint   `json:"coinId" example:"1"`
	Catalog   string `json:"catalog" example:"RIC"`
	Volume    string `json:"volume,omitempty" example:"III"`
	Number    string `json:"number" example:"234"`
	Certainty string `json:"certainty,omitempty" example:"probable"`
	URI       string `json:"uri,omitempty" example:"https://numismatics.org/ocre/id/ric.2.tr.234"`
	CreatedAt string `json:"createdAt" example:"2025-01-15T12:00:00Z"`
	UpdatedAt string `json:"updatedAt" example:"2025-01-15T12:00:00Z"`
}

type CoinReferenceUpsertRequest struct {
	Catalog   string `json:"catalog" example:"RIC"`
	Volume    string `json:"volume,omitempty" example:"III"`
	Number    string `json:"number" example:"234"`
	Certainty string `json:"certainty,omitempty" example:"probable"`
	URI       string `json:"uri,omitempty" example:"https://numismatics.org/ocre/id/ric.2.tr.234"`
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

type StatsResponse struct {
	TotalCoins    int64           `json:"totalCoins" example:"25"`
	TotalWishlist int64           `json:"totalWishlist" example:"5"`
	ByCategory    []CategoryCount `json:"byCategory"`
	ByMaterial    []MaterialCount `json:"byMaterial"`
	Values        ValueSummary    `json:"values"`
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
