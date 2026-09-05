package cli

// Seed the generated `which` index. Printing Press left whichIndex empty
// because novel_features_built was never synced; the skill still tells
// agents to run `which` first.
func init() {
	seedWhichIndex(untappdWhichIndex)
}

func seedWhichIndex(extra []whichEntry) {
	seen := make(map[string]bool, len(whichIndex)+len(extra))
	for _, e := range whichIndex {
		seen[e.Command] = true
	}
	for _, e := range extra {
		if e.Command == "" || seen[e.Command] {
			continue
		}
		seen[e.Command] = true
		whichIndex = append(whichIndex, e)
	}
}

var untappdWhichIndex = []whichEntry{
	{
		Command:      "search beer",
		Description:  "Search public Untappd beers by name; returns brewery, ABV, rating when published",
		Group:        "beer",
		WhyItMatters: "Preferred agent form for “what is this beer / who brews it / what’s the rating”",
	},
	{
		Command:      "beer get",
		Description:  "Fetch a public beer page by id or slug: style, ABV, IBU, global rating, description",
		Group:        "beer",
		WhyItMatters: "Use after search when you already have a beer id such as 4384886",
	},
	{
		Command:      "lookup",
		Description:  "Look up several tap-list beer names in one polite sequential pass",
		Group:        "beer",
		WhyItMatters: "A menu or flight of names — not a venue geo search",
	},
	{
		Command:      "brewery search",
		Description:  "Search public breweries by name",
		Group:        "brewery",
		WhyItMatters: "Find a brewery slug or id before listing its beers",
	},
	{
		Command:      "brewery beers",
		Description:  "List beers on a public brewery page with ratings when Untappd publishes them",
		Group:        "brewery",
		WhyItMatters: "Brewery catalog, not nearby venues",
	},
	{
		Command:      "nearby",
		Description:  "Top public venues near a hotel, address, or lat/lng ranked by Untappd popularity",
		Group:        "venue",
		WhyItMatters: "Answer “what’s worth drinking near my hotel” — popularity is check-in volume, not a star rating",
	},
	{
		Command:      "venue search",
		Description:  "Search public venues by name, or pass --near to search around a place",
		Group:        "venue",
		WhyItMatters: "Find a venue id/slug, or venues around a place name",
	},
	{
		Command:      "venue get",
		Description:  "Public venue page: name, address, lat/lng; venue-level rating stays null unless published",
		Group:        "venue",
		WhyItMatters: "Confirm a venue after nearby or search; do not invent a venue score",
	},
	{
		Command:      "venue top-beers",
		Description:  "On-menu beers at a public venue, sorted by published global beer rating",
		Group:        "venue",
		WhyItMatters: "What’s on tap / on the menu at a known venue id, with beer ratings when present",
	},
	{
		Command:      "search venue",
		Description:  "Search public venues by name via the same index as venue search",
		Group:        "venue",
		WhyItMatters: "Alias of venue search for agents that start from the search parent",
	},
}
