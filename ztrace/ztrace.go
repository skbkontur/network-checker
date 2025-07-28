package ztrace

import (
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"kontur.ru/edoops/network-checker/ztrace/geoip"
	"kontur.ru/edoops/network-checker/ztrace/tsyncmap"
)

type SendMetric struct {
	FlowKey   string
	ID        uint32
	TTL       uint8
	TimeStamp time.Time
}

type RecvMetric struct {
	FlowKey   string
	ID        uint32
	RespAddr  string
	TimeStamp time.Time
}

type TraceRoute struct {
	SrcAddr       string
	Dest          string
	TCPDPort      uint16
	TCPProbePorts []uint16
	MaxPath       int
	MaxTTL        uint8
	Protocol      string
	PacketRate    float32 //pps
	SendChan      chan *SendMetric
	RecvChan      chan *RecvMetric
	WideMode      bool
	PortOffset    int32

	netSrcAddr net.IP //used for raw socket and TCP-Traceroute
	netDstAddr net.IP

	af         string //ip4 or ip6
	stopSignal *int32 //atomic Counters,stop when cnt =1

	recvICMPConn *net.IPConn
	recvTCPConn  *net.IPConn
	geo          *geoip.GeoIPDB

	//stats
	DB        sync.Map
	Metric    []map[string]*ServerRecord
	Latitude  float64
	Longitude float64
	Lock      *sync.RWMutex
}
type StatsDB struct {
	Cache   *tsyncmap.Map
	SendCnt *uint64
}

func NewStatsDB(key string) *StatsDB {
	cacheTimeout := time.Duration(6 * time.Second)
	checkFreq := time.Duration(1 * time.Second)
	var cnt uint64
	px := &StatsDB{
		Cache:   tsyncmap.NewMap(key, cacheTimeout, checkFreq, false),
		SendCnt: &cnt,
	}
	return px
}

func (t *TraceRoute) validateSrcAddress() error {
	if t.SrcAddr != "" {
		addr, err := net.ResolveIPAddr(t.af, t.SrcAddr)
		if err != nil {
			return err
		}
		t.netSrcAddr = addr.IP
		return nil
	}

	//if config does not specify address, fetch local address
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		logrus.Fatal(err)
	}
	result := conn.LocalAddr().(*net.UDPAddr)
	conn.Close()
	t.netSrcAddr = result.IP
	return nil
}

func (t *TraceRoute) VerifyCfg() {
	rAddr, err := net.LookupIP(t.Dest)
	if err != nil {
		logrus.Fatal("dst address validation:", err)
	}
	t.netDstAddr = rAddr[0]

	//update address family
	t.af = "ip4"
	if strings.Contains(t.netDstAddr.String(), ":") {
		t.af = "ip6"
	}

	//verify source address
	err = t.validateSrcAddress()
	if err != nil {
		logrus.Fatal(err)
	}

	var sig int32 = 0
	t.stopSignal = &sig
	atomic.StoreInt32(t.stopSignal, 0)

	if t.MaxPath > 32 {
		logrus.Fatal("Only support max ECMP = 32")
	}
	if t.MaxTTL > 64 {
		logrus.Warn("Large TTL may cause low performance")
	}

	if t.PacketRate < 0 {
		logrus.Fatal("Invalid pps")
	}
}

func New(protocol string, dest string, src string, maxPath int, maxTtl uint8, pps float32, portoffset int, wmode bool, asncfg string, geocfg string) *TraceRoute {
	result := &TraceRoute{
		SrcAddr:       src,
		Dest:          dest,
		TCPDPort:      443,
		TCPProbePorts: []uint16{80, 8080, 443, 8443},
		Protocol:      protocol,
		MaxPath:       maxPath,
		MaxTTL:        maxTtl,
		PacketRate:    pps,
		WideMode:      wmode,
		SendChan:      make(chan *SendMetric, 10),
		RecvChan:      make(chan *RecvMetric, 10),
		geo:           geoip.New(geocfg, asncfg),
		PortOffset:    int32(portoffset),
	}
	result.VerifyCfg()
	result.Lock = &sync.RWMutex{}

	result.Metric = make([]map[string]*ServerRecord, int(maxTtl)+1)
	for i := 0; i < len(result.Metric); i++ {
		result.Metric[i] = make(map[string]*ServerRecord)
	}
	return result
}

func (t *TraceRoute) ProbeTCP() {
	for _, port := range t.TCPProbePorts {
		go t.IPv4TCPProbe(port)
	}
}

func (t *TraceRoute) TraceUDP() {
	for i := 0; i < t.MaxPath; i++ {
		go t.SendIPv4UDP()
	}
	go t.ListenIPv4UDP()
}

func (t *TraceRoute) TraceTCP() {
	for i := 0; i < t.MaxPath; i++ {
		go t.SendIPv4TCP(t.TCPDPort)
	}
	go t.ListenIPv4TCP()
	go t.ListenIPv4TCP_ICMP()
}

func (t *TraceRoute) TraceICMP() {
	go t.SendIPv4ICMP()
	go t.ListenIPv4ICMP()
}

func (t *TraceRoute) Start() {
	go t.Stats()

	switch t.Protocol {
	case "tcp":
		go t.TraceTCP()
	case "udp":
		go t.TraceUDP()
		go t.ProbeTCP()
	case "icmp":
		go t.TraceICMP()
		go t.ProbeTCP()
	default:
		logrus.Fatal("unsupported protocol: only support tcp/udp/icmp")
	}
}

func (t *TraceRoute) Stop() {
	atomic.StoreInt32(t.stopSignal, 1)
	t.recvICMPConn.Close()
	//t.recvTCPConn.Close()
}
