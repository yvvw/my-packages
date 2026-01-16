package runner

import "yvvw/my-packages/bemfa/internal/wol"

type wolRunner struct {
	mac   string
	iface string
	stubRunner
}

func (a *wolRunner) Run() error {
	return wol.Exec(a.iface, a.mac)
}
