package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/jedib0t/go-pretty/v6/table"
)

func ShowNetInterfaces() error {

	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	var ifacesWithWarning []string
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "Name", "MTU", "Flags", "Addresses"})

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 &&
			iface.Flags&net.FlagRunning != 0 &&
			iface.Flags&net.FlagLoopback != 1 {
			addrs, err := iface.Addrs()
			if err != nil {
				ifacesWithWarning = append(ifacesWithWarning, iface.Name)
				continue
			}
			var ipList []string
			for _, a := range addrs {

				ip, ipnet, err := net.ParseCIDR(a.String())
				if err != nil {
					ifacesWithWarning = append(ifacesWithWarning, iface.Name)
					continue
				}
				ipList = append(ipList, fmt.Sprintf("%s (%s)", ip, ipnet))
			}

			t.AppendRow(table.Row{iface.Index, iface.Name, iface.MTU, iface.Flags.String(), strings.Join(ipList, ", ")})
		}
	}
	t.Render()
	if len(ifacesWithWarning) > 0 {
		log.Warnf("Information about interfaces (%s) is not fully (couldn`t get information about IpAddresses)", strings.Join(ifacesWithWarning, ", "))
	}
	return nil
}
