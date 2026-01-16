package subscribe

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"yvvw/my-packages/bemfa/cmd/subscribe/runner"
	bemfa "yvvw/my-packages/bemfa/internal"
	"yvvw/my-packages/bemfa/internal/utils"
)

var Cmd = &cobra.Command{
	Use:     "subscribe",
	Short:   "subscribe bemfa and exec commands",
	Example: "bemfa subscibe [--server tcp://bemfa.com:9501] --token [token] --action 'TOPIC1 WOL xx:xx:xx:xx:xx iface' --action 'TOPIC2 WIN_SHUTDOWN'",
	Run:     run,
}

func init() {
	Cmd.Flags().StringP("server", "S", "tcp://bemfa.com:9501", "bemfa MQTT server address")
	Cmd.Flags().StringP("token", "T", "", "bemfa MQTT access token")
	Cmd.Flags().StringSliceP("action", "A", nil, "bemfa actions")
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

	actions, err := cmd.Flags().GetStringSlice("action")
	if err != nil {
		log.Fatal(err)
	}
	if len(actions) == 0 {
		log.Fatal("action can't be empty")
	}

	runners := utils.Map(actions, func(action string) runner.Runnable {
		r, err := runner.ParseAction(action)
		if err != nil {
			log.Warnf("parse action failed `%s`", action)
			return nil
		}
		return r
	})
	runners = utils.Filter(runners, func(r runner.Runnable) bool {
		return r != nil
	})
	if len(runners) == 0 {
		log.Fatal("no valid runner")
	}
	runnerTopics := utils.Map(runners, func(r runner.Runnable) string {
		return r.Topic()
	})

	var client *bemfa.Client

	handleMessage := func(_ mqtt.Client, message mqtt.Message) {
		log.Infof("receive topic %v", message.Topic())

		topic := message.Topic()
		for _, r := range runners {
			if r.Topic() == topic {
				if err := r.Run(); err != nil {
					log.Errorf("exec failed %s", err.Error())
				}
			}
		}
	}

	onConnected := func(_ mqtt.Client) {
		log.Info("connected")
		log.Infof("subscribing %s", strings.Join(runnerTopics, ", "))
		for _, topic := range runnerTopics {
			if err = client.Subscribe(topic, handleMessage); err != nil {
				log.Errorf("subscribe %s failed %s", topic, err.Error())
			}
		}
	}

	client = bemfa.New(mqtt.NewClientOptions().
		AddBroker(server).
		SetClientID(token).
		SetOrderMatters(false).
		SetConnectRetry(true).
		SetOnConnectHandler(onConnected).
		SetReconnectingHandler(func(_ mqtt.Client, _ *mqtt.ClientOptions) {
			log.Infoln("reconnecting...")
		}).
		SetConnectionLostHandler(func(_ mqtt.Client, err error) {
			log.Warnf("connection lost, reason %s", err.Error())
		}))

	err = client.Connect()
	if err != nil {
		log.Fatalf("connect error %s", err.Error())
	}
	defer client.Disconnect()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit

	if err = client.Unsubscribe(runnerTopics...); err != nil {
		log.Warnf("unsubscribe %s failed %s", strings.Join(runnerTopics, ", "), err.Error())
	}
	log.Info("exited")
}
