package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestImportHiddenFromHelp(t *testing.T) {
	var stdout bytes.Buffer
	root := newRootCmd(&rootFlags{})
	root.SetOut(&stdout)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if strings.Contains(out, "\n  import ") || strings.Contains(out, "Import data from JSONL") {
		t.Fatalf("import must not appear in --help:\n%s", out)
	}
}

func TestImportStubErrors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	flags := &rootFlags{asJSON: true, noInput: true}
	root := newRootCmd(flags)
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"import", "beer", "--input", "data.jsonl"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected import to fail")
	}
	if !strings.Contains(err.Error(), importUnsupportedMsg) {
		t.Fatalf("error=%v", err)
	}
	if got := ExitCode(err); got != 2 {
		t.Fatalf("ExitCode=%d want 2", got)
	}
	if strings.Contains(stdout.String(), `"succeeded"`) {
		t.Fatalf("must not run the generated upsert path: %s", stdout.String())
	}
}

func TestAgentContextOmitsImport(t *testing.T) {
	var stdout bytes.Buffer
	root := newRootCmd(&rootFlags{})
	root.SetOut(&stdout)
	root.SetArgs([]string{"agent-context", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), `"name": "import"`) {
		t.Fatalf("agent-context still advertises import: %s", stdout.String())
	}
}
