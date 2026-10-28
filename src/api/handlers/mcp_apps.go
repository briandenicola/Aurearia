package handlers

import (
	"context"
	_ "embed"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCP Apps (io.modelcontextprotocol/ui, spec 2026-01-26) lets UI-capable
// hosts render an interactive view for a tool result. Hosts without the
// extension ignore tool _meta and keep using the JSON text content, so the
// metadata is always advertised; the stateless transport cannot see the
// client's initialize capabilities anyway.
const (
	mcpAppMIMEType      = "text/html;profile=mcp-app"
	mcpCoinViewerURI    = "ui://aurearia/coin-viewer.html"
	mcpAppVisibilityApp = "app"
)

//go:embed mcpapps/coin-viewer.html
var mcpCoinViewerHTML string

type mcpCoinImagesInput struct {
	CoinIDs []uint `json:"coinIds" jsonschema:"Owned coin identifiers: 1 to 20 for thumb, exactly 1 for medium"`
	Size    string `json:"size,omitempty" jsonschema:"Optional image variant: thumb (default) or medium"`
}

type mcpCoinImageRef struct {
	CoinID    uint             `json:"coinId"`
	ImageID   uint             `json:"imageId"`
	ImageType models.ImageType `json:"imageType"`
	IsPrimary bool             `json:"isPrimary"`
	MIMEType  string           `json:"mimeType"`
}

// mcpCoinImagesOutput describes the image content blocks: Images[i] is the
// metadata for content[i]. Image bytes travel only in the content blocks.
type mcpCoinImagesOutput struct {
	Images    []mcpCoinImageRef `json:"images"`
	Truncated bool              `json:"truncated"`
}

// mcpCoinViewerToolMeta links a model-visible tool to the coin viewer. The
// flat "ui/resourceUri" key is the deprecated form older hosts still read.
func mcpCoinViewerToolMeta() mcp.Meta {
	return mcp.Meta{
		"ui":             map[string]any{"resourceUri": mcpCoinViewerURI},
		"ui/resourceUri": mcpCoinViewerURI,
	}
}

func (h *MCPHandler) addCoinViewerApp(server *mcp.Server, userID uint) {
	server.AddResource(&mcp.Resource{
		URI:         mcpCoinViewerURI,
		Name:        "coin-viewer",
		Title:       "Aurearia coin viewer",
		Description: "Interactive MCP App view for collection search results and coin details.",
		MIMEType:    mcpAppMIMEType,
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
			URI:      mcpCoinViewerURI,
			MIMEType: mcpAppMIMEType,
			Text:     mcpCoinViewerHTML,
			Meta:     mcp.Meta{"ui": map[string]any{"prefersBorder": true}},
		}}}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_coin_images",
		Description: "Return inline photo thumbnails or medium images for coins owned by the authenticated user. Used by the coin viewer app. Read-only.",
		Meta:        mcp.Meta{"ui": map[string]any{"visibility": []string{mcpAppVisibilityApp}}},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input mcpCoinImagesInput) (*mcp.CallToolResult, mcpCoinImagesOutput, error) {
		images, truncated, err := h.service.GetCoinImages(userID, input.CoinIDs, input.Size)
		if err != nil {
			return nil, mcpCoinImagesOutput{}, h.safeToolError("get_coin_images", err)
		}
		content := make([]mcp.Content, 0, len(images))
		output := mcpCoinImagesOutput{Images: make([]mcpCoinImageRef, 0, len(images)), Truncated: truncated}
		for _, image := range images {
			content = append(content, &mcp.ImageContent{Data: image.Data, MIMEType: image.MIMEType})
			output.Images = append(output.Images, mcpCoinImageRef{
				CoinID: image.CoinID, ImageID: image.ImageID, ImageType: image.ImageType,
				IsPrimary: image.IsPrimary, MIMEType: image.MIMEType,
			})
		}
		// A non-nil Content stops the SDK from duplicating the output as text.
		return &mcp.CallToolResult{Content: content}, output, nil
	})
}
