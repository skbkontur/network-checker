package main

import (
	"fmt"
	"os"
	"time"

	"kontur.ru/edoops/network-checker/ztrace"
)

func resolveGeoipDir() string {
	localDir := "geoip"
	if _, err := os.Stat(localDir); err != nil {
		return "/usr/local/lib/geoip"
	}

	return localDir
}

func traceroute(destination string, ports []uint16) {
	geoipDir := resolveGeoipDir()
	t := ztrace.New(
		"icmp", destination, "", 30, uint8(64), float32(3), 0, false,
		fmt.Sprintf("%s/asn.mmdb", geoipDir), fmt.Sprintf("%s/geoip.mmdb", geoipDir))
	t.Latitude = 31.02
	t.Longitude = 121.1
	t.TCPProbePorts = ports

	t.Start()
	time.Sleep(time.Second * 50)
	t.Print()
	t.Stop()
}
