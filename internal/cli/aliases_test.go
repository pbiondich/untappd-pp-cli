package cli

import (
	"reflect"
	"testing"
)

func TestRewriteUserFacingArgs(t *testing.T) {
	cases := []struct {
		in, want []string
	}{
		{
			[]string{"untappd-pp-cli", "beer", "4384886", "--agent"},
			[]string{"untappd-pp-cli", "beer", "get", "4384886", "--agent"},
		},
		{
			[]string{"untappd-pp-cli", "--agent", "beer", "4384886"},
			[]string{"untappd-pp-cli", "--agent", "beer", "get", "4384886"},
		},
		{
			[]string{"untappd-pp-cli", "beer", "get", "4384886"},
			[]string{"untappd-pp-cli", "beer", "get", "4384886"},
		},
		{
			[]string{"untappd-pp-cli", "beer", "search", "Cool Bay Hop Butcher", "--agent"},
			[]string{"untappd-pp-cli", "beer", "search", "--query", "Cool Bay Hop Butcher", "--agent"},
		},
		{
			[]string{"untappd-pp-cli", "brewery", "HopButcher", "beers", "--agent"},
			[]string{"untappd-pp-cli", "brewery", "beers", "HopButcher", "--agent"},
		},
		{
			[]string{"untappd-pp-cli", "brewery", "23570"},
			[]string{"untappd-pp-cli", "brewery", "get", "23570"},
		},
		{
			[]string{"untappd-pp-cli", "brewery", "search", "Hop Butcher"},
			[]string{"untappd-pp-cli", "brewery", "search", "--query", "Hop Butcher"},
		},
		{
			[]string{"untappd-pp-cli", "doctor"},
			[]string{"untappd-pp-cli", "doctor"},
		},
		{
			[]string{"untappd-pp-cli", "venue", "8255451", "--agent"},
			[]string{"untappd-pp-cli", "venue", "get", "8255451", "--agent"},
		},
		{
			[]string{"untappd-pp-cli", "venue", "2714", "top-beers", "--agent"},
			[]string{"untappd-pp-cli", "venue", "top-beers", "2714", "--agent"},
		},
		{
			[]string{"untappd-pp-cli", "venue", "search", "Elliot Park"},
			[]string{"untappd-pp-cli", "venue", "search", "--query", "Elliot Park"},
		},
	}
	for _, tc := range cases {
		got := RewriteUserFacingArgs(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("RewriteUserFacingArgs(%q)\n got %q\nwant %q", tc.in, got, tc.want)
		}
	}
}
