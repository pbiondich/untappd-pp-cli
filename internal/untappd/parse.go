package untappd

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// NoRatingNote is returned whenever Untappd publishes no public score.
// Callers must leave rating null rather than inventing a number.
const NoRatingNote = "Untappd page has no public score"

type Beer struct {
	ID              int64    `json:"id"`
	Slug            string   `json:"slug,omitempty"`
	Name            string   `json:"name"`
	Brewery         string   `json:"brewery,omitempty"`
	BrewerySlug     string   `json:"brewery_slug,omitempty"`
	Style           string   `json:"style,omitempty"`
	ABV             *float64 `json:"abv"`
	IBU             *float64 `json:"ibu"`
	Rating          *float64 `json:"rating"`
	RatingCount     *int64   `json:"rating_count"`
	RatingPresent   bool     `json:"rating_present"`
	RatingNote      string   `json:"rating_note,omitempty"`
	Description     string   `json:"description,omitempty"`
	URL             string   `json:"url,omitempty"`
	CheckinsTotal   *int64   `json:"checkins_total,omitempty"`
	CheckinsUnique  *int64   `json:"checkins_unique,omitempty"`
	CheckinsMonthly *int64   `json:"checkins_monthly,omitempty"`
}

type Brewery struct {
	ID            int64    `json:"id"`
	Slug          string   `json:"slug,omitempty"`
	Name          string   `json:"name"`
	Type          string   `json:"type,omitempty"`
	Location      string   `json:"location,omitempty"`
	Rating        *float64 `json:"rating"`
	RatingCount   *int64   `json:"rating_count"`
	RatingPresent bool     `json:"rating_present"`
	RatingNote    string   `json:"rating_note,omitempty"`
	BeerCount     *int64   `json:"beer_count,omitempty"`
	URL           string   `json:"url,omitempty"`
}

type BeerListItem struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Style         string   `json:"style,omitempty"`
	ABV           *float64 `json:"abv"`
	IBU           *float64 `json:"ibu"`
	Rating        *float64 `json:"rating"`
	RatingCount   *int64   `json:"rating_count"`
	RatingPresent bool     `json:"rating_present"`
	RatingNote    string   `json:"rating_note,omitempty"`
	URL           string   `json:"url,omitempty"`
}

type BeerSearchHit struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Brewery       string   `json:"brewery,omitempty"`
	BreweryID     int64    `json:"brewery_id,omitempty"`
	Style         string   `json:"style,omitempty"`
	ABV           *float64 `json:"abv"`
	IBU           *float64 `json:"ibu"`
	Rating        *float64 `json:"rating"`
	RatingCount   *int64   `json:"rating_count"`
	RatingPresent bool     `json:"rating_present"`
	RatingNote    string   `json:"rating_note,omitempty"`
	Slug          string   `json:"slug,omitempty"`
	URL           string   `json:"url,omitempty"`
}

type BrewerySearchHit struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug,omitempty"`
	City      string `json:"city,omitempty"`
	State     string `json:"state,omitempty"`
	Country   string `json:"country,omitempty"`
	Type      string `json:"type,omitempty"`
	BeerCount *int64 `json:"beer_count,omitempty"`
	URL       string `json:"url,omitempty"`
}

