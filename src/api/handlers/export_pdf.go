package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/go-pdf/fpdf"
)

const (
	pageW      = 210.0 // A4 width mm
	pageH      = 297.0 // A4 height mm
	margin     = 15.0
	contentW   = pageW - 2*margin
	goldR      = 191
	goldG      = 155
	goldB      = 48
	darkR      = 30
	darkG      = 30
	darkB      = 34
	cardBgR    = 42
	cardBgG    = 42
	cardBgB    = 46
	textR      = 220
	textG      = 220
	textB      = 224
	mutedR     = 160
	mutedG     = 160
	mutedB     = 164
)

// writeCatalogPDF generates a styled PDF catalog for the given coins.
func writeCatalogPDF(coins []models.Coin, uploadDir string, username string) (*fpdf.Fpdf, error) {
	logger := services.AppLogger

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, margin)
	pdf.SetMargins(margin, margin, margin)

	// Cover page
	writeCoverPage(pdf, username, len(coins))

	// Coin pages
	for i, coin := range coins {
		if coin.IsWishlist || coin.IsSold {
			continue
		}
		writeCoinPage(pdf, coin, uploadDir, i+1, logger)
	}

	// Summary page
	writeSummaryPage(pdf, coins)

	return pdf, nil
}

func writeCoverPage(pdf *fpdf.Fpdf, username string, count int) {
	pdf.AddPage()
	pdf.SetFillColor(darkR, darkG, darkB)
	pdf.Rect(0, 0, pageW, pageH, "F")

	// Gold decorative line
	pdf.SetDrawColor(goldR, goldG, goldB)
	pdf.SetLineWidth(0.8)
	pdf.Line(margin, 80, pageW-margin, 80)
	pdf.Line(margin, 82, pageW-margin, 82)

	// Title
	pdf.SetTextColor(goldR, goldG, goldB)
	pdf.SetFont("Helvetica", "B", 32)
	pdf.SetY(95)
	pdf.CellFormat(contentW, 14, "Coin Collection", "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 14)
	pdf.SetTextColor(textR, textG, textB)
	pdf.Ln(4)
	pdf.CellFormat(contentW, 8, "Insurance & Provenance Catalog", "", 1, "C", false, 0, "")

	// Owner and date
	pdf.Ln(16)
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetTextColor(mutedR, mutedG, mutedB)
	pdf.CellFormat(contentW, 6, fmt.Sprintf("Owner: %s", username), "", 1, "C", false, 0, "")
	pdf.CellFormat(contentW, 6, fmt.Sprintf("Generated: %s", time.Now().Format("January 2, 2006")), "", 1, "C", false, 0, "")
	pdf.CellFormat(contentW, 6, fmt.Sprintf("Collection: %d coins", count), "", 1, "C", false, 0, "")

	// Bottom decorative line
	pdf.SetDrawColor(goldR, goldG, goldB)
	pdf.Line(margin, 200, pageW-margin, 200)
	pdf.Line(margin, 202, pageW-margin, 202)
}

func writeCoinPage(pdf *fpdf.Fpdf, coin models.Coin, uploadDir string, index int, logger *services.Logger) {
	pdf.AddPage()
	pdf.SetFillColor(darkR, darkG, darkB)
	pdf.Rect(0, 0, pageW, pageH, "F")

	y := margin

	// Coin header
	pdf.SetTextColor(goldR, goldG, goldB)
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(margin, y)
	pdf.CellFormat(contentW, 8, safeStr(coin.Name), "", 1, "L", false, 0, "")
	y += 10

	// Category / era / grade line
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(mutedR, mutedG, mutedB)
	meta := []string{}
	if coin.Category != "" {
		meta = append(meta, string(coin.Category))
	}
	if coin.Era != "" {
		meta = append(meta, coin.Era)
	}
	if coin.Grade != "" {
		meta = append(meta, "Grade: "+coin.Grade)
	}
	if len(meta) > 0 {
		pdf.SetXY(margin, y)
		pdf.CellFormat(contentW, 5, strings.Join(meta, "  |  "), "", 1, "L", false, 0, "")
		y += 7
	}

	// Gold separator
	pdf.SetDrawColor(goldR, goldG, goldB)
	pdf.SetLineWidth(0.3)
	pdf.Line(margin, y, pageW-margin, y)
	y += 4

	// Images (side by side if two exist)
	imageY := y
	images := getDisplayImages(coin.Images)
	imgW := 80.0
	imgH := 80.0

	if len(images) >= 2 {
		imgW = contentW/2 - 2
		for i, img := range images[:2] {
			diskPath := filepath.Join(uploadDir, filepath.FromSlash(img.FilePath))
			if _, err := os.Stat(diskPath); err == nil {
				x := margin + float64(i)*(imgW+4)
				registerImage(pdf, diskPath, x, imageY, imgW, imgH, logger)
			}
		}
		y = imageY + imgH + 4
	} else if len(images) == 1 {
		diskPath := filepath.Join(uploadDir, filepath.FromSlash(images[0].FilePath))
		if _, err := os.Stat(diskPath); err == nil {
			x := margin + (contentW-imgW)/2
			registerImage(pdf, diskPath, x, imageY, imgW, imgH, logger)
		}
		y = imageY + imgH + 4
	}

	// Details card
	y = writeDetailsCard(pdf, coin, y)

	// Provenance / notes section
	if coin.Notes != "" {
		y += 4
		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetTextColor(goldR, goldG, goldB)
		pdf.SetXY(margin, y)
		pdf.CellFormat(contentW, 6, "Notes / Provenance", "", 1, "L", false, 0, "")
		y += 7
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(textR, textG, textB)
		pdf.SetXY(margin, y)
		pdf.MultiCell(contentW, 4.5, safeStr(coin.Notes), "", "L", false)
	}

	// Tags
	if len(coin.Tags) > 0 {
		tagNames := make([]string, len(coin.Tags))
		for i, t := range coin.Tags {
			tagNames[i] = t.Name
		}
		currentY := pdf.GetY() + 4
		pdf.SetFont("Helvetica", "I", 8)
		pdf.SetTextColor(mutedR, mutedG, mutedB)
		pdf.SetXY(margin, currentY)
		pdf.CellFormat(contentW, 5, "Tags: "+strings.Join(tagNames, ", "), "", 1, "L", false, 0, "")
	}
}

func writeDetailsCard(pdf *fpdf.Fpdf, coin models.Coin, startY float64) float64 {
	y := startY
	pdf.SetFillColor(cardBgR, cardBgG, cardBgB)

	rows := []struct{ label, value string }{}

	if coin.Ruler != "" {
		rows = append(rows, struct{ label, value string }{"Ruler", coin.Ruler})
	}
	if coin.Denomination != "" {
		rows = append(rows, struct{ label, value string }{"Denomination", coin.Denomination})
	}
	if coin.Material != "" {
		rows = append(rows, struct{ label, value string }{"Material", string(coin.Material)})
	}
	if coin.Mint != "" {
		rows = append(rows, struct{ label, value string }{"Mint", coin.Mint})
	}
	if coin.WeightGrams != nil {
		rows = append(rows, struct{ label, value string }{"Weight", fmt.Sprintf("%.2f g", *coin.WeightGrams)})
	}
	if coin.DiameterMm != nil {
		rows = append(rows, struct{ label, value string }{"Diameter", fmt.Sprintf("%.1f mm", *coin.DiameterMm)})
	}
	if coin.RarityRating != "" {
		rows = append(rows, struct{ label, value string }{"Rarity", coin.RarityRating})
	}
	if coin.PurchasePrice != nil {
		rows = append(rows, struct{ label, value string }{"Purchase Price", fmt.Sprintf("$%.2f", *coin.PurchasePrice)})
	}
	if coin.PurchaseDate != nil {
		rows = append(rows, struct{ label, value string }{"Purchase Date", coin.PurchaseDate.Format("Jan 2, 2006")})
	}
	if coin.PurchaseLocation != "" {
		rows = append(rows, struct{ label, value string }{"Purchased From", coin.PurchaseLocation})
	}
	if coin.CurrentValue != nil {
		rows = append(rows, struct{ label, value string }{"Current Value", fmt.Sprintf("$%.2f", *coin.CurrentValue)})
	}

	if len(rows) == 0 {
		return y
	}

	cardH := float64(len(rows))*6.5 + 4
	pdf.RoundedRect(margin, y, contentW, cardH, 2, "1234", "F")

	y += 2
	for _, row := range rows {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(mutedR, mutedG, mutedB)
		pdf.SetXY(margin+4, y)
		pdf.CellFormat(40, 6, row.label, "", 0, "L", false, 0, "")

		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(textR, textG, textB)
		pdf.SetXY(margin+44, y)
		pdf.CellFormat(contentW-48, 6, safeStr(row.value), "", 0, "L", false, 0, "")
		y += 6.5
	}

	return y + 2
}

func writeSummaryPage(pdf *fpdf.Fpdf, coins []models.Coin) {
	pdf.AddPage()
	pdf.SetFillColor(darkR, darkG, darkB)
	pdf.Rect(0, 0, pageW, pageH, "F")

	y := margin

	// Title
	pdf.SetTextColor(goldR, goldG, goldB)
	pdf.SetFont("Helvetica", "B", 20)
	pdf.SetXY(margin, y)
	pdf.CellFormat(contentW, 10, "Collection Summary", "", 1, "C", false, 0, "")
	y += 14

	pdf.SetDrawColor(goldR, goldG, goldB)
	pdf.SetLineWidth(0.3)
	pdf.Line(margin, y, pageW-margin, y)
	y += 6

	// Calculate stats
	var totalCount int
	var totalValue float64
	var totalPurchase float64
	categories := map[string]int{}
	materials := map[string]int{}

	for _, coin := range coins {
		if coin.IsWishlist || coin.IsSold {
			continue
		}
		totalCount++
		if coin.CurrentValue != nil {
			totalValue += *coin.CurrentValue
		} else if coin.PurchasePrice != nil {
			totalValue += *coin.PurchasePrice
		}
		if coin.PurchasePrice != nil {
			totalPurchase += *coin.PurchasePrice
		}
		if coin.Category != "" {
			categories[string(coin.Category)]++
		}
		if coin.Material != "" {
			materials[string(coin.Material)]++
		}
	}

	// Summary stats
	summaryRows := []struct{ label, value string }{
		{"Total Coins", fmt.Sprintf("%d", totalCount)},
		{"Estimated Value", fmt.Sprintf("$%.2f", totalValue)},
		{"Total Invested", fmt.Sprintf("$%.2f", totalPurchase)},
	}

	pdf.SetFillColor(cardBgR, cardBgG, cardBgB)
	cardH := float64(len(summaryRows))*8 + 8
	pdf.RoundedRect(margin, y, contentW, cardH, 2, "1234", "F")
	y += 4

	for _, row := range summaryRows {
		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetTextColor(mutedR, mutedG, mutedB)
		pdf.SetXY(margin+6, y)
		pdf.CellFormat(60, 7, row.label, "", 0, "L", false, 0, "")

		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetTextColor(goldR, goldG, goldB)
		pdf.SetXY(margin+66, y)
		pdf.CellFormat(contentW-72, 7, row.value, "", 0, "L", false, 0, "")
		y += 8
	}
	y += 8

	// Category breakdown
	if len(categories) > 0 {
		pdf.SetFont("Helvetica", "B", 12)
		pdf.SetTextColor(goldR, goldG, goldB)
		pdf.SetXY(margin, y)
		pdf.CellFormat(contentW, 7, "By Category", "", 1, "L", false, 0, "")
		y += 9

		for cat, count := range categories {
			pdf.SetFont("Helvetica", "", 10)
			pdf.SetTextColor(textR, textG, textB)
			pdf.SetXY(margin+6, y)
			pdf.CellFormat(60, 6, cat, "", 0, "L", false, 0, "")
			pdf.SetXY(margin+66, y)
			pdf.CellFormat(30, 6, fmt.Sprintf("%d", count), "", 0, "L", false, 0, "")
			y += 6.5
		}
		y += 4
	}

	// Material breakdown
	if len(materials) > 0 {
		pdf.SetFont("Helvetica", "B", 12)
		pdf.SetTextColor(goldR, goldG, goldB)
		pdf.SetXY(margin, y)
		pdf.CellFormat(contentW, 7, "By Material", "", 1, "L", false, 0, "")
		y += 9

		for mat, count := range materials {
			pdf.SetFont("Helvetica", "", 10)
			pdf.SetTextColor(textR, textG, textB)
			pdf.SetXY(margin+6, y)
			pdf.CellFormat(60, 6, mat, "", 0, "L", false, 0, "")
			pdf.SetXY(margin+66, y)
			pdf.CellFormat(30, 6, fmt.Sprintf("%d", count), "", 0, "L", false, 0, "")
			y += 6.5
		}
	}

	// Footer
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(mutedR, mutedG, mutedB)
	pdf.SetXY(margin, pageH-margin-5)
	pdf.CellFormat(contentW, 5, fmt.Sprintf("Generated on %s — For insurance and documentation purposes", time.Now().Format("January 2, 2006")), "", 0, "C", false, 0, "")
}

// getDisplayImages returns primary + secondary images in preferred order.
func getDisplayImages(images []models.CoinImage) []models.CoinImage {
	if len(images) == 0 {
		return nil
	}
	var primary *models.CoinImage
	var obverse, reverse *models.CoinImage
	var others []models.CoinImage

	for i := range images {
		if images[i].IsPrimary {
			primary = &images[i]
		}
		switch images[i].ImageType {
		case "obverse":
			obverse = &images[i]
		case "reverse":
			reverse = &images[i]
		default:
			others = append(others, images[i])
		}
	}

	result := []models.CoinImage{}
	if obverse != nil {
		result = append(result, *obverse)
	}
	if reverse != nil {
		result = append(result, *reverse)
	}
	if primary != nil && len(result) == 0 {
		result = append(result, *primary)
	}
	result = append(result, others...)
	return result
}

func registerImage(pdf *fpdf.Fpdf, path string, x, y, maxW, maxH float64, logger *services.Logger) {
	ext := strings.ToLower(filepath.Ext(path))
	var imgType string
	switch ext {
	case ".jpg", ".jpeg":
		imgType = "JPEG"
	case ".png":
		imgType = "PNG"
	default:
		logger.Warn("pdf", "Unsupported image format %s, skipping", ext)
		return
	}

	opts := fpdf.ImageOptions{ImageType: imgType, ReadDpi: true}
	info := pdf.RegisterImageOptions(path, opts)
	if info == nil {
		logger.Warn("pdf", "Failed to register image %s", path)
		return
	}

	imgW := info.Width()
	imgH := info.Height()
	if imgW == 0 || imgH == 0 {
		return
	}

	// Scale to fit within maxW x maxH
	scale := maxW / imgW
	if imgH*scale > maxH {
		scale = maxH / imgH
	}
	finalW := imgW * scale
	finalH := imgH * scale

	// Center within the allocated space
	offsetX := x + (maxW-finalW)/2
	offsetY := y + (maxH-finalH)/2

	pdf.ImageOptions(path, offsetX, offsetY, finalW, finalH, false, opts, 0, "")
}

// safeStr ensures the string is safe for PDF output (strip non-printable chars).
func safeStr(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, s)
}
