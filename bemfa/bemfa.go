package main

import (
	"os"

	"github.com/spf13/cobra"

	"yvvw/my-packages/bemfa/cmd/subscribe"
	"yvvw/my-packages/bemfa/cmd/unsubscribe"
)

var cmd = &cobra.Command{
	Use: "bemfa",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

func init() {
	cmd.AddCommand(subscribe.Cmd)
	cmd.AddCommand(unsubscribe.Cmd)

	cmd.PersistentFlags().StringP("server", "S", "tcp://bemfa.com:9501", "bemfa MQTT server address")
	cmd.PersistentFlags().StringP("token", "T", "", "bemfa MQTT access token")

}

func main() {
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