var (
	reH1            = regexp.MustCompile(`(?is)<h1[^>]*>\s*(.*?)\s*</h1>`)
	reCanonical     = regexp.MustCompile(`(?is)<link[^>]+rel=["']canonical["'][^>]*href=["']([^"']+)["']|href=["']([^"']+)["'][^>]*rel=["']canonical["']`)
	reOGImage       = regexp.MustCompile(`(?is)<meta[^>]+property=["']og:image["'][^>]*content=["']([^"']+)["']`)
	reOGURL         = regexp.MustCompile(`(?is)<meta[^>]+property=["']og:url["'][^>]*content=["']([^"']+)["']`)
	reCapsRating    = regexp.MustCompile(`(?is)data-rating=["']([0-9.]+)["']`)
	reBeerItem      = regexp.MustCompile(`(?is)class=["'][^"']*beer-item[^"']*["'][^>]*data-bid=["'](\d+)["']|data-bid=["'](\d+)["'][^>]*class=["'][^"']*beer-item`)
	reBreweryAnchor = regexp.MustCompile(`(?is)<p[^>]*class=["'][^"']*\bbrewery\b[^"']*["'][^>]*>\s*<a[^>]+href=["']([^"']+)["'][^>]*>(.*?)</a>`)
	reBreweryText   = regexp.MustCompile(`(?is)<p[^>]*class=["'][^"']*\bbrewery\b[^"']*["'][^>]*>(.*?)</p>`)
	reStyle         = regexp.MustCompile(`(?is)<p[^>]*class=["'][^"']*\bstyle\b[^"']*["'][^>]*>(.*?)</p>`)
	reABV           = regexp.MustCompile(`(?is)<(?:p|div)[^>]*class=["'][^"']*\babv\b[^"']*["'][^>]*>(.*?)</(?:p|div)>`)
	reIBU           = regexp.MustCompile(`(?is)<(?:p|div)[^>]*class=["'][^"']*\bibu\b[^"']*["'][^>]*>(.*?)</(?:p|div)>`)
	reRaters        = regexp.MustCompile(`(?is)<p[^>]*class=["'][^"']*\braters\b[^"']*["'][^>]*>(.*?)</p>`)
	reDescLess      = regexp.MustCompile(`(?is)<div[^>]*class=["'][^"']*beer-descrption-read-less[^"']*["'][^>]*>(.*?)</div>`)
	reDescMore      = regexp.MustCompile(`(?is)<div[^>]*class=["'][^"']*beer-descrption-read-more[^"']*["'][^>]*>(.*?)</div>`)
	reBeerCount     = regexp.MustCompile(`(?is)<p[^>]*class=["'][^"']*\bcount\b[^"']*["'][^>]*>.*?<a[^>]*>(.*?)</a>`)
	reStatBlock     = regexp.MustCompile(`(?is)<span[^>]*class=["'][^"']*\bstat\b[^"']*["'][^>]*>(.*?)</span>\s*<span[^>]*class=["'][^"']*\bcount\b[^"']*["'][^>]*>(.*?)</span>`)
	reJSONLD        = regexp.MustCompile(`(?is)<script[^>]+type=["']application/ld\+json["'][^>]*>(.*?)</script>`)
	reBeerPath      = regexp.MustCompile(`(?i)/b/([^/]+)/(\d+)`)
	reBreweryOG     = regexp.MustCompile(`(?i)/brewery/(\d+)`)
	reBeerOG        = regexp.MustCompile(`(?i)/beer/(\d+)`)
	reNameLink      = regexp.MustCompile(`(?is)<p[^>]*class=["'][^"']*\bname\b[^"']*["'][^>]*>\s*<a[^>]+href=["']([^"']+)["'][^>]*>(.*?)</a>`)
	reItemStyle     = regexp.MustCompile(`(?is)<p[^>]*class=["'][^"']*\bstyle\b[^"']*["'][^>]*>(.*?)</p>`)
	reDetailsABV    = regexp.MustCompile(`(?is)class=["'][^"']*\babv\b[^"']*["'][^>]*>\s*([^<]+)`)
	reDetailsIBU    = regexp.MustCompile(`(?is)class=["'][^"']*\bibu\b[^"']*["'][^>]*>\s*([^<]+)`)
	reSearchConfig  = regexp.MustCompile(`(?s)window\.UNTAPPD_SEARCH_CONFIG\s*=\s*(\{.*?\});`)
	reDigits        = regexp.MustCompile(`[0-9]+(?:\.[0-9]+)?`)
	reIntCommas     = regexp.MustCompile(`(?i)([0-9][0-9,]*)`)
)

