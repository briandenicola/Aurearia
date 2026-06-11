package services

import "testing"

func TestParseWatchlistAcceptsCurrentAbsoluteLotLinks(t *testing.T) {
	html := `
		<section>
			<h2>Watched Lots in Current Auctions</h2>
			<div class="lot">
				<a href="https://www.numisbids.com/sale/10749/lot/10003">
					GREEK EASTERN EUROPE, Imitations of Alexander III of Macedon.
					1st century BC. Silver Drachm (3.55g).
				</a>
				<img src="//images.numisbids.com/sales/hosted/status/10749/image10003.jpg">
				<span>Estimate: 100 AUD</span>
			</div>
		</section>`

	lots := NewNumisBidsService().ParseWatchlist(html)
	if len(lots) != 1 {
		t.Fatalf("ParseWatchlist returned %d lots, want 1", len(lots))
	}

	lot := lots[0]
	if lot.URL != "https://www.numisbids.com/sale/10749/lot/10003" {
		t.Fatalf("URL = %q, want canonical NumisBids lot URL", lot.URL)
	}
	if lot.SaleID != "10749" {
		t.Fatalf("SaleID = %q, want 10749", lot.SaleID)
	}
	if lot.LotNumber != 10003 {
		t.Fatalf("LotNumber = %d, want 10003", lot.LotNumber)
	}
	if lot.Title == "" {
		t.Fatal("Title is empty")
	}
	if lot.ImageURL != "https://images.numisbids.com/sales/hosted/status/10749/image10003.jpg" {
		t.Fatalf("ImageURL = %q, want protocol-normalized image URL", lot.ImageURL)
	}
	if lot.Estimate == nil || *lot.Estimate != 100 {
		t.Fatalf("Estimate = %v, want 100", lot.Estimate)
	}
	if lot.Currency != "AUD" {
		t.Fatalf("Currency = %q, want AUD", lot.Currency)
	}
}

func TestParseWatchlistAcceptsLegacyLotLinks(t *testing.T) {
	html := `
		<a href='/n.php?p=lot&sid=7996&lot=10003'>Status International Auction 406, Lot 10003</a>
		<span>Estimate: 150 USD</span>`

	lots := NewNumisBidsService().ParseWatchlist(html)
	if len(lots) != 1 {
		t.Fatalf("ParseWatchlist returned %d lots, want 1", len(lots))
	}
	if lots[0].URL != "https://www.numisbids.com/n.php?p=lot&sid=7996&lot=10003" {
		t.Fatalf("URL = %q, want preserved legacy NumisBids lot URL", lots[0].URL)
	}
	if lots[0].SaleID != "7996" {
		t.Fatalf("SaleID = %q, want 7996", lots[0].SaleID)
	}
	if lots[0].LotNumber != 10003 {
		t.Fatalf("LotNumber = %d, want 10003", lots[0].LotNumber)
	}
	if lots[0].Title != "Status International Auction 406, Lot 10003" {
		t.Fatalf("Title = %q, want legacy link title", lots[0].Title)
	}
}

func TestParseWatchlistIgnoresNonNumisBidsAbsoluteLinks(t *testing.T) {
	html := `<a href="https://example.com/sale/10749/lot/10003">External Lot</a>`

	lots := NewNumisBidsService().ParseWatchlist(html)
	if len(lots) != 0 {
		t.Fatalf("ParseWatchlist returned %d lots, want 0", len(lots))
	}
}
