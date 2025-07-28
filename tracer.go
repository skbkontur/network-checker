package main

import (
	"kontur.ru/edoops/network-checker/ztrace"
	"time"
)

func traceroute(destination string, ports []uint16) {
	t := ztrace.New("icmp", destination, "", 30, uint8(64), float32(3), 0, false, "geoip/asn.mmdb", "geoip/geoip.mmdb")
	t.Latitude = 31.02
	t.Longitude = 121.1
	t.TCPProbePorts = ports

	t.Start()
	time.Sleep(time.Second * 50)
	t.Print()
	t.Stop()
}
