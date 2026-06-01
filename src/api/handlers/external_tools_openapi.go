package handlers

import (
	_ "embed"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

//go:embed contracts/external-tool-server.openapi.yaml
var externalToolServerSpec string

// ExternalToolsOpenAPIHandler serves the scoped OpenAPI document for external clients.
type ExternalToolsOpenAPIHandler struct{}

func NewExternalToolsOpenAPIHandler() *ExternalToolsOpenAPIHandler {
	return &ExternalToolsOpenAPIHandler{}
}

// GetOpenAPISpec godoc
// @Summary      Fetch the external tool server OpenAPI document
// @Description  Returns the OpenAPI 3.0 document describing the /v1/tools/* surface, suitable for client auto-import.
// @Tags         External Tools
// @Produce      json
// @Success      200 {object} map[string]interface{} "OpenAPI 3.0 document"
// @Failure      500 {object} map[string]interface{}
// @Router       /v1/tools/openapi.json [get]
func (h *ExternalToolsOpenAPIHandler) GetOpenAPISpec(c *gin.Context) {
	// Parse YAML into a map
	var spec map[string]interface{}
	if err := yaml.Unmarshal([]byte(externalToolServerSpec), &spec); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse OpenAPI spec"})
		return
	}

	// Serve as JSON
	c.JSON(http.StatusOK, spec)
}

// GetOpenAPISpecJSON returns the spec as JSON bytes for direct use (e.g., testing).
func (h *ExternalToolsOpenAPIHandler) GetOpenAPISpecJSON() ([]byte, error) {
	var spec map[string]interface{}
	if err := yaml.Unmarshal([]byte(externalToolServerSpec), &spec); err != nil {
		return nil, err
	}
	return json.Marshal(spec)
}
