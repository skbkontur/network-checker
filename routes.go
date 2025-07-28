package main

import (
	"net"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/vishvananda/netlink"
)

func getRoutes() error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "Destination", "Dev", "Gateway"})

	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		return err
	}

	var i int
	for _, route := range routes {
		i++
		var dst string
		if route.Dst == nil {
			dst = "default"
		} else {
			dst = route.Dst.String()
		}

		var gw string
		if route.Gw != nil {
			gw = route.Gw.String()
		} else {
			gw = "-"
		}

		link, err := netlink.LinkByIndex(route.LinkIndex)
		if err != nil {
			t.AppendRow(table.Row{i, dst, "[unknown]", gw})
			continue
		}

		t.AppendRow(table.Row{i, dst, link.Attrs().Name, gw})
	}

	t.Render()
	return nil
}

func getRoutesByDestination(ip net.IP) error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "Destination", "Source", "Interface", "Gateway"})

	routes, err := netlink.RouteGet(ip)
	if err != nil {
		return err
	}

	iface, err := net.InterfaceByIndex(routes[0].LinkIndex)
	if err != nil {
		return err
	}

	t.AppendRow(table.Row{1, ip.String(), routes[0].Src, iface.Name, routes[0].Gw})
	t.Render()
	return nil
}
