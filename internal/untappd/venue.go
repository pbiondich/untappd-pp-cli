package untappd

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// NoVenueRatingNote explains a missing venue-level score. Untappd venue
// pages do not publish a global venue rating the way beer pages do.
const NoVenueRatingNote = "Untappd page has no public venue score"

const MilesToMeters = 1609.344

type Venue struct {
	ID            int64    `json:"id"`
	Slug          string   `json:"slug,omitempty"`
	Name          string   `json:"name"`
	Category      string   `json:"category,omitempty"`
	Address       string   `json:"address,omitempty"`
	City          string   `json:"city,omitempty"`
	State         string   `json:"state,omitempty"`
	Country       string   `json:"country,omitempty"`
	Lat           *float64 `json:"lat"`
	Lng           *float64 `json:"lng"`
	Popularity    *int64   `json:"popularity,omitempty"`
	Popularity30  *int64   `json:"popularity_30,omitempty"`
	HasBeer       bool     `json:"has_beer"`
	Closed        bool     `json:"closed,omitempty"`
	URL           string   `json:"url,omitempty"`
	DistanceMi    *float64 `json:"distance_mi,omitempty"`
	Rating        *float64 `json:"rating"`
	RatingCount   *int64   `json:"rating_count"`
	RatingPresent bool     `json:"rating_present"`
	RatingNote    string   `json:"rating_note,omitempty"`
}

type VenueMenuBeer struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Style         string   `json:"style,omitempty"`
	Brewery       string   `json:"brewery,omitempty"`
	ABV           *float64 `json:"abv"`
	IBU           *float64 `json:"ibu"`
	Rating        *float64 `json:"rating"`
	RatingPresent bool     `json:"rating_present"`
	RatingNote    string   `json:"rating_note,omitempty"`
	URL           string   `json:"url,omitempty"`
}

type Place struct {
	Name   string  `json:"name,omitempty"`
	Query  string  `json:"query,omitempty"`
	Lat    float64 `json:"lat"`
	Lng    float64 `json:"lng"`
	Source string  `json:"source"`
	URL    string  `json:"url,omitempty"`
}

type NearbyResult struct {
	Origin   Place   `json:"origin"`
	RadiusMi float64 `json:"radius_mi"`
	Sort     string  `json:"sort"`
	BeerOnly bool    `json:"beer_only"`
	Venues   []Venue `json:"venues"`
}

var (
	reVenueIDAttr   = regexp.MustCompile(`data-venue-id=["'](\d+)["']`)
	reVenueSlugAttr = regexp.MustCompile(`data-venue-slug=["']([^"']+)["']`)
	reVenueH2       = regexp.MustCompile(`(?is)<h2>(.*?)</h2>`)
	reAddress       = regexp.MustCompile(`(?is)<p[^>]*class=["'][^"']*\baddress\b[^"']*["'][^>]*>(.*?)</p>`)
	reOGLat         = regexp.MustCompile(`(?is)<meta[^>]+property=["']place:location:latitude["'][^>]*content=["']([^"']+)["']`)
	reOGLng         = regexp.MustCompile(`(?is)<meta[^>]+property=["']place:location:longitude["'][^>]*content=["']([^"']+)["']`)
	reVenuePath     = regexp.MustCompile(`(?i)/v/([^/]+)/(\d+)`)
	reMenuItem      = regexp.MustCompile(`(?is)<li[^>]*class=["'][^"']*menu-item[^"']*["'][^>]*>.*?</li>`)
	reBeerHref      = regexp.MustCompile(`(?is)href=["'](/b/[^"']+)["']`)
	reBeerStyleEm   = regexp.MustCompile(`(?is)<em>(.*?)</em>`)
	reMenuBrewery   = regexp.MustCompile(`(?is)href=["']/w/[^"']+["'][^>]*>(.*?)</a>`)
	reMenuABVIBU    = regexp.MustCompile(`(?is)([0-9.]+%|N/A)\s*ABV\s*•\s*([^•<]+)`)
	reHandlebars    = regexp.MustCompile(`\{\{`)
)

