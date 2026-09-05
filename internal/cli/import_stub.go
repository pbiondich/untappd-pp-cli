package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const importUnsupportedMsg = "not supported on this public-page CLI"

func init() {
	registerNovelCommand(func(root *cobra.Command, _ *rootFlags) {
		disableGeneratedImport(root)
	})
}

// disableGeneratedImport hides the Printing Press import verb and replaces
// its create/upsert RunE. This CLI has no write API.
func disableGeneratedImport(root *cobra.Command) {
	if root == nil {
		return
	}
	cmd, _, err := root.Find([]string{"import"})
	if err != nil || cmd == nil || cmd.Name() != "import" {
		return
	}
	cmd.Hidden = true
	cmd.Short = "Not supported on this public-page CLI."
	cmd.Long = "import is a generated create/upsert path. untappd-pp-cli only reads public Untappd pages and is not a write API.\n\n" +
		"Use search, lookup, nearby, or venue instead."
	cmd.Example = ""
	cmd.Use = "import"
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations["mcp:hidden"] = "true"
	cmd.RunE = func(c *cobra.Command, _ []string) error {
		return usageErr(fmt.Errorf("%s", importUnsupportedMsg))
	}
}
