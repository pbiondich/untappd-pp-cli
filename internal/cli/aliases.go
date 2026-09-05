package cli

import (
	"strings"
)

var flagsWithValues = map[string]bool{
	"--select":        true,
	"--config":        true,
	"--home":          true,
	"--profile":       true,
	"--client":        true,
	"--timeout":       true,
	"--rate-limit":    true,
	"--query":         true,
	"--limit":         true,
	"--hits-per-page": true,
	"--fail-on":       true,
	"--deliver":       true,
	"--data-source":   true,
	"--receipt-file":  true,
	"--audit-dir":     true,
	"--near":          true,
	"--lat":           true,
	"--lng":           true,
	"--radius-mi":     true,
	"--sort":          true,
}

var beerSubcommands = map[string]bool{
	"get": true, "search": true, "help": true,
}

var brewerySubcommands = map[string]bool{
	"get": true, "search": true, "beers": true, "help": true,
}

var venueSubcommands = map[string]bool{
	"get": true, "search": true, "near": true, "nearby": true, "top-beers": true, "help": true,
}

// RewriteUserFacingArgs maps the documented agent verbs onto generated
// Printing Press commands:
//
//	beer 4384886              -> beer get 4384886
//	brewery 23570             -> brewery get 23570
//	brewery HopButcher beers  -> brewery beers HopButcher
//	brewery search Hop Butcher -> brewery search --query Hop Butcher
//	beer search Cool Bay      -> beer search --query Cool Bay
func RewriteUserFacingArgs(argv []string) []string {
	if len(argv) < 2 {
		return argv
	}
	prog := argv[0]
	rest := append([]string{}, argv[1:]...)
	pos := positionalIndexes(rest)
	if len(pos) == 0 {
		return argv
	}
	cmd := rest[pos[0]]
	switch cmd {
	case "beer":
		if len(pos) >= 2 && !beerSubcommands[rest[pos[1]]] && !strings.HasPrefix(rest[pos[1]], "-") {
			out := insertAt(rest, pos[1], "get")
			out = ensureQueryFlag(out, "beer", "search")
			return append([]string{prog}, out...)
		}
		out := ensureQueryFlag(rest, "beer", "search")
		return append([]string{prog}, out...)
	case "brewery":
		if len(pos) >= 3 && !brewerySubcommands[rest[pos[1]]] && rest[pos[2]] == "beers" {
			id := rest[pos[1]]
			without := removeAt(rest, pos[2])
			// pos[1] still points at id after removing later beers token
			without[pos[1]] = "beers"
			without = insertAt(without, pos[1]+1, id)
			return append([]string{prog}, without...)
		}
		if len(pos) >= 2 && !brewerySubcommands[rest[pos[1]]] && !strings.HasPrefix(rest[pos[1]], "-") {
			out := insertAt(rest, pos[1], "get")
			return append([]string{prog}, out...)
		}
		out := ensureQueryFlag(rest, "brewery", "search")
		return append([]string{prog}, out...)
	case "venue":
		if len(pos) >= 3 && !venueSubcommands[rest[pos[1]]] && rest[pos[2]] == "top-beers" {
			id := rest[pos[1]]
			without := removeAt(rest, pos[2])
			without[pos[1]] = "top-beers"
			without = insertAt(without, pos[1]+1, id)
			return append([]string{prog}, without...)
		}
		if len(pos) >= 2 && !venueSubcommands[rest[pos[1]]] && !strings.HasPrefix(rest[pos[1]], "-") {
			out := insertAt(rest, pos[1], "get")
			return append([]string{prog}, out...)
		}
		out := ensureQueryFlag(rest, "venue", "search")
		return append([]string{prog}, out...)
	default:
		return argv
	}
}

func ensureQueryFlag(args []string, parent, child string) []string {
	pos := positionalIndexes(args)
	if len(pos) < 2 || args[pos[0]] != parent || args[pos[1]] != child {
		return args
	}
	for _, a := range args {
		if a == "--query" || strings.HasPrefix(a, "--query=") {
			return args
		}
	}
	if len(pos) < 3 {
		return args
	}
	// Remaining positionals become --query <joined>
	start := pos[2]
	var qparts []string
	var drop []int
	for _, idx := range pos[2:] {
		if idx < start {
			continue
		}
		qparts = append(qparts, args[idx])
		drop = append(drop, idx)
	}
	if len(qparts) == 0 {
		return args
	}
	out := append([]string{}, args...)
	for i := len(drop) - 1; i >= 0; i-- {
		out = removeAt(out, drop[i])
	}
	insertAt := start
	if insertAt > len(out) {
		insertAt = len(out)
	}
	tail := append([]string{"--query", strings.Join(qparts, " ")}, out[insertAt:]...)
	return append(out[:insertAt], tail...)
}

func positionalIndexes(args []string) []int {
	var idxs []int
	skipNext := false
	for i := 0; i < len(args); i++ {
		if skipNext {
			skipNext = false
			continue
		}
		a := args[i]
		if a == "--" {
			for j := i + 1; j < len(args); j++ {
				idxs = append(idxs, j)
			}
			break
		}
		if strings.HasPrefix(a, "-") {
			name, _, ok := strings.Cut(a, "=")
			if !ok && flagsWithValues[name] && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				skipNext = true
			}
			continue
		}
		idxs = append(idxs, i)
	}
	return idxs
}

func insertAt(args []string, idx int, val string) []string {
	out := make([]string, 0, len(args)+1)
	out = append(out, args[:idx]...)
	out = append(out, val)
	out = append(out, args[idx:]...)
	return out
}

func removeAt(args []string, idx int) []string {
	out := make([]string, 0, len(args)-1)
	out = append(out, args[:idx]...)
	out = append(out, args[idx+1:]...)
	return out
}

func argsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

