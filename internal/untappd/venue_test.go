package untappd

import (
	"encoding/json"
	"math"
	"testing"
)

func TestCleanCategoryString(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"[Brewery]", "Brewery"},
		{"[Brewery American Restaurant]", "Brewery American Restaurant"},
		{"Dining and Drinking", "Dining and Drinking"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := cleanCategoryString(tc.in); got != tc.want {
			t.Fatalf("cleanCategoryString(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
	if got := cleanCategoryValue([]any{"Brewery", "American Restaurant"}); got != "Brewery / American Restaurant" {
		t.Fatalf("array category=%q", got)
	}
}

func TestNormalizeVenueHits_CleansStringifiedCategory(t *testing.T) {
	raw := []byte(`{"hits":[{"venue_id":1,"venue_name":"Modist","venue_slug":"modist","venue_primary_category":"[Brewery]","_geoloc":{"lat":1,"lng":2}}]}`)
	venues, err := NormalizeVenueHits(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(venues) != 1 || venues[0].Category != "Brewery" {
		t.Fatalf("category=%q venues=%+v", venues[0].Category, venues)
	}
}

func TestNormalizeVenueHits(t *testing.T) {
	venues, err := NormalizeVenueHits(fixture(t, "search-venue-algolia.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(venues) != 2 {
		t.Fatalf("len=%d", len(venues))
	}
	hotel := venues[0]
	if hotel.ID != 8255451 || hotel.Name != "Elliot Park Hotel, Autograph Collection" {
		t.Fatalf("hotel=%+v", hotel)
	}
	if hotel.Lat == nil || hotel.Lng == nil {
		t.Fatal("missing geoloc")
	}
	if hotel.RatingPresent || hotel.Rating != nil {
		t.Fatalf("invented venue rating: %+v", hotel)
	}
	if hotel.RatingNote != NoVenueRatingNote {
		t.Fatalf("note=%q", hotel.RatingNote)
	}
	if hotel.URL != "https://untappd.com/v/elliot-park-hotel-autograph-collection/8255451" {
		t.Fatalf("url=%q", hotel.URL)
	}
}

func TestParseVenuePageHotel(t *testing.T) {
	v, err := ParseVenuePage(fixture(t, "venue-elliot-park-hotel.html"))
	if err != nil {
		t.Fatal(err)
	}
	if v.Name != "Elliot Park Hotel, Autograph Collection" {
		t.Fatalf("name=%q", v.Name)
	}
	if v.ID != 8255451 || v.Slug != "elliot-park-hotel-autograph-collection" {
		t.Fatalf("id/slug=%d %q", v.ID, v.Slug)
	}
	if v.Lat == nil || math.Abs(*v.Lat-44.972332) > 0.0001 {
		t.Fatalf("lat=%v", v.Lat)
	}
	if v.Lng == nil || math.Abs(*v.Lng+93.2663956) > 0.0001 {
		t.Fatalf("lng=%v", v.Lng)
	}
	if !containsAll(v.Address, "823", "Minneapolis") {
		t.Fatalf("address=%q", v.Address)
	}
	if v.RatingPresent || v.Rating != nil {
		t.Fatalf("invented rating on hotel page: %+v", v)
	}
}

func TestParseVenuePageTownHall(t *testing.T) {
	v, err := ParseVenuePage(fixture(t, "venue-town-hall.html"))
	if err != nil {
		t.Fatal(err)
	}
	if v.Name != "Town Hall Brewery" || v.ID != 2714 {
		t.Fatalf("venue=%+v", v)
	}
	if v.Rating != nil {
		t.Fatal("must not treat check-in/menu caps as a venue score")
	}
}

func TestParseVenueMenuBeers(t *testing.T) {
	beers, err := ParseVenueMenuBeers(fixture(t, "venue-town-hall.html"))
	if err != nil {
		t.Fatal(err)
	}
	if len(beers) < 2 {
		t.Fatalf("len=%d", len(beers))
	}
	if beers[0].Rating == nil || beers[1].Rating == nil {
		t.Fatalf("expected menu ratings: %+v", beers)
	}
	if *beers[0].Rating < *beers[1].Rating {
		t.Fatalf("not sorted high-to-low: %v then %v", *beers[0].Rating, *beers[1].Rating)
	}
	empty, err := ParseVenueMenuBeers(fixture(t, "venue-elliot-park-hotel.html"))
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("hotel should have no menu beers, got %+v", empty)
	}
}

func TestParsePhoton(t *testing.T) {
	p, err := ParsePhoton(fixture(t, "geocode-photon-elliot.json"), "Elliot Park Hotel, Minneapolis")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(p.Lat-44.9720511) > 0.0001 || math.Abs(p.Lng+93.2664612) > 0.0001 {
		t.Fatalf("place=%+v", p)
	}
	if p.Source != "photon" {
		t.Fatalf("source=%q", p.Source)
	}
}

func TestDistanceAndSort(t *testing.T) {
	venues, err := NormalizeVenueHits(fixture(t, "nearby-venue-algolia.json"))
	if err != nil {
		t.Fatal(err)
	}
	AttachDistances(venues, 44.972332, -93.2663956)
	SortVenues(venues, "popularity")
	if len(venues) == 0 || venues[0].Popularity == nil {
		t.Fatal("expected popularity sort")
	}
	for i := 1; i < len(venues); i++ {
		if intOrZero(venues[i-1].Popularity) < intOrZero(venues[i].Popularity) {
			t.Fatalf("popularity order broken at %d", i)
		}
	}
	d := DistanceMiles(44.972332, -93.2663956, 44.9733315, -93.2477875)
	if d < 0.5 || d > 2 {
		t.Fatalf("unexpected town hall distance %v", d)
	}
}

func TestAlgoliaAroundBody(t *testing.T) {
	body := AlgoliaAroundBody("", 44.97, -93.26, 2, true, 15)
	if body["filters"] != "is_closed:0 AND has_beer:1" {
		t.Fatalf("filters=%v", body["filters"])
	}
	if body["aroundRadius"] != RadiusMeters(2) {
		t.Fatalf("radius=%v", body["aroundRadius"])
	}
}

func TestNearbyJSONRoundTrip(t *testing.T) {
	raw, err := NormalizeVenueHitsJSON(fixture(t, "search-venue-algolia.json"))
	if err != nil {
		t.Fatal(err)
	}
	var venues []Venue
	if err := json.Unmarshal(raw, &venues); err != nil {
		t.Fatal(err)
	}
	if venues[0].Rating != nil {
		t.Fatal("rating leaked into JSON")
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !containsFoldString(s, p) {
			return false
		}
	}
	return true
}

func containsFoldString(s, sub string) bool {
	return len(sub) == 0 || indexFoldRunes(s, sub) >= 0
}

func indexFoldRunes(s, sub string) int {
	sl, subl := []rune(s), []rune(sub)
	for i := 0; i+len(subl) <= len(sl); i++ {
		ok := true
		for j := range subl {
			a, b := sl[i+j], subl[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}
