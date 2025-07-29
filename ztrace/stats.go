package ztrace

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/olekukonko/tablewriter"
	"kontur.ru/edoops/network-checker/ztrace/geoip"
	"kontur.ru/edoops/network-checker/ztrace/stats/describe"
	"kontur.ru/edoops/network-checker/ztrace/stats/quantile"
)

type ServerRecord struct {
	TTL             uint8
	Addr            string
	Name            string
	Session         string
	GeoLocation     geoip.GeoLocation
	LatencyDescribe *describe.Item
	Quantile        *quantile.Stream
	RecvCnt         uint64
	Lock            *sync.Mutex
}

func (s *ServerRecord) LookUPAddr() {
	rA, _ := net.LookupAddr(s.Addr)
	var buf bytes.Buffer
	for _, item := range rA {
		if len(item) > 0 {
			//some platform may add dot in suffix
			item = strings.TrimSuffix(item, ".")
			if !strings.HasSuffix(item, ".in-addr.arpa") {
				buf.WriteString(item)
			}
		}
	}
	s.Name = buf.String()
}
func (t *TraceRoute) NewServerRecord(ipaddr string, ttl uint8, key string) *ServerRecord {
	r := &ServerRecord{
		TTL:             ttl,
		Addr:            ipaddr,
		LatencyDescribe: describe.New(),
		Session:         key,
		Quantile: quantile.NewTargeted(map[float64]float64{
			0.50: 0.005,
			0.90: 0.001,
			0.99: 0.0001,
		}),
		RecvCnt: 0,
		Lock:    &sync.Mutex{},
	}
	if strings.Contains(ipaddr, "tcp") {
		addr := strings.Split(ipaddr, ":")
		r.Addr = addr[1] + ":" + addr[2]
		r.GeoLocation = t.geo.Lookup(addr[1])
	} else {
		r.GeoLocation = t.geo.Lookup(ipaddr)
	}

	return r
}

func (t *TraceRoute) Stats() {
	for {
		select {
		case v := <-t.SendChan:
			tdb, ok := t.DB.Load(v.FlowKey)
			if !ok {
				continue
			}
			db := tdb.(*StatsDB)
			db.Cache.Store(v.ID, v, v.TimeStamp)

		case v := <-t.RecvChan:
			tdb, ok := t.DB.Load(v.FlowKey)
			if !ok {
				continue
			}
			db := tdb.(*StatsDB)
			tsendInfo, valid := db.Cache.Load(v.ID)
			if !valid {
				continue
			}
			sendInfo := tsendInfo.(*SendMetric)
			server, valid := t.Metric[sendInfo.TTL][v.RespAddr]
			//create server
			if !valid {
				server = t.NewServerRecord(v.RespAddr, uint8(sendInfo.TTL), sendInfo.FlowKey)
				t.Metric[sendInfo.TTL][v.RespAddr] = server
			}

			server.Lock.Lock()
			server.RecvCnt++
			latency := float64(v.TimeStamp.Sub(sendInfo.TimeStamp) / time.Microsecond)
			//logrus.Info(v.RespAddr, ":", latency)
			server.LatencyDescribe.Append(latency, 2)
			server.Quantile.Insert(latency)
			server.Lock.Unlock()
			if server.Name == "" {
				go server.LookUPAddr()
			}

		}
		if atomic.LoadInt32(t.stopSignal) == 1 {
			return
		}
	}
}

func GetColorByLatency(latency float64) tablewriter.Colors {
	if latency < 20 {
		return tablewriter.Colors{tablewriter.FgHiGreenColor}
	}
	if latency > 150 {
		return tablewriter.Colors{tablewriter.FgHiRedColor}
	}
	if latency > 100 {
		return tablewriter.Colors{tablewriter.FgHiYellowColor}
	}
	return tablewriter.Colors{}
}

func GetColorByLoss(loss float32) tablewriter.Colors {
	if loss < 0.5 {
		return tablewriter.Colors{tablewriter.FgHiGreenColor}
	}
	if loss > 10 {
		return tablewriter.Colors{tablewriter.FgHiRedColor}
	}
	if loss > 3 {
		return tablewriter.Colors{tablewriter.FgHiYellowColor}
	}
	return tablewriter.Colors{}
}

