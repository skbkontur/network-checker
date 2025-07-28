package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

type uint16Slice []uint16

// String returns the string representation of the flag's value
func (i *uint16Slice) String() string {
	strVals := make([]string, len(*i))
	for idx, val := range *i {
		strVals[idx] = strconv.FormatUint(uint64(val), 10)
	}
	return strings.Join(strVals, ",")
}

// Set appends a new value to the slice
func (i *uint16Slice) Set(value string) error {
	val, err := strconv.ParseUint(value, 10, 16)
	if err != nil {
		return err
	}
	*i = append(*i, uint16(val))
	return nil
}

func main() {

	destinationPtr := flag.String("destination", "diadoc.kontur.ru", "a string")
	var ports uint16Slice
	flag.Var(&ports, "port", "Specify one or more ports (can be repeated)")
	flag.Parse()
	if len(ports) == 0 {
		ports.Set("443")
	}

	log.Info("Get local network settings")

	fmt.Println("\nPublic IP")
	if err := getPublicIP(); err != nil {
		log.Error("Can't get public IP")
	}

	fmt.Println("\nInterfaces")
	if err := ShowNetInterfaces(); err != nil {
		log.Error("Can't fetch info about local network interfaces")
	}

	fmt.Println("\nRoutes:")
	if err := getRoutes(); err != nil {
		log.Error("Can't fetch info about local network interfaces")
	}

	fmt.Println()
	log.Infof("Start network diagnostics\nTarget destination: %s\nPorts: %s\n", *destinationPtr, ports.String())
	ip := net.ParseIP(*destinationPtr)
	if ip == nil {
		fmt.Printf("\nResolution for fqdn (%s):", *destinationPtr)
		fmt.Println()
		ips, err := getDNSNameResolution(*destinationPtr)
		if err != nil {
			log.Errorf("Can`t resolve %s", *destinationPtr)
			return
		}
		ip = ips[0]
	}

	fmt.Printf("\nRoute to destination %s (%s):", *destinationPtr, ip.String())
	fmt.Println()
	err := getRoutesByDestination(ip)
	if err != nil {
		log.Errorf("Can't get route to %s", ip.String())
	}

	fmt.Printf("\nTcp-connection to ports (%s):", ports.String())
	fmt.Println()
	isPortsOpen(fmt.Sprint(*destinationPtr), ports, 2*time.Second)

	fmt.Printf("\nTraceroute to %s (%s):", *destinationPtr, ip.String())
	traceroute(ip.String(), ports)
}