func NormalizeVenueHits(raw json.RawMessage) ([]Venue, error) {
	hits, err := extractHits(raw)
	if err != nil {
		return nil, err
	}
	out := make([]Venue, 0, len(hits))
	for _, hit := range hits {
		v := Venue{
			ID:           intFrom(hit, "venue_id"),
			Name:         stringFrom(hit, "venue_name"),
			Slug:         stringFrom(hit, "venue_slug"),
			Category:     firstNonEmpty(stringFrom(hit, "venue_primary_category"), stringFrom(hit, "venue_categories")),
			Address:      firstNonEmpty(stringFrom(hit, "venue_address"), stringFrom(hit, "venue_full_address")),
			City:         stringFrom(hit, "venue_city"),
			State:        stringFrom(hit, "venue_state"),
			Country:      stringFrom(hit, "venue_country"),
			Popularity:   intPtrFrom(hit, "popularity"),
			Popularity30: intPtrFrom(hit, "popularity_30"),
			HasBeer:      truthy(hit["has_beer"]),
			Closed:       truthy(hit["is_closed"]),
		}
		if geo, ok := hit["_geoloc"].(map[string]any); ok {
			if lat, ok := asFloat(geo["lat"]); ok {
				v.Lat = &lat
			}
			if lng, ok := asFloat(geo["lng"]); ok {
				v.Lng = &lng
			}
		}
		v.Rating, v.RatingCount, v.RatingPresent, v.RatingNote = ApplyRating(nil, nil)
		if v.RatingNote == NoRatingNote {
			v.RatingNote = NoVenueRatingNote
		}
		if v.ID > 0 && v.Slug != "" {
			v.URL = fmt.Sprintf("https://untappd.com/v/%s/%d", v.Slug, v.ID)
		} else if v.ID > 0 {
			v.URL = fmt.Sprintf("https://untappd.com/venue/%d", v.ID)
		}
		out = append(out, v)
	}
	return out, nil
}

func NormalizeVenueHitsJSON(raw json.RawMessage) (json.RawMessage, error) {
	venues, err := NormalizeVenueHits(raw)
	if err != nil {
		return nil, err
	}
	return MarshalJSON(venues)
}

func ParseVenuePage(raw []byte) (*Venue, error) {
	s := string(raw)
	if strings.TrimSpace(s) == "" {
		return nil, fmt.Errorf("empty venue page")
	}
	v := &Venue{
		Name:     cleanText(firstSub(reH1, s)),
		Category: cleanText(firstSub(reVenueH2, s)),
		Address:  cleanVenueAddress(firstSub(reAddress, s)),
		URL:      firstNonEmpty(firstAlt(reCanonical, s), firstSub(reOGURL, s)),
	}
	if m := reVenueIDAttr.FindStringSubmatch(s); len(m) >= 2 {
		v.ID = parseInt64(m[1])
	}
	if m := reVenueSlugAttr.FindStringSubmatch(s); len(m) >= 2 {
		v.Slug = m[1]
	}
	if v.URL != "" {
		if m := reVenuePath.FindStringSubmatch(v.URL); len(m) >= 3 {
			if v.Slug == "" {
				v.Slug = m[1]
			}
			if v.ID == 0 {
				v.ID = parseInt64(m[2])
			}
		}
	}
	if lat := parseCoord(firstSub(reOGLat, s)); lat != nil {
		v.Lat = lat
	}
	if lng := parseCoord(firstSub(reOGLng, s)); lng != nil {
		v.Lng = lng
	}
	if ld := parseVenueJSONLD(s); ld != nil {
		if v.Name == "" {
			v.Name = ld.Name
		}
		if v.Address == "" {
			v.Address = ld.Address
		}
		if v.City == "" {
			v.City = ld.City
		}
		if v.State == "" {
			v.State = ld.State
		}
		if v.Lat == nil {
			v.Lat = ld.Lat
		}
		if v.Lng == nil {
			v.Lng = ld.Lng
		}
	}
	v.Rating, v.RatingCount, v.RatingPresent, v.RatingNote = ApplyRating(nil, nil)
	v.RatingNote = NoVenueRatingNote
	if v.Name == "" {
		return nil, fmt.Errorf("venue page missing name")
	}
	if v.URL == "" && v.ID > 0 {
		if v.Slug != "" {
			v.URL = fmt.Sprintf("https://untappd.com/v/%s/%d", v.Slug, v.ID)
		} else {
			v.URL = fmt.Sprintf("https://untappd.com/venue/%d", v.ID)
		}
	}
	return v, nil
}

