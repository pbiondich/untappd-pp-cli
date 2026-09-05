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

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newVenueCmd(flags))
		addNovelCommandIfAbsent(root, newNearbyCmd(flags))
	})
}

func newVenueCmd(flags *rootFlags) *cobra.Command {
	// pp:data-source live
	cmd := &cobra.Command{
		Use:   "venue",
		Short: "Public Untappd venues — search, nearby, venue pages, and on-menu beers.",
		Example: `  untappd-pp-cli venue search "Elliot Park" --agent
  untappd-pp-cli venue search --near "Elliot Park Hotel, Minneapolis" --agent
  untappd-pp-cli venue 8255451 --agent
  untappd-pp-cli venue 2714 top-beers --agent
  untappd-pp-cli venue near --near "Elliot Park Hotel, Minneapolis" --agent`,
		Annotations: map[string]string{"mcp:read-only": "true", "pp:parent-group": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newVenueSearchCmd(flags))
	cmd.AddCommand(newVenueGetCmd(flags))
	cmd.AddCommand(newVenueTopBeersCmd(flags))
	cmd.AddCommand(newVenueNearCmd(flags))
	return cmd
}

func newVenueSearchCmd(flags *rootFlags) *cobra.Command {
	var query, near string
	var limit int
	var beerOnly bool
	cmd := &cobra.Command{
		Use:     "search [query]",
		Short:   "Search public venues by name, or venues near a place.",
		Example: "  untappd-pp-cli venue search \"Elliot Park\" --agent\n  untappd-pp-cli venue search --near \"Elliot Park Hotel, Minneapolis\" --agent",
		Annotations: map[string]string{
			"pp:endpoint": "venue.search", "mcp:read-only": "true", "pp:requires-input": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			q := strings.TrimSpace(strings.Join(args, " "))
			if cmd.Flags().Changed("query") {
				q = strings.TrimSpace(query)
			}
			if q == "" && near == "" {
				return usageErr(fmt.Errorf("venue search requires a query or --near"))
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			if flags.dryRun {
				return printOutputWithFlagsMeta(cmd.OutOrStdout(), []byte(`{"dry_run":true}`), flags, map[string]any{"source": "dry-run"}, venueCompactFields())
			}
			if near != "" {
				result, err := runNearby(cmd.Context(), c, nearbyOpts{
					Near: near, Query: q, Limit: limit, RadiusMi: 2, Sort: "popularity", BeerOnly: beerOnly,
				})
				if err != nil {
					return classifyAPIError(cmd.OutOrStdout(), err, flags)
				}
				return printNearby(cmd, flags, result)
			}
			data, err := runVenueSearch(cmd.Context(), c, q, limit)
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			return printSearchResults(cmd, flags, data)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Venue name query.")
	cmd.Flags().StringVar(&near, "near", "", "Place name or address to search around.")
	cmd.Flags().IntVar(&limit, "limit", 10, "Max matches (polite default 10).")
	cmd.Flags().BoolVar(&beerOnly, "beer-only", false, "When used with --near, keep venues Untappd marks as having beer.")
	return cmd
}

func newVenueGetCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id-or-slug>",
		Short:   "Fetch a public venue page (name, address, lat/lng). Venue-level rating stays null when Untappd does not publish one.",
		Example: "  untappd-pp-cli venue 8255451 --agent\n  untappd-pp-cli venue get 2714 --agent",
		Annotations: map[string]string{
			"pp:endpoint": "venue.get", "mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
				return usageErr(fmt.Errorf("venue id or slug is required"))
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			if flags.dryRun {
				return printOutputWithFlagsMeta(cmd.OutOrStdout(), []byte(`{"dry_run":true}`), flags, map[string]any{"source": "dry-run"}, venueCompactFields())
			}
			data, err := runVenueGet(cmd.Context(), c, args[0])
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			return printOutputWithFlagsMeta(cmd.OutOrStdout(), data, flags, map[string]any{"source": "live"}, venueCompactFields())
		},
	}
	return cmd
}

func newVenueTopBeersCmd(flags *rootFlags) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:     "top-beers <id-or-slug>",
		Short:   "List beers on a public venue menu, sorted by published global rating when present.",
		Example: "  untappd-pp-cli venue 2714 top-beers --agent\n  untappd-pp-cli venue top-beers 2714 --agent",
		Annotations: map[string]string{
			"pp:endpoint": "venue.top_beers", "mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
				return usageErr(fmt.Errorf("venue id or slug is required"))
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			if flags.dryRun {
				return printOutputWithFlagsMeta(cmd.OutOrStdout(), []byte(`{"dry_run":true}`), flags, map[string]any{"source": "dry-run"}, map[string]bool{"name": true, "rating": true})
			}
			data, err := c.GetWithHeaders(cmd.Context(), venueRequestPath(args[0]), nil, htmlHeaders())
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			beers, err := untappd.ParseVenueMenuBeers(data)
			if err != nil {
				return err
			}
			if limit > 0 && len(beers) > limit {
				beers = beers[:limit]
			}
			out, err := untappd.MarshalJSON(beers)
			if err != nil {
				return err
			}
			return printSearchResults(cmd, flags, out)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 15, "Max menu beers to return.")
	return cmd
}

func newVenueNearCmd(flags *rootFlags) *cobra.Command {
	return newNearbyCommand(flags, "near")
}

func newNearbyCmd(flags *rootFlags) *cobra.Command {
	// pp:data-source live
	return newNearbyCommand(flags, "nearby")
}

type nearbyOpts struct {
	Near     string
	Query    string
	Lat, Lng float64
	HasCoord bool
	RadiusMi float64
	Limit    int
	Sort     string
	BeerOnly bool
}

func newNearbyCommand(flags *rootFlags, use string) *cobra.Command {
	var near string
	var lat, lng, radius float64
	var limit int
	var sortBy string
	var beerOnly bool
	cmd := &cobra.Command{
		Use:   use,
		Short: "Top public venues near a place name or lat/lng (Untappd popularity, not an invented rating).",
		Example: `  untappd-pp-cli nearby --near "Elliot Park Hotel, Minneapolis" --agent
  untappd-pp-cli nearby --lat 44.972332 --lng -93.266396 --radius-mi 2 --agent
  untappd-pp-cli venue near --near "Elliot Park Hotel, Minneapolis" --sort recent --agent`,
		Annotations: map[string]string{
			"pp:endpoint": "venue.nearby", "mcp:read-only": "true", "pp:requires-input": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if near == "" && len(args) > 0 {
				near = strings.TrimSpace(strings.Join(args, " "))
			}
			hasCoord := cmd.Flags().Changed("lat") && cmd.Flags().Changed("lng")
			if near == "" && !hasCoord {
				return usageErr(fmt.Errorf("%s requires --near <place> or --lat/--lng", use))
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			if flags.dryRun {
				return printOutputWithFlagsMeta(cmd.OutOrStdout(), []byte(`{"dry_run":true}`), flags, map[string]any{"source": "dry-run"}, venueCompactFields())
			}
			result, err := runNearby(cmd.Context(), c, nearbyOpts{
				Near: near, Lat: lat, Lng: lng, HasCoord: hasCoord,
				RadiusMi: radius, Limit: limit, Sort: sortBy, BeerOnly: beerOnly,
			})
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			return printNearby(cmd, flags, result)
		},
	}
	cmd.Flags().StringVar(&near, "near", "", "Place name or address (geocoded via Untappd venue index, then OSM Photon).")
	cmd.Flags().Float64Var(&lat, "lat", 0, "Latitude (use with --lng).")
	cmd.Flags().Float64Var(&lng, "lng", 0, "Longitude (use with --lat).")
	cmd.Flags().Float64Var(&radius, "radius-mi", 2, "Search radius in miles.")
	cmd.Flags().IntVar(&limit, "limit", 10, "Max venues to return.")
	cmd.Flags().StringVar(&sortBy, "sort", "popularity", "Sort: popularity (default), recent, or distance.")
	cmd.Flags().BoolVar(&beerOnly, "beer-only", true, "Keep venues Untappd marks as having beer (default true).")
	return cmd
}

func printNearby(cmd *cobra.Command, flags *rootFlags, result *untappd.NearbyResult) error {
	out, err := untappd.MarshalJSON(result)
	if err != nil {
		return err
	}
	if wantsHumanTable(cmd.OutOrStdout(), flags) {
		items := make([]map[string]any, 0, len(result.Venues))
		raw, _ := json.Marshal(result.Venues)
		_ = json.Unmarshal(raw, &items)
		if len(items) > 0 {
			return printAutoTable(cmd.OutOrStdout(), items)
		}
	}
	return printOutputWithFlagsMeta(cmd.OutOrStdout(), out, flags, map[string]any{"source": "live"}, venueCompactFields())
}

func venueCompactFields() map[string]bool {
	return map[string]bool{
		"id": true, "name": true, "category": true, "address": true, "city": true,
		"lat": true, "lng": true, "distance_mi": true, "popularity": true,
		"has_beer": true, "url": true, "rating": true, "origin": true, "venues": true,
	}
}

func venueRequestPath(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "/venue/"
	}
	if strings.HasPrefix(id, "http://") || strings.HasPrefix(id, "https://") {
		if u, err := url.Parse(id); err == nil && u.Path != "" {
			return u.Path
		}
	}
	if strings.HasPrefix(id, "/") {
		return id
	}
	if strings.Contains(id, "/") {
		return "/v/" + strings.TrimPrefix(id, "v/")
	}
	if _, err := strconv.Atoi(id); err == nil {
		return "/venue/" + id
	}
	return "/v/" + id
}

func runVenueSearch(ctx context.Context, c *client.Client, query string, limit int) (json.RawMessage, error) {
	if limit <= 0 {
		limit = 10
	}
	body := map[string]any{"query": query, "hitsPerPage": limit}
	data, _, err := c.PostQueryWithParamsAndHeaders(ctx, untappd.AlgoliaQueryURL("venue"), nil, body, untappd.AlgoliaHeaders())
	if err != nil {
		return nil, err
	}
	return untappd.NormalizeVenueHitsJSON(data)
}

func runVenueGet(ctx context.Context, c *client.Client, id string) (json.RawMessage, error) {
	data, err := c.GetWithHeaders(ctx, venueRequestPath(id), nil, htmlHeaders())
	if err != nil {
		return nil, err
	}
	v, err := untappd.ParseVenuePage(data)
	if err != nil {
		return nil, err
	}
	return untappd.MarshalJSON(v)
}

func runNearby(ctx context.Context, c *client.Client, opt nearbyOpts) (*untappd.NearbyResult, error) {
	if opt.RadiusMi <= 0 {
		opt.RadiusMi = 2
	}
	if opt.Limit <= 0 {
		opt.Limit = 10
	}
	if opt.Sort == "" {
		opt.Sort = "popularity"
	}
	origin, err := resolvePlace(ctx, c, opt)
	if err != nil {
		return nil, err
	}
	fetch := opt.Limit * 3
	if fetch < 25 {
		fetch = 25
	}
	body := untappd.AlgoliaAroundBody(opt.Query, origin.Lat, origin.Lng, opt.RadiusMi, opt.BeerOnly, fetch)
	data, _, err := c.PostQueryWithParamsAndHeaders(ctx, untappd.AlgoliaQueryURL("venue"), nil, body, untappd.AlgoliaHeaders())
	if err != nil {
		return nil, err
	}
	venues, err := untappd.NormalizeVenueHits(data)
	if err != nil {
		return nil, err
	}
	untappd.AttachDistances(venues, origin.Lat, origin.Lng)
	untappd.SortVenues(venues, opt.Sort)
	if len(venues) > opt.Limit {
		venues = venues[:opt.Limit]
	}
	return &untappd.NearbyResult{
		Origin:   *origin,
		RadiusMi: opt.RadiusMi,
		Sort:     opt.Sort,
		BeerOnly: opt.BeerOnly,
		Venues:   venues,
	}, nil
}

func resolvePlace(ctx context.Context, c *client.Client, opt nearbyOpts) (*untappd.Place, error) {
	if opt.HasCoord {
		return &untappd.Place{Query: opt.Near, Lat: opt.Lat, Lng: opt.Lng, Source: "latlng"}, nil
	}
	q := strings.TrimSpace(opt.Near)
	if q == "" {
		return nil, fmt.Errorf("place is required")
	}
	data, _, err := c.PostQueryWithParamsAndHeaders(ctx, untappd.AlgoliaQueryURL("venue"), nil, map[string]any{
		"query": q, "hitsPerPage": 5,
	}, untappd.AlgoliaHeaders())
	if err == nil {
		venues, nerr := untappd.NormalizeVenueHits(data)
		if nerr == nil {
			if v := pickAnchorVenue(venues, q); v != nil {
				if p := untappd.VenueFromAnchor(*v, q); p != nil {
					return p, nil
				}
			}
		}
	}
	photon, err := c.Get(ctx, untappd.PhotonURL(q), nil)
	if err != nil {
		return nil, fmt.Errorf("geocode %q: pass --lat/--lng (Untappd venue miss and Photon failed: %w)", q, err)
	}
	return untappd.ParsePhoton(photon, q)
}

func pickAnchorVenue(venues []untappd.Venue, query string) *untappd.Venue {
	if len(venues) == 0 {
		return nil
	}
	ql := strings.ToLower(query)
	for i := range venues {
		name := strings.ToLower(venues[i].Name)
		if strings.Contains(name, ql) || strings.Contains(ql, name) || strings.Contains(ql, strings.ToLower(venues[i].City)) {
			if venues[i].Lat != nil {
				return &venues[i]
			}
		}
	}
	if venues[0].Lat != nil {
		return &venues[0]
	}
	return nil
}
