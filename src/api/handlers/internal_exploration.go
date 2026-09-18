package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type ExplorationDecider interface {
	DecideBrowserExploration(context.Context, services.ExplorationDecisionRequest) (*services.ExplorationDecisionResponse, error)
}

var (
	explorationRunID      = regexp.MustCompile(`^aibr_[0-9]{8}T[0-9]{6}Z_[a-f0-9]{12}$`)
	explorationStepID     = regexp.MustCompile(`^step-[0-9]{3}$`)
	explorationEvidenceID = regexp.MustCompile(`^ev_[a-z]+_[0-9]{4}$`)
)

var explorationActions = map[string]struct{}{
	"navigate": {}, "click": {}, "fill": {}, "select": {}, "upload_fixture": {},
	"set_viewport": {}, "back": {}, "wait_for_ui": {}, "checkpoint": {}, "finish": {},
}

var explorationEvidenceKinds = map[string]struct{}{
	"route": {}, "screenshot": {}, "console": {}, "network": {}, "accessibility": {}, "ui": {},
}

func validExplorationRequest(req services.ExplorationDecisionRequest) bool {
	if req.SchemaVersion != services.ExplorationDecisionSchemaVersion ||
		!explorationRunID.MatchString(req.RunID) || !explorationStepID.MatchString(req.StepID) ||
		(req.Provider != "anthropic" && req.Provider != "ollama") ||
		len(req.Model) < 1 || len(req.Model) > 120 || len(req.Workflow) < 1 || len(req.Workflow) > 80 ||
		len(req.Goal) < 1 || len(req.Goal) > 1000 ||
		len(req.AllowedActions) < 1 || len(req.AllowedRoutes) < 1 || len(req.AllowedRoutes) > 30 ||
		len(req.Observations) > 100 || strings.Contains(strings.ToLower(req.Model), "://") {
		return false
	}
	seenActions := map[string]struct{}{}
	for _, action := range req.AllowedActions {
		if _, ok := explorationActions[action]; !ok {
			return false
		}
		if _, duplicate := seenActions[action]; duplicate {
			return false
		}
		seenActions[action] = struct{}{}
	}
	seenRoutes := map[string]struct{}{}
	for _, route := range req.AllowedRoutes {
		if !strings.HasPrefix(route, "/") || strings.HasPrefix(route, "//") || len(route) > 200 {
			return false
		}
		if _, duplicate := seenRoutes[route]; duplicate {
			return false
		}
		seenRoutes[route] = struct{}{}
	}
	for _, observation := range req.Observations {
		if !explorationEvidenceID.MatchString(observation.EvidenceID) ||
			!strings.HasPrefix(observation.Route, "/") || len(observation.Route) > 200 ||
			len(observation.Summary) > 2000 {
			return false
		}
		if _, ok := explorationEvidenceKinds[observation.Kind]; !ok {
			return false
		}
	}
	return true
}

// RegisterInternalExplorationRoute adds no route at all unless explicitly
// enabled. Authentication uses a separate per-run bearer credential.
func RegisterInternalExplorationRoute(r *gin.Engine, enabled bool, token string, decider ExplorationDecider) {
	if !enabled {
		return
	}
	r.POST("/api/internal/browser-exploration/decide", func(c *gin.Context) {
		provided := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if token == "" || provided == "" ||
			subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		var req services.ExplorationDecisionRequest
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 256<<10)
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil || decoder.Decode(&struct{}{}) != io.EOF || !validExplorationRequest(req) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exploration request"})
			return
		}
		response, err := decider.DecideBrowserExploration(c.Request.Context(), req)
		if err != nil {
			status := http.StatusBadGateway
			switch {
			case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
				status = http.StatusGatewayTimeout
			case errors.Is(err, services.ErrExplorationRateLimited):
				status = http.StatusTooManyRequests
			case errors.Is(err, services.ErrExplorationInvalidOutput):
				status = http.StatusUnprocessableEntity
			}
			c.JSON(status, gin.H{"error": "exploration decision unavailable"})
			return
		}
		c.JSON(http.StatusOK, response)
	})
}
