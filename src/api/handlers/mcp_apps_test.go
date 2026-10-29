package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// writeMCPTestImage stores a coin image record plus its files under the
// fixture upload dir. Variants map a suffix such as "_thumb.jpg" to bytes.
func writeMCPTestImage(t *testing.T, fixture *mcpRouteFixture, coinID uint, name string, imageType models.ImageType, primary bool, original []byte, variants map[string][]byte) models.CoinImage {
	t.Helper()
	coinDir := filepath.Join(fixture.uploadDir, fmt.Sprintf("coin-%d", coinID))
	if err := os.MkdirAll(coinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coinDir, name+".jpg"), original, 0o644); err != nil {
		t.Fatal(err)
	}
	for suffix, data := range variants {
		if err := os.WriteFile(filepath.Join(coinDir, name+suffix), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	image := models.CoinImage{
		CoinID: coinID, FilePath: fmt.Sprintf("coin-%d/%s.jpg", coinID, name),
		ImageType: imageType, IsPrimary: primary,
	}
	if err := fixture.db.Create(&image).Error; err != nil {
		t.Fatal(err)
	}
	return image
}

func mcpToolByName(t *testing.T, tools []*mcp.Tool, name string) *mcp.Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %q not registered", name)
	return nil
}

func mcpUIMeta(t *testing.T, tool *mcp.Tool) map[string]any {
	t.Helper()
	ui, ok := tool.Meta["ui"].(map[string]any)
	if !ok {
		t.Fatalf("%s _meta.ui=%#v, want object", tool.Name, tool.Meta["ui"])
	}
	return ui
}

func decodeMCPCoinImages(t *testing.T, result *mcp.CallToolResult) mcpCoinImagesOutput {
	t.Helper()
	encoded, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var output mcpCoinImagesOutput
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatal(err)
	}
	return output
}

func TestMCPCoinViewerAppIsAdvertisedAndServed(t *testing.T) {
	fixture := setupMCPRouteFixture(t, 100)
	session := connectMCPRoute(t, fixture, "read")

	capabilities := session.InitializeResult().Capabilities
	extension, ok := capabilities.Extensions[mcpAppsExtension].(map[string]any)
	if !ok {
		t.Fatalf("initialize extensions=%#v, want %s declared", capabilities.Extensions, mcpAppsExtension)
	}
	mimeTypes, _ := extension["mimeTypes"].([]any)
	if len(mimeTypes) != 1 || mimeTypes[0] != mcpAppMIMEType {
		t.Fatalf("%s mimeTypes=%#v, want [%s]", mcpAppsExtension, extension["mimeTypes"], mcpAppMIMEType)
	}
	if capabilities.Logging == nil || capabilities.Tools == nil || capabilities.Resources == nil {
		t.Fatalf("capabilities=%+v, want logging, tools and resources kept", capabilities)
	}

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"search_collection", "get_coin"} {
		tool := mcpToolByName(t, tools.Tools, name)
		if got := mcpUIMeta(t, tool)["resourceUri"]; got != mcpCoinViewerURI {
			t.Fatalf("%s resourceUri=%v, want %s", name, got, mcpCoinViewerURI)
		}
		if got := tool.Meta["ui/resourceUri"]; got != mcpCoinViewerURI {
			t.Fatalf("%s legacy ui/resourceUri=%v, want %s", name, got, mcpCoinViewerURI)
		}
	}
	imagesUI := mcpUIMeta(t, mcpToolByName(t, tools.Tools, "get_coin_images"))
	visibility, _ := imagesUI["visibility"].([]any)
	if len(visibility) != 1 || visibility[0] != mcpAppVisibilityApp {
		t.Fatalf("get_coin_images visibility=%#v, want [app]", imagesUI["visibility"])
	}
	if _, linked := imagesUI["resourceUri"]; linked {
		t.Fatal("app-only image tool must not open its own view")
	}
	for _, tool := range tools.Tools {
		if tool.Name != "search_collection" && tool.Name != "get_coin" && tool.Meta["ui/resourceUri"] != nil {
			t.Fatalf("unexpected viewer link on %s", tool.Name)
		}
	}

	resources, err := session.ListResources(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources.Resources) != 1 || resources.Resources[0].URI != mcpCoinViewerURI ||
		resources.Resources[0].MIMEType != mcpAppMIMEType {
		t.Fatalf("resources=%+v, want only the coin viewer", resources.Resources)
	}
	read, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: mcpCoinViewerURI})
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Contents) != 1 {
		t.Fatalf("contents=%d, want 1", len(read.Contents))
	}
	contents := read.Contents[0]
	if contents.MIMEType != mcpAppMIMEType || !strings.Contains(contents.Text, "ui/initialize") {
		t.Fatalf("viewer contents mime=%q, missing ui/initialize handshake", contents.MIMEType)
	}
	for _, external := range []string{"<script src", "<link ", "@import", "fetch("} {
		if strings.Contains(contents.Text, external) {
			t.Fatalf("viewer must stay self-contained; found %q", external)
		}
	}
}

