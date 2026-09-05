package cli

import (
	"reflect"
	"testing"

	"github.com/pbiondich/untappd-pp-cli/internal/learn"
)

func TestLearnConfig_KeepsHotelVenuePhrase(t *testing.T) {
	cfg := newLearnConfig()
	n := learn.Normalize("top beers near Elliot Park Hotel, Minneapolis", cfg)
	want := []string{"Elliot Park Hotel", "Minneapolis"}
	if !reflect.DeepEqual(n.Entities, want) {
		t.Fatalf("entities=%v want %v (non-entity=%q)", n.Entities, want, n.NonEntityNormalized)
	}
	if n.NonEntityNormalized != "" {
		t.Fatalf("wrapper words should be stopwords, got %q", n.NonEntityNormalized)
	}
}

func TestLearnConfig_DoesNotSplitOnSportsAliases(t *testing.T) {
	cfg := newLearnConfig()
	n := learn.Normalize("Town Hall Brewery nearby", cfg)
	if len(n.Entities) != 1 || n.Entities[0] != "Town Hall Brewery" {
		t.Fatalf("entities=%v want [Town Hall Brewery]", n.Entities)
	}
}

func TestLearnIdentityFields_IncludesVenue(t *testing.T) {
	fields := learnResourceTypeFields()
	if _, ok := fields["venue"]; !ok {
		t.Fatal("venue identity fields missing")
	}
}
