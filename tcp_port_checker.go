package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
)

func isPortsOpen(host string, ports []uint16, timeout time.Duration) uint16 {
	var anyOpenPort uint16 = 0
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Host", "Port", "Result"})

	for _, port := range ports {
		address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err != nil {
			t.AppendRow(table.Row{host, port, "Closed"})
			continue
		}
		conn.Close()
		t.AppendRow(table.Row{host, port, "Open"})
		anyOpenPort = port
	}
	t.Render()
	return anyOpenPort
}
