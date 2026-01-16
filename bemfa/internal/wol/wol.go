package wol

import (
	"fmt"
	"net"

	"github.com/sabhiram/go-wol/wol"
	log "github.com/sirupsen/logrus"
)

func Exec(iface string, mac string) error {
	var laddr *net.UDPAddr
	if iface != "" {
		var err error
		laddr, err = ipFromInterface(iface)
		if err != nil {
			return err
		}
	}

	raddr, err := net.ResolveUDPAddr("udp", "255.255.255.255:9")
	if err != nil {
		return err
	}

	mp, err := wol.New(mac)
	if err != nil {
		return err
	}

	bs, err := mp.Marshal()
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	n, err := conn.Write(bs)
	if err == nil && n != 102 {
		err = fmt.Errorf("magic packet sent was %d bytes (expected 102 bytes sent)", n)
	}
	if err != nil {
		return err
	}

	log.Infof("magic packet sent successfully to %s over %s", mac, iface)
	return nil
}

func ipFromInterface(iface string) (*net.UDPAddr, error) {
	ief, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, err
	}

	addrs, err := ief.Addrs()
	if err != nil {
		return nil, err
	}
	if err == nil && len(addrs) <= 0 {
		err = fmt.Errorf("no address associated with interface %s", iface)
	}

	// Validate that one of the addrs is a valid network IP address.
	for _, addr := range addrs {
		switch ip := addr.(type) {
		case *net.IPNet:
			if !ip.IP.IsLoopback() && ip.IP.To4() != nil {
				return &net.UDPAddr{
					IP: ip.IP,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("no address associated with interface %s", iface)
}
