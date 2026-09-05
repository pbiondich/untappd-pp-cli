package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/pbiondich/untappd-pp-cli/internal/client"
	"github.com/pbiondich/untappd-pp-cli/internal/untappd"
	"github.com/spf13/cobra"
)

func htmlHeaders() map[string]string {
	return map[string]string{
		"X-Printing-Press-HTML-Response": "true",
	}
}

func algoliaSearchHeaders() map[string]string {
	return untappd.AlgoliaHeaders()
}

func normalizeBeerSearchHits(data json.RawMessage) (json.RawMessage, error) {
	return untappd.NormalizeBeerHits(data)
}

func normalizeBrewerySearchHits(data json.RawMessage) (json.RawMessage, error) {
	return untappd.NormalizeBreweryHits(data)
}

func beerRequestPath(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "/beer/"
	}
	if strings.HasPrefix(id, "http://") || strings.HasPrefix(id, "https://") {
		if u, err := url.Parse(id); err == nil && u.Host != "" {
			if u.Path != "" {
				return u.Path
			}
		}
	}
	if strings.HasPrefix(id, "/") {
		return id
	}
	if strings.Contains(id, "/") {
		return "/b/" + strings.TrimPrefix(id, "b/")
	}
	return "/beer/" + id
}

func breweryRequestPath(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "/brewery/"
	}
	if strings.HasPrefix(id, "http://") || strings.HasPrefix(id, "https://") {
		if u, err := url.Parse(id); err == nil && u.Path != "" {
			return u.Path
		}
	}
	if strings.HasPrefix(id, "/") {
		return id
	}
	if _, err := strconv.Atoi(id); err == nil {
		return "/brewery/" + id
	}
	return "/" + id
}

func runBeerSearch(ctx context.Context, c *client.Client, query string, limit int) (json.RawMessage, error) {
	if limit <= 0 {
		limit = 10
	}
	body := map[string]any{"query": query, "hitsPerPage": limit}
	data, _, err := c.PostQueryWithParamsAndHeaders(ctx, untappd.AlgoliaQueryURL("beer"), nil, body, untappd.AlgoliaHeaders())
	if err != nil {
		return nil, err
	}
	return untappd.NormalizeBeerHits(data)
}

func runBrewerySearch(ctx context.Context, c *client.Client, query string, limit int) (json.RawMessage, error) {
	if limit <= 0 {
		limit = 10
	}
	body := map[string]any{"query": query, "hitsPerPage": limit}
	data, _, err := c.PostQueryWithParamsAndHeaders(ctx, untappd.AlgoliaQueryURL("brewery"), nil, body, untappd.AlgoliaHeaders())
	if err != nil {
		return nil, err
	}
	return untappd.NormalizeBreweryHits(data)
}

func parseBeerHTML(data json.RawMessage) (json.RawMessage, error) {
	beer, err := untappd.ParseBeerPage(data)
	if err != nil {
		return nil, err
	}
	return untappd.MarshalJSON(beer)
}

func parseBreweryHTML(data json.RawMessage) (json.RawMessage, error) {
	b, err := untappd.ParseBreweryPage(data)
	if err != nil {
		return nil, err
	}
	return untappd.MarshalJSON(b)
}

func parseBreweryBeersHTML(data json.RawMessage) (json.RawMessage, error) {
	items, err := untappd.ParseBreweryBeerList(data)
	if err != nil {
		return nil, err
	}
	return untappd.MarshalJSON(items)
}

func searchLimitFromFlags(cmd *cobra.Command, limit, hitsPerPage int) int {
	switch {
	case cmd.Flags().Changed("limit"):
		return limit
	case cmd.Flags().Changed("hits-per-page"):
		return hitsPerPage
	case limit > 0:
		return limit
	case hitsPerPage > 0:
		return hitsPerPage
	default:
		return 10
	}
}

func searchQueryFrom(cmd *cobra.Command, args []string, flagQuery string) (string, error) {
	if cmd.Flags().Changed("query") || strings.TrimSpace(flagQuery) != "" {
		return strings.TrimSpace(flagQuery), nil
	}
	if len(args) > 0 {
		return strings.TrimSpace(strings.Join(args, " ")), nil
	}
	return "", fmt.Errorf("query is required")
}
