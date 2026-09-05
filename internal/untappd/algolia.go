package untappd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Published frontend Algolia credentials from window.UNTAPPD_SEARCH_CONFIG
// on https://untappd.com/search. This is a search-only key Untappd embeds
// in public HTML, not a user secret.
const (
	DefaultAlgoliaAppID     = "9WBO4RQ3HO"
	DefaultAlgoliaSearchKey = "1d347324d67ec472bb7132c66aead485"
	AlgoliaHost             = "https://9WBO4RQ3HO-dsn.algolia.net"
)

func AlgoliaHeaders() map[string]string {
	appID := strings.TrimSpace(os.Getenv("UNTAPPD_ALGOLIA_APP_ID"))
	if appID == "" {
		appID = DefaultAlgoliaAppID
	}
	key := strings.TrimSpace(os.Getenv("UNTAPPD_ALGOLIA_SEARCH_KEY"))
	if key == "" {
		key = DefaultAlgoliaSearchKey
	}
	return map[string]string{
		"X-Algolia-Application-Id": appID,
		"X-Algolia-API-Key":        key,
		"Content-Type":             "application/json",
	}
}

func AlgoliaQueryURL(index string) string {
	return fmt.Sprintf("%s/1/indexes/%s/query", AlgoliaHost, index)
}

func NormalizeBeerHits(raw json.RawMessage) (json.RawMessage, error) {
	hits, err := extractHits(raw)
	if err != nil {
		return nil, err
	}
	out := make([]BeerSearchHit, 0, len(hits))
	for _, hit := range hits {
		item := BeerSearchHit{
			ID:        intFrom(hit, "bid"),
			Name:      stringFrom(hit, "beer_name"),
			Brewery:   stringFrom(hit, "brewery_name"),
			BreweryID: intFrom(hit, "brewery_id"),
			Style:     stringFrom(hit, "type_name"),
			Slug:      stringFrom(hit, "beer_slug"),
			ABV:       floatPtrFrom(hit, "beer_abv", false),
			IBU:       floatPtrFrom(hit, "beer_ibu", true),
		}
		score := floatPtrFrom(hit, "rating_score", true)
		count := intPtrFrom(hit, "rating_count")
		item.Rating, item.RatingCount, item.RatingPresent, item.RatingNote = ApplyRating(score, count)
		if item.ID > 0 && item.Slug != "" {
			item.URL = fmt.Sprintf("https://untappd.com/b/%s/%d", item.Slug, item.ID)
		} else if item.ID > 0 {
			item.URL = fmt.Sprintf("https://untappd.com/beer/%d", item.ID)
		}
		out = append(out, item)
	}
	return MarshalJSON(out)
}

func NormalizeBreweryHits(raw json.RawMessage) (json.RawMessage, error) {
	hits, err := extractHits(raw)
	if err != nil {
		return nil, err
	}
	out := make([]BrewerySearchHit, 0, len(hits))
	for _, hit := range hits {
		page := stringFrom(hit, "brewery_page_url")
		item := BrewerySearchHit{
			ID:        intFrom(hit, "brewery_id"),
			Name:      stringFrom(hit, "brewery_name"),
			City:      stringFrom(hit, "brewery_city"),
			State:     stringFrom(hit, "brewery_state"),
			Country:   stringFrom(hit, "brewery_country"),
			Type:      stringFrom(hit, "brewery_type"),
			BeerCount: intPtrFrom(hit, "brewery_beer_count"),
			Slug:      strings.TrimPrefix(page, "/"),
		}
		if page != "" {
			item.URL = absolutize(page)
		}
		out = append(out, item)
	}
	return MarshalJSON(out)
}

func extractHits(raw json.RawMessage) ([]map[string]any, error) {
	var wrapped struct {
		Hits []map[string]any `json:"hits"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Hits != nil {
		return wrapped.Hits, nil
	}
	var hits []map[string]any
	if err := json.Unmarshal(raw, &hits); err == nil {
		return hits, nil
	}
	return nil, fmt.Errorf("decode Algolia hits: unexpected payload")
}

func stringFrom(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func intFrom(m map[string]any, key string) int64 {
	if n, ok := asInt(m[key]); ok {
		return n
	}
	return 0
}

func intPtrFrom(m map[string]any, key string) *int64 {
	if _, ok := m[key]; !ok || m[key] == nil {
		return nil
	}
	n, ok := asInt(m[key])
	if !ok {
		return nil
	}
	return &n
}

func floatPtrFrom(m map[string]any, key string, zeroMeansAbsent bool) *float64 {
	if _, ok := m[key]; !ok || m[key] == nil {
		return nil
	}
	v, ok := asFloat(m[key])
	if !ok {
		return nil
	}
	if zeroMeansAbsent && v == 0 {
		return nil
	}
	return &v
}