func ParseBeerPage(raw []byte) (*Beer, error) {
	s := string(raw)
	if strings.TrimSpace(s) == "" {
		return nil, fmt.Errorf("empty beer page")
	}
	beer := &Beer{
		Name:        cleanText(firstSub(reH1, s)),
		Style:       cleanText(firstSub(reStyle, s)),
		Description: firstNonEmpty(cleanText(stripReadMore(firstSub(reDescLess, s))), cleanText(stripReadMore(firstSub(reDescMore, s)))),
		URL:         firstNonEmpty(firstAlt(reCanonical, s), firstSub(reOGURL, s)),
	}
	if m := reBreweryAnchor.FindStringSubmatch(s); len(m) >= 3 {
		beer.BrewerySlug = strings.TrimPrefix(strings.TrimSpace(m[1]), "/")
		beer.Brewery = cleanText(m[2])
	}
	beer.ABV = parseMeasure(firstSub(reABV, s))
	beer.IBU = parseMeasure(firstSub(reIBU, s))

	if beer.URL != "" {
		if m := reBeerPath.FindStringSubmatch(beer.URL); len(m) >= 3 {
			beer.Slug = m[1]
			beer.ID = parseInt64(m[2])
		}
	}
	if beer.ID == 0 {
		if m := reBeerOG.FindStringSubmatch(firstSub(reOGImage, s)); len(m) >= 2 {
			beer.ID = parseInt64(m[1])
		}
	}

	score, count := pageRating(s)
	if score == nil {
		if ld := parseJSONLD(s); ld != nil {
			if score == nil {
				score = ld.Rating
			}
			if count == nil {
				count = ld.RatingCount
			}
			if beer.ID == 0 {
				beer.ID = ld.ID
			}
			if beer.Description == "" {
				beer.Description = ld.Description
			}
		}
	} else if count == nil {
		if ld := parseJSONLD(s); ld != nil && ld.RatingCount != nil {
			count = ld.RatingCount
		}
	}
	if count == nil {
		count = parseRaterCount(firstSub(reRaters, s))
	}
	beer.Rating, beer.RatingCount, beer.RatingPresent, beer.RatingNote = ApplyRating(score, count)

	for _, m := range reStatBlock.FindAllStringSubmatch(s, -1) {
		label := strings.ToLower(cleanText(m[1]))
		n := parseStatCount(cleanText(m[2]))
		if n == nil {
			continue
		}
		switch {
		case strings.Contains(label, "total"):
			beer.CheckinsTotal = n
		case strings.Contains(label, "unique"):
			beer.CheckinsUnique = n
		case strings.Contains(label, "monthly"):
			beer.CheckinsMonthly = n
		}
	}
	if beer.Name == "" {
		return nil, fmt.Errorf("beer page missing name")
	}
	if beer.URL == "" && beer.ID > 0 {
		if beer.Slug != "" {
			beer.URL = fmt.Sprintf("https://untappd.com/b/%s/%d", beer.Slug, beer.ID)
		} else {
			beer.URL = fmt.Sprintf("https://untappd.com/beer/%d", beer.ID)
		}
	}
	return beer, nil
}

func ParseBreweryPage(raw []byte) (*Brewery, error) {
	s := string(raw)
	if strings.TrimSpace(s) == "" {
		return nil, fmt.Errorf("empty brewery page")
	}
	b := &Brewery{
		Name:     cleanText(firstSub(reH1, s)),
		Type:     cleanText(firstSub(reStyle, s)),
		Location: cleanText(stripTags(firstSub(reBreweryText, s))),
		URL:      firstNonEmpty(firstAlt(reCanonical, s), firstSub(reOGURL, s)),
	}
	if b.URL != "" {
		if u, err := url.Parse(b.URL); err == nil {
			b.Slug = strings.Trim(u.Path, "/")
		}
	}
	if m := reBreweryOG.FindStringSubmatch(firstSub(reOGImage, s)); len(m) >= 2 {
		b.ID = parseInt64(m[1])
	}
	score, count := pageRating(s)
	if score == nil || count == nil {
		if ld := parseJSONLD(s); ld != nil {
			if score == nil {
				score = ld.Rating
			}
			if count == nil {
				count = ld.RatingCount
			}
			if b.Name == "" {
				b.Name = ld.Name
			}
			if b.ID == 0 {
				b.ID = ld.ID
			}
		}
	}
	if count == nil {
		count = parseRaterCount(firstSub(reRaters, s))
	}
	b.Rating, b.RatingCount, b.RatingPresent, b.RatingNote = ApplyRating(score, count)
	if n := parseStatCount(firstSub(reBeerCount, s)); n != nil {
		b.BeerCount = n
	}
	if b.Name == "" {
		return nil, fmt.Errorf("brewery page missing name")
	}
	if b.URL == "" && b.Slug != "" {
		b.URL = "https://untappd.com/" + b.Slug
	}
	return b, nil
}

func ParseBreweryBeerList(raw []byte) ([]BeerListItem, error) {
	s := string(raw)
	ids := uniqueBeerIDs(s)
	if len(ids) == 0 {
		return []BeerListItem{}, nil
	}
	items := make([]BeerListItem, 0, len(ids))
	for i, id := range ids {
		start := strings.Index(s, fmt.Sprintf(`data-bid="%d"`, id))
		if start < 0 {
			start = strings.Index(s, fmt.Sprintf(`data-bid='%d'`, id))
		}
		if start < 0 {
			continue
		}
		end := len(s)
		if i+1 < len(ids) {
			if next := strings.Index(s[start+1:], fmt.Sprintf(`data-bid="%d"`, ids[i+1])); next >= 0 {
				end = start + 1 + next
			}
		}
		chunk := s[start:end]
		item := BeerListItem{ID: id}
		if m := reNameLink.FindStringSubmatch(chunk); len(m) >= 3 {
			item.URL = absolutize(m[1])
			item.Name = cleanText(m[2])
		}
		item.Style = cleanText(firstSub(reItemStyle, chunk))
		item.ABV = parseMeasure(firstSub(reDetailsABV, chunk))
		item.IBU = parseMeasure(firstSub(reDetailsIBU, chunk))
		score, _ := pageRating(chunk)
		item.Rating, item.RatingCount, item.RatingPresent, item.RatingNote = ApplyRating(score, nil)
		if item.URL == "" && item.ID > 0 {
			item.URL = fmt.Sprintf("https://untappd.com/beer/%d", item.ID)
		}
		items = append(items, item)
	}
	return items, nil
}

