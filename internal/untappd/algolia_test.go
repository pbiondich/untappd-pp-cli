package untappd

import (
	"encoding/json"
	"testing"
)

func TestNormalizeBeerHits(t *testing.T) {
	raw, err := NormalizeBeerHits(fixture(t, "search-beer-algolia.json"))
	if err != nil {
		t.Fatal(err)
	}
	var hits []BeerSearchHit
	if err := json.Unmarshal(raw, &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("len=%d", len(hits))
	}
	cool := hits[0]
	if cool.Name != "Cool Bay" || cool.ID != 4384886 {
		t.Fatalf("first=%+v", cool)
	}
	if !cool.RatingPresent || cool.Rating == nil || *cool.Rating != 4.04 {
		t.Fatalf("rating=%v present=%v", cool.Rating, cool.RatingPresent)
	}
	if cool.URL != "https://untappd.com/b/hop-butcher-for-the-world-cool-bay/4384886" {
		t.Fatalf("url=%q", cool.URL)
	}
	if cool.IBU != nil {
		t.Fatalf("ibu 0 should be null, got %v", *cool.IBU)
	}
	unrated := hits[1]
	if unrated.RatingPresent || unrated.Rating != nil {
		t.Fatalf("invented search rating: %+v", unrated)
	}
	if unrated.RatingNote != NoRatingNote {
		t.Fatalf("note=%q", unrated.RatingNote)
	}
}

func TestNormalizeBreweryHits(t *testing.T) {
	raw, err := NormalizeBreweryHits(fixture(t, "search-brewery-algolia.json"))
	if err != nil {
		t.Fatal(err)
	}
	var hits []BrewerySearchHit
	if err := json.Unmarshal(raw, &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Name != "Hop Butcher For The World" {
		t.Fatalf("hits=%+v", hits)
	}
	if hits[0].ID != 23570 || hits[0].Slug != "HopButcher" {
		t.Fatalf("id/slug=%+v", hits[0])
	}
	if hits[0].URL != "https://untappd.com/HopButcher" {
		t.Fatalf("url=%q", hits[0].URL)
	}
}

func TestAlgoliaHeaders(t *testing.T) {
	h := AlgoliaHeaders()
	if h["X-Algolia-Application-Id"] != DefaultAlgoliaAppID {
		t.Fatalf("app id=%q", h["X-Algolia-Application-Id"])
	}
	if h["X-Algolia-API-Key"] == "" {
		t.Fatal("missing search key")
	}
}
