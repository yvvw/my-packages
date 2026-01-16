package runner

import (
	"fmt"
	"strings"

	"yvvw/my-packages/bemfa/internal/utils"
)

const (
	wolRunnerCommand             string = "WOL"
	windowsRebootRunnerCommand          = "WIN_REBOOT"
	windowsShutdownRunnerCommand        = "WIN_SHUTDOWN"
)

type Runnable interface {
	Topic() string
	Run() error
}

func ParseAction(action string) (runner Runnable, err error) {
	args := parseArgs(action)

	argLength := len(args)
	if argLength < 2 {
		err = fmt.Errorf("invalid argument `%s`, must have topic and command", action)
		return
	}

	topic, runnerCommand := args[0], args[1]

	if runnerCommand == wolRunnerCommand {
		if argLength != 4 {
			err = fmt.Errorf("invalid wol argument `%s`, rule [topic WOL mac_address broadcast_interface]")
		}
		runner = &wolRunner{args[2], args[3], stubRunner{topic}}
	} else if runnerCommand == windowsRebootRunnerCommand {
		runner = &windowsRebootRunner{stubRunner{topic}}
	} else if runnerCommand == windowsShutdownRunnerCommand {
		runner = &windowsShutdownRunner{stubRunner{topic}}
	} else {
		err = fmt.Errorf("unknow command `%s`", runnerCommand)
	}

	return
}

func parseArgs(rawAction string) (args []string) {
	args = strings.Split(rawAction, " ")
	args = utils.Map(args, func(i string) string {
		return strings.TrimSpace(i)
	})
	args = utils.Filter(args, func(i string) bool {
		return i != ""
	})
	return
}
