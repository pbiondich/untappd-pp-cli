package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newSearchCmd(flags))
		addNovelCommandIfAbsent(root, newLookupCmd(flags))
	})
}

func newSearchCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search public Untappd beers, breweries, or venues.",
		Example: `  untappd-pp-cli search beer "Hop Butcher Put On the Glasses" --agent
  untappd-pp-cli search brewery "Hop Butcher" --agent
  untappd-pp-cli search venue "Elliot Park" --agent`,
		Annotations: map[string]string{"mcp:read-only": "true", "pp:parent-group": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newSearchBeerCmd(flags))
	cmd.AddCommand(newSearchBreweryCmd(flags))
	cmd.AddCommand(newSearchVenueCmd(flags))
	return cmd
}

func newSearchBeerCmd(flags *rootFlags) *cobra.Command {
	var query string
	var limit int
	cmd := &cobra.Command{
		Use:     "beer [query]",
		Short:   "Search public beers (Algolia index published on untappd.com/search).",
		Example: "  untappd-pp-cli search beer \"Hop Butcher Put On the Glasses\" --agent",
		Annotations: map[string]string{
			"pp:endpoint": "beer.search", "pp:method": "POST",
			"mcp:read-only": "true", "pp:requires-input": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			q, err := searchQueryFrom(cmd, args, query)
			if err != nil {
				if flags.asJSON {
					_ = printJSONFiltered(cmd.OutOrStdout(), map[string]any{
						"error": "requires input",
						"usage": `untappd-pp-cli search beer "<query>" --agent`,
					}, flags)
				}
				return usageErr(err)
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			if flags.dryRun {
				return printOutputWithFlagsMeta(cmd.OutOrStdout(), []byte(`{"dry_run":true}`), flags, map[string]any{"source": "dry-run"}, map[string]bool{"name": true, "brewery": true, "rating": true})
			}
			data, err := runBeerSearch(cmd.Context(), c, q, searchLimitFromFlags(cmd, limit, limit))
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			return printSearchResults(cmd, flags, data)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Beer search query.")
	cmd.Flags().IntVar(&limit, "limit", 10, "Max matches to return (polite default 10).")
	return cmd
}

func newSearchBreweryCmd(flags *rootFlags) *cobra.Command {
	var query string
	var limit int
	cmd := &cobra.Command{
		Use:     "brewery [query]",
		Short:   "Search public breweries (Algolia index published on untappd.com/search).",
		Example: "  untappd-pp-cli search brewery \"Hop Butcher\" --agent",
		Annotations: map[string]string{
			"pp:endpoint": "brewery.search", "pp:method": "POST",
			"mcp:read-only": "true", "pp:requires-input": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			q, err := searchQueryFrom(cmd, args, query)
			if err != nil {
				if flags.asJSON {
					_ = printJSONFiltered(cmd.OutOrStdout(), map[string]any{
						"error": "requires input",
						"usage": `untappd-pp-cli search brewery "<query>" --agent`,
					}, flags)
				}
				return usageErr(err)
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			if flags.dryRun {
				return printOutputWithFlagsMeta(cmd.OutOrStdout(), []byte(`{"dry_run":true}`), flags, map[string]any{"source": "dry-run"}, map[string]bool{"name": true, "slug": true})
			}
			data, err := runBrewerySearch(cmd.Context(), c, q, searchLimitFromFlags(cmd, limit, limit))
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			return printSearchResults(cmd, flags, data)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Brewery search query.")
	cmd.Flags().IntVar(&limit, "limit", 10, "Max matches to return (polite default 10).")
	return cmd
}

func newSearchVenueCmd(flags *rootFlags) *cobra.Command {
	var query string
	var limit int
	cmd := &cobra.Command{
		Use:     "venue [query]",
		Short:   "Search public venues (Algolia venue index published on untappd.com/search).",
		Example: "  untappd-pp-cli search venue \"Elliot Park\" --agent",
		Annotations: map[string]string{
			"pp:endpoint": "venue.search", "pp:method": "POST",
			"mcp:read-only": "true", "pp:requires-input": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			q, err := searchQueryFrom(cmd, args, query)
			if err != nil {
				return usageErr(err)
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			if flags.dryRun {
				return printOutputWithFlagsMeta(cmd.OutOrStdout(), []byte(`{"dry_run":true}`), flags, map[string]any{"source": "dry-run"}, venueCompactFields())
			}
			data, err := runVenueSearch(cmd.Context(), c, q, searchLimitFromFlags(cmd, limit, limit))
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			return printSearchResults(cmd, flags, data)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Venue search query.")
	cmd.Flags().IntVar(&limit, "limit", 10, "Max matches to return (polite default 10).")
	return cmd
}

func printSearchResults(cmd *cobra.Command, flags *rootFlags, data []byte) error {
	if wantsHumanTable(cmd.OutOrStdout(), flags) {
		var items []map[string]any
		if err := json.Unmarshal(data, &items); err == nil && len(items) > 0 {
			if err := printAutoTable(cmd.OutOrStdout(), items); err != nil {
				return err
			}
			return nil
		}
	}
	return printOutputWithFlagsMeta(cmd.OutOrStdout(), data, flags, map[string]any{"source": "live"}, map[string]bool{"id": true, "name": true, "brewery": true, "style": true, "abv": true, "ibu": true, "rating": true, "rating_count": true, "url": true})
}

func newLookupCmd(flags *rootFlags) *cobra.Command {
	var brewery string
	var limit int
	cmd := &cobra.Command{
		Use:   "lookup [beer names...]",
		Short: "Look up several public beers in one polite pass (no parallel stampede).",
		Example: `  untappd-pp-cli lookup "Put On the Glasses" "Cool Bay" "Lord Octopus" --brewery "Hop Butcher" --agent
  untappd-pp-cli lookup "Saturation Above Replacement Hop Butcher" --agent`,
		Annotations: map[string]string{"mcp:read-only": "true", "pp:requires-input": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return usageErr(fmt.Errorf("lookup requires at least one beer name"))
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			if flags.dryRun {
				return printOutputWithFlagsMeta(cmd.OutOrStdout(), []byte(`{"dry_run":true}`), flags, map[string]any{"source": "dry-run"}, map[string]bool{"query": true})
			}
			if limit <= 0 {
				limit = 5
			}
			type row struct {
				Query   string `json:"query"`
				Matches any    `json:"matches"`
			}
			var rows []row
			for _, name := range args {
				q := name
				if brewery != "" && !containsFold(name, brewery) {
					q = name + " " + brewery
				}
				data, err := runBeerSearch(cmd.Context(), c, q, limit)
				if err != nil {
					return classifyAPIError(cmd.OutOrStdout(), err, flags)
				}
				var matches any
				if err := json.Unmarshal(data, &matches); err != nil {
					return err
				}
				rows = append(rows, row{Query: q, Matches: matches})
			}
			out, err := json.Marshal(rows)
			if err != nil {
				return err
			}
			return printOutputWithFlagsMeta(cmd.OutOrStdout(), out, flags, map[string]any{"source": "live"}, map[string]bool{"query": true, "matches": true})
		},
	}
	cmd.Flags().StringVar(&brewery, "brewery", "", "Restrict lookup queries to this brewery name.")
	cmd.Flags().IntVar(&limit, "limit", 5, "Hits per beer name.")
	return cmd
}

func containsFold(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		(haystack == needle || stringContainsFold(haystack, needle))
}

func stringContainsFold(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (indexFold(s, sub) >= 0))
}

func indexFold(s, sub string) int {
	sl, subl := []rune(s), []rune(sub)
	if len(subl) == 0 {
		return 0
	}
	for i := 0; i+len(subl) <= len(sl); i++ {
		ok := true
		for j := 0; j < len(subl); j++ {
			if foldRune(sl[i+j]) != foldRune(subl[j]) {
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

func foldRune(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}
