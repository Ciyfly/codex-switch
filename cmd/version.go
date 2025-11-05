package cmd

import (
	"fmt"

	"github.com/codex-switch/codex-switch/internal/version"

	"github.com/spf13/cobra"
)

func init() {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "显示当前版本信息",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "Codex Key Manager %s\n", version.Version)
		},
	}

	RootCommand().AddCommand(versionCmd)
}
