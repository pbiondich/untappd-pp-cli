package untappd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "fixtures", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func TestParseBeerPageCoolBay(t *testing.T) {
	beer, err := ParseBeerPage(fixture(t, "beer-cool-bay.html"))
	if err != nil {
		t.Fatal(err)
	}
	if beer.Name != "Cool Bay" {
		t.Fatalf("name=%q", beer.Name)
	}
	if beer.Brewery != "Hop Butcher For The World" {
		t.Fatalf("brewery=%q", beer.Brewery)
	}
	if beer.BrewerySlug != "HopButcher" {
		t.Fatalf("brewery_slug=%q", beer.BrewerySlug)
	}
	if beer.Style != "IPA - New England / Hazy" {
		t.Fatalf("style=%q", beer.Style)
	}
	if beer.ABV == nil || *beer.ABV != 6.5 {
		t.Fatalf("abv=%v", beer.ABV)
	}
	if beer.IBU != nil {
		t.Fatalf("ibu should be null for N/A, got %v", *beer.IBU)
	}
	if !beer.RatingPresent || beer.Rating == nil || *beer.Rating < 4.0 || *beer.Rating > 4.1 {
		t.Fatalf("rating=%v present=%v", beer.Rating, beer.RatingPresent)
	}
	if beer.RatingCount == nil || *beer.RatingCount != 2914 {
		t.Fatalf("rating_count=%v", beer.RatingCount)
	}
	if beer.ID != 4384886 {
		t.Fatalf("id=%d", beer.ID)
	}
	if beer.Slug != "hop-butcher-for-the-world-cool-bay" {
		t.Fatalf("slug=%q", beer.Slug)
	}
	if beer.CheckinsTotal == nil || *beer.CheckinsTotal != 3512 {
		t.Fatalf("checkins_total=%v", beer.CheckinsTotal)
	}
	if beer.Description == "" {
		t.Fatal("missing description")
	}
}

func TestParseBeerPageNoRating(t *testing.T) {
	beer, err := ParseBeerPage(fixture(t, "beer-no-rating.html"))
	if err != nil {
		t.Fatal(err)
	}
	if beer.Name != "New Tap" {
		t.Fatalf("name=%q", beer.Name)
	}
	if beer.RatingPresent || beer.Rating != nil {
		t.Fatalf("invented rating: %+v", beer.Rating)
	}
	if beer.RatingNote != NoRatingNote {
		t.Fatalf("note=%q", beer.RatingNote)
	}
	if beer.IBU != nil {
		t.Fatalf("ibu=%v", beer.IBU)
	}
	if beer.ABV == nil || *beer.ABV != 5.0 {
		t.Fatalf("abv=%v", beer.ABV)
	}
}

func TestParseBreweryPage(t *testing.T) {
	b, err := ParseBreweryPage(fixture(t, "brewery-hop-butcher.html"))
	if err != nil {
		t.Fatal(err)
	}
	if b.Name != "Hop Butcher For The World" {
		t.Fatalf("name=%q", b.Name)
	}
	if b.ID != 23570 {
		t.Fatalf("id=%d", b.ID)
	}
	if b.Slug != "HopButcher" {
		t.Fatalf("slug=%q", b.Slug)
	}
	if b.Type != "Micro Brewery" {
		t.Fatalf("type=%q", b.Type)
	}
	if !b.RatingPresent || b.Rating == nil {
		t.Fatal("expected brewery rating")
	}
	if b.BeerCount == nil || *b.BeerCount != 733 {
		t.Fatalf("beer_count=%v", b.BeerCount)
	}
	if b.RatingCount == nil || *b.RatingCount != 1218035 {
		t.Fatalf("rating_count=%v", b.RatingCount)
	}
}

func TestParseBreweryBeerList(t *testing.T) {
	items, err := ParseBreweryBeerList(fixture(t, "brewery-beer-list.html"))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("len=%d", len(items))
	}
	if items[0].Name != "Double Grid" || items[0].ID != 2796806 {
		t.Fatalf("first=%+v", items[0])
	}
	if items[0].IBU == nil || *items[0].IBU != 65 {
		t.Fatalf("ibu=%v", items[0].IBU)
	}
	if !items[0].RatingPresent || items[0].Rating == nil {
		t.Fatal("expected rating on Double Grid")
	}
	if items[1].IBU != nil {
		t.Fatalf("second ibu should be null, got %v", items[1].IBU)
	}
	if items[2].Name != "Green Moss" {
		t.Fatalf("third=%q", items[2].Name)
	}
}

func TestParseSearchConfig(t *testing.T) {
	appID, key, err := ParseSearchConfig(fixture(t, "search-config.html"))
	if err != nil {
		t.Fatal(err)
	}
	if appID != DefaultAlgoliaAppID || key != DefaultAlgoliaSearchKey {
		t.Fatalf("appID=%s key=%s", appID, key)
	}
}

func TestApplyRatingNeverInvents(t *testing.T) {
	zero := 0.0
	rating, _, present, note := ApplyRating(&zero, int64Ptr(0))
	if rating != nil || present || note != NoRatingNote {
		t.Fatalf("zero score leaked: %v %v %q", rating, present, note)
	}
	rating, _, present, note = ApplyRating(nil, nil)
	if rating != nil || present || note != NoRatingNote {
		t.Fatalf("nil score leaked: %v %v %q", rating, present, note)
	}
}
