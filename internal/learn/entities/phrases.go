package entities

import "strings"

// EnableAdjacentEntityPhrases joins consecutive capitalized entity tokens
// into one place or beer name ("Elliot Park Hotel" not Elliot + Park + Hotel).
// Default Extract behavior is unchanged until a CLI opts in.
func (c *Config) EnableAdjacentEntityPhrases() {
	if c != nil {
		c.joinAdjacentEntities = true
	}
}

func appendJoinedEntity(ents []string, tok string, join bool) []string {
	if join && len(ents) > 0 {
		ents[len(ents)-1] = ents[len(ents)-1] + " " + tok
		return ents
	}
	return append(ents, tok)
}

func hasPhraseBreak(raw string) bool {
	return strings.ContainsAny(raw, ",")
}
