package main

import (
	"net"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

func getDNSNameResolution(fqdn string) ([]net.IP, error) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"FQDN", "IP-Address"})

	ips, err := net.LookupIP(fqdn)
	if err != nil {
		return nil, err
	}
	var ipStrings []string
	for _, ip := range ips {
		if ip != nil { // Ensure the IP is not nil (e.g., from ParseIP errors)
			ipStrings = append(ipStrings, ip.String())
		}
	}

	t.AppendRow(table.Row{fqdn, strings.Join(ipStrings, ",")})
	t.Render()
	return ips, nil
}
