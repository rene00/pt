package cli

import (
	"context"
	"os"
	"path"

	"github.com/spf13/cobra"
)

// Execute is the main entry point to the cli
func Execute() {
	cli := &cli{}
	rootCmd := buildRootCmd(cli)
	rootCmd.AddCommand(initCmd(cli))
	rootCmd.AddCommand(copyCmd(cli))
	rootCmd.AddCommand(exifCmd(cli))
	rootCmd.AddCommand(scanCmd(cli))
	rootCmd.AddCommand(cr2DupeCmd(cli))
	if err := rootCmd.ExecuteContext(context.TODO()); err != nil {
		os.Exit(1)
	}
}

func buildRootCmd(cli *cli) *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "pt",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return cli.setup(cmd.Context())
		},
	}
	rootCmd.PersistentFlags().BoolVar(&cli.debug, "debug", false, "Enable debug")
	rootCmd.PersistentFlags().StringVar(&cli.configFile, "config-file", path.Join(os.Getenv("HOME"), ".config", "pt", "config.json"), "Config file path")
	return rootCmd
}