func ParseVenueMenuBeers(raw []byte) ([]VenueMenuBeer, error) {
	s := string(raw)
	seen := map[int64]bool{}
	var out []VenueMenuBeer
	for _, chunk := range reMenuItem.FindAllString(s, -1) {
		if reHandlebars.MatchString(chunk) {
			continue
		}
		href := firstSub(reBeerHref, chunk)
		if href == "" {
			continue
		}
		item := VenueMenuBeer{
			Style: cleanText(firstSub(reBeerStyleEm, chunk)),
			URL:   absolutize(href),
		}
		if m := regexp.MustCompile(`(?is)<a[^>]+href=["']/b/[^"']+["'][^>]*>(.*?)</a>`).FindStringSubmatch(chunk); len(m) >= 2 {
			item.Name = cleanText(m[1])
		}
		if m := reBeerPath.FindStringSubmatch(href); len(m) >= 3 {
			item.ID = parseInt64(m[2])
		}
		if item.ID == 0 || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		item.Brewery = cleanText(firstSub(reMenuBrewery, chunk))
		if m := reMenuABVIBU.FindStringSubmatch(chunk); len(m) >= 3 {
			item.ABV = parseMeasure(m[1])
			item.IBU = parseMeasure(m[2])
		}
		score, _ := pageRating(chunk)
		item.Rating, _, item.RatingPresent, item.RatingNote = ApplyRating(score, nil)
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := out[i].Rating, out[j].Rating
		if ri == nil && rj == nil {
			return out[i].Name < out[j].Name
		}
		if ri == nil {
			return false
		}
		if rj == nil {
			return true
		}
		return *ri > *rj
	})
	return out, nil
}

func ParsePhoton(raw []byte, query string) (*Place, error) {
	var fc struct {
		Features []struct {
			Geometry struct {
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
			Properties map[string]any `json:"properties"`
		} `json:"features"`
	}
	if err := json.Unmarshal(raw, &fc); err != nil {
		return nil, fmt.Errorf("parse photon: %w", err)
	}
	if len(fc.Features) == 0 || len(fc.Features[0].Geometry.Coordinates) < 2 {
		return nil, fmt.Errorf("photon returned no coordinates for %q", query)
	}
	lng := fc.Features[0].Geometry.Coordinates[0]
	lat := fc.Features[0].Geometry.Coordinates[1]
	props := fc.Features[0].Properties
	name := firstNonEmpty(asString(props["name"]), query)
	city := asString(props["city"])
	if city != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(city)) {
		name = name + ", " + city
	}
	return &Place{Name: name, Query: query, Lat: lat, Lng: lng, Source: "photon"}, nil
}

func PhotonURL(query string) string {
	u, _ := url.Parse("https://photon.komoot.io/api/")
	q := u.Query()
	q.Set("q", query)
	q.Set("limit", "1")
	u.RawQuery = q.Encode()
	return u.String()
}

func DistanceMiles(lat1, lng1, lat2, lng2 float64) float64 {
	const earthMi = 3958.7613
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthMi * math.Asin(math.Min(1, math.Sqrt(a)))
}

