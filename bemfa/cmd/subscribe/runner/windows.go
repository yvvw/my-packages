package runner

import (
	"errors"
	"os/exec"
	"runtime"
)

type windowsRebootRunner struct {
	stubRunner
}

func (a *windowsRebootRunner) Run() error {
	if runtime.GOOS != "windows" {
		return errors.New("WIN_REBOOT only support windows platform")
	}

	return exec.Command("cmd", "/C", "shutdown", "/r", "/t", "0").Run()
}

type windowsShutdownRunner struct {
	stubRunner
}

func (a *windowsShutdownRunner) Run() error {
	if runtime.GOOS != "windows" {
		return errors.New("WIN_SHUTDOWN only support windows platform")
	}

	return exec.Command("cmd", "/C", "shutdown", "/s", "/t", "0").Run()
}
