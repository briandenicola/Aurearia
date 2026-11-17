package services

import (
	"os"
	"strings"
	"testing"
)

// testdata/vcoins/product_ptolemaic_tetradrachm.html is a trimmed copy of a
// real vCoins product page: ASP.NET wraps the whole page in one <form>.
func TestCleanWishlistListingHTMLKeepsASPNetPageForm(t *testing.T) {
	raw, err := os.ReadFile("testdata/vcoins/product_ptolemaic_tetradrachm.html")
	if err != nil {
		t.Fatal(err)
	}
	title, text, metadata, err := cleanWishlistListingHTML(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(title, "PTOLEMAIC KINGS of EGYPT.") ||
		metadata.ImageURL != "https://images.vcoins.com/product_image/89/7/5/7Yzq64RjgEF5DS8i9CcLWeo2e6aL3K.jpg" {
		t.Fatalf("unexpected metadata: title=%q metadata=%+v", title, metadata)
	}
	for _, want := range []string{
		"AR Tetradrachm (24mm, 14.13g., 12h). Alexandreia mint. Dated RY 8 (110/09 BC).",
		"Diademed head of Ptolemy I right, wearing aegis around neck",
		"CPE 1513; Svoronos 1669; Olivier 5484–74; SNG Copenhagen 352; DCA2 1144. Good VF, toned",
		"360.00",
		"Listed price: 360 GBP",
		"Availability: in stock",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("cleaned text missing %q:\n%s", want, text)
		}
	}
	for _, excluded := range []string{
		"Magnentius", "165.00", // other items from the store
		"Search in this store", "__VIEWSTATE", "Add To Cart", "var theForm",
	} {
		if strings.Contains(text, excluded) {
			t.Errorf("cleaned text retained %q:\n%s", excluded, text)
		}
	}
}

func TestCleanWishlistListingHTMLStillDropsSmallForms(t *testing.T) {
	_, text, _, err := cleanWishlistListingHTML(strings.NewReader(`
		<html><body><main><h1>Hadrian Denarius</h1><p>Silver denarius</p>
		<form><label>Your bid</label><select><option>USD 100</option></select>
		<textarea>Message the dealer</textarea></form>
		<form class="newsletter-signup"><h2>Join us</h2><p>Weekly offers</p></form>
		</main></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Silver denarius") {
		t.Fatalf("listing text missing: %q", text)
	}
	for _, excluded := range []string{"Your bid", "USD 100", "Message the dealer", "Weekly offers"} {
		if strings.Contains(text, excluded) {
			t.Errorf("cleaned text retained %q: %q", excluded, text)
		}
	}
}

func TestWishlistURLOfferLines(t *testing.T) {
	cases := []struct {
		name string
		json string
		want []string
	}{
		{"single offer", `{"@type":"Product","offers":{"price":"360","priceCurrency":"GBP","availability":"https://schema.org/InStock"}}`,
			[]string{"Listed price: 360 GBP", "Availability: in stock"}},
		{"numeric price in offer list", `{"@type":["Product"],"offers":[{"price":1250.5,"priceCurrency":"usd","availability":"http://schema.org/SoldOut"}]}`,
			[]string{"Listed price: 1250.5 USD", "Availability: sold"}},
		{"graph", `{"@graph":[{"@type":"WebPage"},{"@type":"Product","offers":{"price":"95.00","priceCurrency":"EUR"}}]}`,
			[]string{"Listed price: 95.00 EUR"}},
		{"not a product", `{"@type":"Store","priceRange":"$$","offers":{"price":"1","priceCurrency":"USD"}}`, nil},
		{"missing currency", `{"@type":"Product","offers":{"price":"125"}}`, nil},
		{"non-numeric price", `{"@type":"Product","offers":{"price":"Ask","priceCurrency":"USD"}}`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, text, _, err := cleanWishlistListingHTML(strings.NewReader(
				`<html><head><script type="application/ld+json">` + tc.json + `</script></head><body></body></html>`))
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tc.want {
				if !strings.Contains(text, want) {
					t.Errorf("missing %q in %q", want, text)
				}
			}
			if tc.want == nil && (strings.Contains(text, "Listed price:") || strings.Contains(text, "Availability:")) {
				t.Errorf("unexpected offer line in %q", text)
			}
		})
	}
}
