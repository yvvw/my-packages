//go:build with_tools_upgrade

package main

import (
	"fmt"
	"os"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/experimental/tools_upgrade"
	"github.com/spf13/cobra"
)

var upgradeOptions tools_upgrade.Options

var commandToolsUpgrade = &cobra.Command{
	Use:                   "upgrade [-f] [-R <repo>]",
	Short:                 "Upgrade sing-box to latest version",
	DisableFlagsInUseLine: true,
	Run:                   runUpgradeCmd,
}

func init() {
	flags := commandToolsUpgrade.Flags()
	flags.BoolVarP(&upgradeOptions.Force, "force", "f", false, "force upgrade even if version is latest")
	flags.StringVarP(&upgradeOptions.Repo, "repo", "R", "yvvw/my-packages", "GitHub repository to check releases from")

	commandTools.AddCommand(commandToolsUpgrade)
}

func runUpgradeCmd(cmd *cobra.Command, args []string) {
	opts := upgradeOptions
	if opts.CurrentVersion == "" {
		opts.CurrentVersion = C.Version
	}
	if err := tools_upgrade.Upgrade(opts); err != nil {
		fmt.Fprintf(os.Stderr, "upgrade error: %v\n", err)
		os.Exit(1)
	}
}