func ParseSearchConfig(raw []byte) (appID, searchKey string, err error) {
	m := reSearchConfig.FindSubmatch(raw)
	if m == nil {
		return "", "", fmt.Errorf("UNTAPPD_SEARCH_CONFIG not found")
	}
	var cfg struct {
		AppID     string `json:"appId"`
		SearchKey string `json:"searchKey"`
	}
	if err := json.Unmarshal(m[1], &cfg); err != nil {
		return "", "", fmt.Errorf("parse search config: %w", err)
	}
	if cfg.AppID == "" || cfg.SearchKey == "" {
		return "", "", fmt.Errorf("search config missing appId or searchKey")
	}
	return cfg.AppID, cfg.SearchKey, nil
}

func ApplyRating(score *float64, count *int64) (*float64, *int64, bool, string) {
	if score == nil || *score <= 0 {
		return nil, count, false, NoRatingNote
	}
	if count != nil && *count <= 0 {
		return nil, count, false, NoRatingNote
	}
	return score, count, true, ""
}

func MarshalJSON(v any) (json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func uniqueBeerIDs(s string) []int64 {
	seen := map[int64]bool{}
	var ids []int64
	for _, m := range regexp.MustCompile(`data-bid=["'](\d+)["']`).FindAllStringSubmatch(s, -1) {
		id := parseInt64(m[1])
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

type jsonLD struct {
	Name        string
	Description string
	ID          int64
	Rating      *float64
	RatingCount *int64
}

func parseJSONLD(s string) *jsonLD {
	m := reJSONLD.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(html.UnescapeString(m[1])), &raw); err != nil {
		return nil
	}
	out := &jsonLD{
		Name:        asString(raw["name"]),
		Description: asString(raw["description"]),
	}
	if n, ok := asInt(raw["mpn"]); ok {
		out.ID = n
	} else if n, ok := asInt(raw["sku"]); ok {
		out.ID = n
	}
	if agg, ok := raw["aggregateRating"].(map[string]any); ok {
		if v, ok := asFloat(agg["ratingValue"]); ok {
			out.Rating = &v
		}
		if n, ok := asInt(agg["reviewCount"]); ok {
			out.RatingCount = &n
		}
	}
	return out
}

func pageRating(s string) (*float64, *int64) {
	m := reCapsRating.FindStringSubmatch(s)
	if m == nil {
		return nil, nil
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return nil, nil
	}
	return &v, parseRaterCount(firstSub(reRaters, s))
}

func parseMeasure(s string) *float64 {
	t := strings.TrimSpace(cleanText(s))
	if t == "" || strings.EqualFold(t, "n/a") || strings.Contains(strings.ToLower(t), "n/a") {
		return nil
	}
	m := reDigits.FindString(t)
	if m == "" {
		return nil
	}
	v, err := strconv.ParseFloat(m, 64)
	if err != nil {
		return nil
	}
	return &v
}

func parseRaterCount(s string) *int64 {
	t := strings.ToLower(cleanText(s))
	if t == "" || strings.Contains(t, "no rating") {
		return int64Ptr(0)
	}
	return parseStatCount(t)
}

func parseStatCount(s string) *int64 {
	t := strings.TrimSpace(cleanText(s))
	if t == "" {
		return nil
	}
	m := reIntCommas.FindString(t)
	if m == "" {
		return nil
	}
	n, err := strconv.ParseInt(strings.ReplaceAll(m, ",", ""), 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

func parseInt64(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

func int64Ptr(n int64) *int64 { return &n }

func firstSub(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func firstAlt(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	for _, g := range m[1:] {
		if strings.TrimSpace(g) != "" {
			return g
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func stripReadMore(s string) string {
	s = regexp.MustCompile(`(?is)<a[^>]*>.*?</a>`).ReplaceAllString(s, "")
	return s
}

func stripTags(s string) string {
	return regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(s, " ")
}

func cleanText(s string) string {
	s = html.UnescapeString(stripTags(s))
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

func absolutize(href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "/") {
		return "https://untappd.com" + href
	}
	return "https://untappd.com/" + href
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}

func asFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case json.Number:
		n, err := t.Float64()
		return n, err == nil
	case string:
		n, err := strconv.ParseFloat(t, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func asInt(v any) (int64, bool) {
	switch t := v.(type) {
	case float64:
		return int64(t), true
	case json.Number:
		n, err := t.Int64()
		return n, err == nil
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}