func (t *TraceRoute) PrintRow(table *tablewriter.Table, id int) string {

	ColorNormal := tablewriter.Colors{}

	RespAddr := ""
	hid := fmt.Sprintf("%4d", id)
	if id == 0 {
		hid = "TCP"
	}
	for _, v := range t.Metric[id] {

		latency := fmt.Sprintf("%8.2fms", v.LatencyDescribe.Mean/1000)
		jitter := fmt.Sprintf("%8.2fms", v.LatencyDescribe.Std()/1000)
		p95 := fmt.Sprintf("%12.2fms", v.Quantile.Query(0.95)/1000)
		tdb, ok := t.DB.Load(v.Session)

		loss := float32(0)
		if ok {
			statsDB := tdb.(*StatsDB)
			sendCnt := atomic.LoadUint64(statsDB.SendCnt)
			if sendCnt != 0 {
				loss = (1 - float32(v.RecvCnt)/float32(sendCnt)) * 100
			}

			if v.RecvCnt > sendCnt {
				loss = 0
			}
		}

		city := fmt.Sprintf("%-16.16s", v.GeoLocation.City)
		country := fmt.Sprintf("%-16.16s", v.GeoLocation.Country)
		asn := fmt.Sprintf("%-10d", v.GeoLocation.ASN)

		sp := fmt.Sprintf("%-16.16s", v.GeoLocation.SPName)
		saddr := fmt.Sprintf("%-21.21s", v.Addr)

		if RespAddr == "" {
			RespAddr = v.Addr
		}
		sname := fmt.Sprintf("%-26.26s", v.Name)
		if t.WideMode {
			sname = fmt.Sprintf("%-30.30s", v.Name)
			sp = fmt.Sprintf("%-30.30s", v.GeoLocation.SPName)
			distance := geoip.ComputeDistance(t.Latitude, t.Longitude, v.GeoLocation.Latitude, v.GeoLocation.Longitude)

			latencyByDistance := distance/75 + float64(id)*3
			if id == 0 {
				latencyByDistance += 30
			}
			/*
			  LightSpeed over Fiber is nearly 150,000km/s
			  RTT(ms) = distance *2 / Fiber_LightSpeed *1000 = 2 * distance /150,000 * 1000 = distance /100
			  Each hop contribute 3ms latency,based on average QoS and forwarding latency estimation
			*/
			distanceStr := fmt.Sprintf("%6.0fkm[%3.0fms]", distance, latencyByDistance)
			if v.GeoLocation.Latitude == 0 && v.GeoLocation.Longitude == 0 {
				distanceStr = fmt.Sprintf("%12s", "")
			}
			data := []string{hid, saddr, sname, city, country, asn, sp, distanceStr, p95, latency, jitter, fmt.Sprintf("%4.1f%%", loss)}
			rowColor := make([]tablewriter.Colors, len(data))
			for i := 0; i < len(data); i++ {
				rowColor[i] = ColorNormal
			}
			rowColor[8] = GetColorByLatency(v.Quantile.Query(0.95) / 1000)
			rowColor[9] = GetColorByLatency(v.LatencyDescribe.Mean / 1000)
			rowColor[11] = GetColorByLoss(loss)
			table.Rich(data, rowColor)
		} else {
			data := []string{hid, saddr, sname, country, sp, p95, latency, jitter, fmt.Sprintf("%4.1f%%", loss)}
			rowColor := make([]tablewriter.Colors, len(data))
			for i := 0; i < len(data); i++ {
				rowColor[i] = ColorNormal
			}
			rowColor[5] = GetColorByLatency(v.Quantile.Query(0.95) / 1000)
			rowColor[6] = GetColorByLatency(v.LatencyDescribe.Mean / 1000)
			rowColor[8] = GetColorByLoss(loss)
			table.Rich(data, rowColor)
		}
		if hid != "" {
			hid = ""
		}
	}
	return RespAddr
}

func (t *TraceRoute) Print() {
	//fmt.Printf("\033[H\033[2J")
	fmt.Printf("\n[%s]Traceroute Report\n\n", t.Dest)

	table := tablewriter.NewWriter(os.Stdout)

	if t.WideMode {
		table.SetHeader([]string{"TTL  ", "Server", "Name", "City", "Country", "ASN", "SP", "Distance[tRTT]", "p95", "Latency", "Jitter", "Loss"})
	} else {
		table.SetHeader([]string{"TTL  ", "Server", "Name", "Country", "SP", "p95", "Latency", "Jitter", "Loss"})

	}
	table.SetAutoFormatHeaders(false)
	/*
		table.SetRowLine(true)
		if t.WideMode {
			table.SetAutoMergeCellsByColumnIndex([]int{0, 3, 4, 5, 6})
		} else {
			table.SetAutoMergeCellsByColumnIndex([]int{0, 3, 4, 5})
		}*/
	for ttl := 1; ttl <= int(t.MaxTTL); ttl++ {
		respAddr := t.PrintRow(table, ttl)

		if strings.Contains(respAddr, fmt.Sprintf(":%d", t.TCPDPort)) {
			break
		}
		if respAddr == t.netDstAddr.String() {
			break
		}
	}

	table.Render()

	if t.Protocol != "tcp" {
		t1 := tablewriter.NewWriter(os.Stdout)

		if t.WideMode {
			t1.SetHeader([]string{"Probe", "Server", "Name", "City", "Country", "ASN", "SP", "Distance[tRTT]", "p95", "Latency", "Jitter", "Loss"})
		} else {
			t1.SetHeader([]string{"Probe", "Server", "Name", "Country", "SP", "p95", "Latency", "Jitter", "Loss"})
		}

		t1.SetAutoFormatHeaders(false)
		t.PrintRow(t1, 0)
		t1.Render()
	}
}

func (t *TraceRoute) Report(freq time.Duration) {
	for {
		t.Print()
		if atomic.LoadInt32(t.stopSignal) == 1 {
			return
		}
		time.Sleep(freq)
	}
}