func TestMCPGetCoinImagesIsOwnerScopedBoundedAndExcludesCards(t *testing.T) {
	fixture := setupMCPRouteFixture(t, 100)
	thumb := []byte("owner-obverse-thumb")
	medium := []byte("owner-obverse-medium")
	obverse := writeMCPTestImage(t, fixture, fixture.ownerCoinID, "obverse", models.ImageTypeObverse, true,
		[]byte("original"), map[string][]byte{"_thumb.jpg": thumb, "_medium.jpg": medium})
	reverse := writeMCPTestImage(t, fixture, fixture.ownerCoinID, "reverse", models.ImageTypeReverse, false,
		[]byte("original"), map[string][]byte{"_medium.jpg": []byte("owner-reverse-medium")})
	writeMCPTestImage(t, fixture, fixture.ownerCoinID, "card", models.ImageTypeCard, false,
		[]byte("card"), map[string][]byte{"_thumb.jpg": []byte("card-thumb"), "_medium.jpg": []byte("card-medium")})
	writeMCPTestImage(t, fixture, fixture.foreignCoinID, "foreign", models.ImageTypeObverse, true,
		[]byte("foreign"), map[string][]byte{"_thumb.jpg": []byte("foreign-thumb")})
	session := connectMCPRoute(t, fixture, "read")

	thumbs := callMCPTool(t, session, "get_coin_images", map[string]any{
		"coinIds": []uint{fixture.ownerCoinID, fixture.foreignCoinID},
	})
	if thumbs.IsError {
		t.Fatalf("thumb call returned tool error: %+v", thumbs.Content)
	}
	output := decodeMCPCoinImages(t, thumbs)
	if len(thumbs.Content) != 1 || len(output.Images) != 1 {
		t.Fatalf("thumb content=%d images=%+v, want one owner primary thumb", len(thumbs.Content), output.Images)
	}
	image, ok := thumbs.Content[0].(*mcp.ImageContent)
	if !ok || !bytes.Equal(image.Data, thumb) || image.MIMEType != "image/jpeg" {
		t.Fatalf("thumb content=%#v, want owner obverse thumb bytes", thumbs.Content[0])
	}
	if output.Images[0].ImageID != obverse.ID || output.Images[0].CoinID != fixture.ownerCoinID || output.Truncated {
		t.Fatalf("thumb metadata=%+v, want owner obverse", output)
	}

	mediums := callMCPTool(t, session, "get_coin_images", map[string]any{
		"coinIds": []uint{fixture.ownerCoinID}, "size": "medium",
	})
	output = decodeMCPCoinImages(t, mediums)
	if mediums.IsError || len(mediums.Content) != 2 || len(output.Images) != 2 ||
		output.Images[0].ImageID != obverse.ID || output.Images[1].ImageID != reverse.ID {
		t.Fatalf("medium result=%+v images=%+v, want obverse then reverse without card", mediums.Content, output.Images)
	}

	for _, content := range append(thumbs.Content, mediums.Content...) {
		data := content.(*mcp.ImageContent).Data
		for _, leaked := range []string{"card-thumb", "card-medium", "foreign-thumb", "original"} {
			if bytes.Equal(data, []byte(leaked)) {
				t.Fatalf("image results leaked %q bytes", leaked)
			}
		}
	}

	for _, arguments := range []map[string]any{
		{"coinIds": []uint{}},
		{"coinIds": []uint{fixture.ownerCoinID, fixture.foreignCoinID}, "size": "medium"},
		{"coinIds": []uint{fixture.ownerCoinID}, "size": "full"},
		{"coinIds": []uint{0}},
		{"coinIds": make([]uint, 21)},
	} {
		result := callMCPTool(t, session, "get_coin_images", arguments)
		raw, _ := json.Marshal(result)
		if !result.IsError || !bytes.Contains(raw, []byte("invalid tool input")) {
			t.Fatalf("arguments %v result=%s, want invalid tool input", arguments, raw)
		}
	}
}

func TestMCPGetCoinImagesFallsBackOnlyToSmallSupportedOriginals(t *testing.T) {
	fixture := setupMCPRouteFixture(t, 100)
	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 64)...)
	legacy := writeMCPTestImage(t, fixture, fixture.ownerCoinID, "legacy", models.ImageTypeObverse, true, png, nil)
	writeMCPTestImage(t, fixture, fixture.ownerCoinID, "huge", models.ImageTypeReverse, false,
		append([]byte("\xff\xd8\xff"), bytes.Repeat([]byte{0}, 600*1024)...), nil)
	writeMCPTestImage(t, fixture, fixture.ownerCoinID, "notimage", models.ImageTypeDetail, false, []byte("<svg/>"), nil)
	session := connectMCPRoute(t, fixture, "read")

	result := callMCPTool(t, session, "get_coin_images", map[string]any{
		"coinIds": []uint{fixture.ownerCoinID}, "size": "medium",
	})
	output := decodeMCPCoinImages(t, result)
	if result.IsError || len(output.Images) != 1 || output.Images[0].ImageID != legacy.ID || output.Images[0].MIMEType != "image/png" {
		t.Fatalf("fallback images=%+v, want only the small legacy PNG", output.Images)
	}
}
