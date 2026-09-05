package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWhichIndex_HotelQueryRanksNearby(t *testing.T) {
	got := rankWhich(whichIndex, "venues near my hotel", 3)
	if len(got) == 0 {
		t.Fatal("expected which index to match hotel/nearby queries")
	}
	if got[0].Entry.Command != "nearby" {
		t.Fatalf("top match for hotel nearby query: want nearby, got %s (%+v)", got[0].Entry.Command, got)
	}
}

func TestWhichIndex_MenuBeersRanksTopBeers(t *testing.T) {
	got := rankWhich(whichIndex, "on-menu beers at a venue", 3)
	if len(got) == 0 {
		t.Fatal("expected a menu/top-beers match")
	}
	if got[0].Entry.Command != "venue top-beers" {
		t.Fatalf("top match: want venue top-beers, got %s (%+v)", got[0].Entry.Command, got)
	}
}

func TestWhichCommand_HotelQueryJSON(t *testing.T) {
	var stdout bytes.Buffer
	flags := &rootFlags{asJSON: true}
	cmd := newWhichCmd(flags)
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"venues near my hotel"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("which hotel query: %v\nstdout=%s", err, stdout.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("json: %v stdout=%s", err, stdout.String())
	}
	matches, _ := envelope["matches"].([]any)
	if len(matches) == 0 {
		t.Fatalf("empty matches: %s", stdout.String())
	}
	top, _ := matches[0].(map[string]any)
	entry, _ := top["entry"].(map[string]any)
	if cmdName, _ := entry["command"].(string); cmdName != "nearby" {
		t.Fatalf("top command=%q want nearby; stdout=%s", cmdName, stdout.String())
	}
}

func TestWhichCommand_EmptyQueryListsIndex(t *testing.T) {
	var stdout bytes.Buffer
	flags := &rootFlags{asJSON: true}
	cmd := newWhichCmd(flags)
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("which with no query: %v\nstdout=%s", err, stdout.String())
	}
	if !strings.Contains(stdout.String(), `"nearby"`) {
		t.Fatalf("listing mode should include nearby: %s", stdout.String())
	}
}
