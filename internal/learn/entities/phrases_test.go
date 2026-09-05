package entities

import (
	"reflect"
	"testing"
)

func TestExtract_JoinAdjacentPlaceName(t *testing.T) {
	cfg := NewConfig()
	cfg.EnableAdjacentEntityPhrases()
	cfg.RegisterStopwords("near", "nearby", "top", "beers", "beer")

	got := Extract("top beers near Elliot Park Hotel, Minneapolis", cfg)
	want := []string{"Elliot Park Hotel", "Minneapolis"}
	if !reflect.DeepEqual(got.Entities, want) {
		t.Fatalf("entities=%v want %v", got.Entities, want)
	}
}

func TestExtract_JoinDoesNotCrossStopwords(t *testing.T) {
	cfg := NewConfig()
	cfg.EnableAdjacentEntityPhrases()
	got := Extract("Elliot Park Hotel in Minneapolis", cfg)
	want := []string{"Elliot Park Hotel", "Minneapolis"}
	if !reflect.DeepEqual(got.Entities, want) {
		t.Fatalf("entities=%v want %v", got.Entities, want)
	}
}

func TestExtract_DefaultStillSplitsNames(t *testing.T) {
	got := Extract("find Will Smith bio", NewConfig())
	want := []string{"Will", "Smith"}
	if !reflect.DeepEqual(got.Entities, want) {
		t.Fatalf("default extract must not join: %v", got.Entities)
	}
}
