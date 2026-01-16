package unsubscribe

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	bemfa "yvvw/my-packages/bemfa/internal"
)

var Cmd = &cobra.Command{
	Use:     "unsubscribe",
	Short:   "unsubscribe bemfa topics",
	Example: "bemfa ubsubscibe [--server tcp://bemfa.com:9501] --token [token] --topics [topic1,topic2] [--times 5]",
	Run:     run,
}

func init() {
	Cmd.Flags().Uint("times", 1, "unsubscribe times")
	Cmd.Flags().StringSlice("topics", nil, "bemfa topics")
}

func run(cmd *cobra.Command, _ []string) {
	server, err := cmd.Flags().GetString("server")
	if err != nil {
		log.Fatal(err)
	}

	token, err := cmd.Flags().GetString("token")
	if err != nil {
		log.Fatal(err)
	}

	topics, err := cmd.Flags().GetStringSlice("topics")
	if err != nil {
		log.Fatal(err)
	}
	if len(topics) == 0 {
		log.Fatal("topic length is 0")
	}

	times, err := cmd.Flags().GetUint("times")
	if err != nil {
		log.Fatal(err)
	}
	if times == 0 {
		log.Fatal("times must be greater then 0")
	}

	client := bemfa.NewSimple(server, token)

	err = client.Connect()
	if err != nil {
		log.Fatalf("bemfa connect error %s", err.Error())
	}

	for i := 0; i < int(times); i++ {
		err = client.Unsubscribe(topics...)
		if err != nil {
			log.Fatalf("bemfa unsubscribe %v error %s", topics, err.Error())
		}
	}
}