func AttachDistances(venues []Venue, lat, lng float64) {
	for i := range venues {
		if venues[i].Lat == nil || venues[i].Lng == nil {
			continue
		}
		d := DistanceMiles(lat, lng, *venues[i].Lat, *venues[i].Lng)
		d = math.Round(d*100) / 100
		venues[i].DistanceMi = &d
	}
}

func SortVenues(venues []Venue, sortBy string) {
	switch strings.ToLower(strings.TrimSpace(sortBy)) {
	case "distance":
		sort.SliceStable(venues, func(i, j int) bool {
			di, dj := venues[i].DistanceMi, venues[j].DistanceMi
			if di == nil && dj == nil {
				return venues[i].Name < venues[j].Name
			}
			if di == nil {
				return false
			}
			if dj == nil {
				return true
			}
			return *di < *dj
		})
	case "recent":
		sort.SliceStable(venues, func(i, j int) bool {
			return intOrZero(venues[i].Popularity30) > intOrZero(venues[j].Popularity30)
		})
	default: // popularity
		sort.SliceStable(venues, func(i, j int) bool {
			return intOrZero(venues[i].Popularity) > intOrZero(venues[j].Popularity)
		})
	}
}

func RadiusMeters(miles float64) int {
	if miles <= 0 {
		miles = 2
	}
	return int(math.Round(miles * MilesToMeters))
}

func AlgoliaAroundBody(query string, lat, lng float64, radiusMi float64, beerOnly bool, hits int) map[string]any {
	if hits <= 0 {
		hits = 25
	}
	if hits > 50 {
		hits = 50
	}
	body := map[string]any{
		"query":          query,
		"hitsPerPage":    hits,
		"aroundLatLng":   fmt.Sprintf("%f,%f", lat, lng),
		"aroundRadius":   RadiusMeters(radiusMi),
		"getRankingInfo": true,
	}
	filters := []string{"is_closed:0"}
	if beerOnly {
		filters = append(filters, "has_beer:1")
	}
	body["filters"] = strings.Join(filters, " AND ")
	return body
}

func VenueFromAnchor(v Venue, query string) *Place {
	if v.Lat == nil || v.Lng == nil {
		return nil
	}
	return &Place{
		Name:   v.Name,
		Query:  query,
		Lat:    *v.Lat,
		Lng:    *v.Lng,
		Source: "untappd-venue",
		URL:    v.URL,
	}
}

type venueLD struct {
	Name    string
	Address string
	City    string
	State   string
	Lat     *float64
	Lng     *float64
}

func parseVenueJSONLD(s string) *venueLD {
	m := reJSONLD.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(htmlUnescape(m[1])), &raw); err != nil {
		return nil
	}
	out := &venueLD{Name: asString(raw["name"])}
	if addr, ok := raw["address"].(map[string]any); ok {
		out.Address = asString(addr["streetAddress"])
		out.City = asString(addr["addressLocality"])
		out.State = asString(addr["addressRegion"])
	}
	if geo, ok := raw["geo"].(map[string]any); ok {
		if lat, ok := asFloat(geo["latitude"]); ok {
			out.Lat = &lat
		}
		if lng, ok := asFloat(geo["longitude"]); ok {
			out.Lng = &lng
		}
	}
	return out
}

func htmlUnescape(s string) string {
	return strings.NewReplacer(`\/`, `/`).Replace(s)
}

func cleanVenueAddress(s string) string {
	s = regexp.MustCompile(`(?is)\([^)]*Map[^)]*\)`).ReplaceAllString(s, "")
	return cleanText(s)
}

func parseCoord(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case json.Number:
		n, _ := t.Int64()
		return n != 0
	case string:
		return t == "1" || strings.EqualFold(t, "true")
	default:
		n, ok := asInt(v)
		return ok && n != 0
	}
}

func intOrZero(n *int64) int64 {
	if n == nil {
		return 0
	}
	return *n
}
