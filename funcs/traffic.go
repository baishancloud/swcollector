package funcs

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"os/exec"
	"strconv"
	"strings"

	pfc "github.com/baishancloud/goperfcounter"
	"github.com/baishancloud/octopux-swcollector/g"
	"github.com/open-falcon/common/model"
	"github.com/toolkits/nux"
)

var (
	lanlist        []net.IPNet
	iptablesbinary string
	//ipface  map[string]string
)

func init() {
	var lookErr error
	iptablesbinary, lookErr = exec.LookPath("iptables")
	if lookErr != nil {
		//panic(lookErr)
		log.Println("cant find iptables!")
	}
}

func getIpFaces() (map[string]string, []*net.IPNet) {
	facelist, err := net.Interfaces()
	if err != nil {
		log.Println("ERROR: get interfaces!", err)
		return nil, nil
	}
	if len(lanlist) == 0 {
		if g.Config().Collector == nil || g.Config().Collector.LanIpnet == nil {
			return nil, nil
		}
		lanstrs := g.Config().Collector.LanIpnet
		parseIPNets(&lanstrs, &lanlist)

	}
	ipFaces := make(map[string]string)
	nets := make([]*net.IPNet, 0)
	for _, iface := range facelist {
		if (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagLoopback) != 0 || (iface.Flags&net.FlagPointToPoint) != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if len(addrs) == 0 || err != nil {
			continue
		}

		islan := false

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipnet.IP
			if ip.To4() == nil {
				continue
			}

			for _, in := range lanlist {
				if in.Contains(ip) {
					islan = true
					break
				}
			}
			if islan == false {
				ipFaces[ip.String()] = iface.Name
				nets = append(nets, ipnet)

			}

		}
	}
	return ipFaces, nets
}

func parseIPNets(lanstrs *[]string, nw *[]net.IPNet) {

	rows := len(*lanstrs)
	if rows == 0 {
		return
	}
	nw1 := make([]net.IPNet, 0, rows)
	i := 0
	for _, s := range *lanstrs {
		_, ipNet, err := net.ParseCIDR(s)
		if err == nil {
			i++
			nw1 = append(nw1, *ipNet)
		}
	}
	*nw = (nw1[:i])
	return
}

func resetipt() {
	cmdrt := exec.Command("/usr/local/bin/ipt_server.sh")
	_, err := cmdrt.Output()
	if err != nil {
		log.Println("reset iptable error", err)
	}
}

func ipchain(t string, ip string) string {
	switch t {
	case "ai":
		return "traffic_in_" + ip
	case "li":
		return "traffic_lan_in_" + ip
	case "ao":
		return "traffic_out_" + ip
	case "lo":
		return "traffic_lan_out_" + ip
		// ...
	default:
		return "no_find"
	}
}

func CmdTimeout(timeout int, name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)

	var out bytes.Buffer
	cmd.Stdout = &out

	cmd.Start()
	timer := time.AfterFunc(time.Duration(timeout)*time.Millisecond, func() {
		err := cmd.Process.Kill()
		if err != nil {
			log.Println("failed to kill: ", err)
		}
	})
	err := cmd.Wait()
	timer.Stop()

	return out.String(), err
}
func getIptableTraiffic(in string) (out uint64, err error) {
	if iptablesbinary == "" {
		return 0, errors.New("cant find iptables!")
	}

	iout, err := CmdTimeout(1000, iptablesbinary, "-L", in, "1", "-vnx")
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Exec  %s iptable chain error %s", in, err.Error()))
	}
	fis := strings.Fields(iout)
	if len(fis) <= 6 {
		return 0, errors.New(fmt.Sprintf("read  %s iptable output error %s", in, iout))
	}
	out, err = strconv.ParseUint(fis[1], 10, 64)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("%s convert uint error %s", in, err.Error()))
	}
	return

}

func getTraffic(ipt string) (wiv uint64, liv uint64, wov uint64, lov uint64, err error) {

	aiv, err := getIptableTraiffic(ipchain("ai", ipt))
	if err != nil {
		return 0, 0, 0, 0, err
	}
	liv, err = getIptableTraiffic(ipchain("li", ipt))
	if err != nil {
		return 0, 0, 0, 0, err
	}
	aov, err := getIptableTraiffic(ipchain("ao", ipt))
	if err != nil {
		return 0, 0, 0, 0, err
	}

	lov, err = getIptableTraiffic(ipchain("lo", ipt))
	if err != nil {
		return 0, 0, 0, 0, err
	}

	wiv = aiv - liv
	wov = aov - lov

	return wiv, liv, wov, lov, nil

}

func TrafficMetrics() (L []*model.MetricValue) {
	interfaceMetrics := CoreInterfaceMetrics()
	trafficeMetrics := CoreTrafficMetrics()
	L = append(L, interfaceMetrics...)
	L = append(L, trafficeMetrics...)
	return L
}

func CoreInterfaceMetrics() (L []*model.MetricValue) {
	myip, myipconf := g.ConfigIp()
	iptag := "ip=" + myip
	if !myipconf {
		iptag = "ip=" + g.IP()
		myip = g.IP()
	}
	log.Println("myip: ", myip)
	conf := g.Config()
	ifacePrefix := conf.Collector.IfacePrefix
	ctime := time.Now().Unix()
	netIfs, err := nux.NetIfs(ifacePrefix)
	if err != nil {
		log.Println(err)
		return []*model.MetricValue{}
	}

	for _, netIf := range netIfs {
		ifacetag := "iface=" + netIf.Iface
		if !conf.Rate {
			L = append(L, CounterValue("netcard.if.in.bytes", netIf.InBytes, ifacetag, iptag))
			L = append(L, CounterValue("netcard.if.out.bytes", netIf.OutBytes, ifacetag, iptag))
		} else {
			inr, ts, err := g.Rate(myip, netIf.Iface, "ifin", uint64(netIf.InBytes), ctime)
			if err == nil {
				L = append(L, GaugeValueSliceTS("ifinrate", ts, inr, ifacetag, iptag)...)
			}
			outr, ts, err := g.Rate(myip, netIf.Iface, "ifout", uint64(netIf.OutBytes), ctime)
			if err == nil {
				L = append(L, GaugeValueSliceTS("ifoutrate", ts, outr, ifacetag, iptag)...)
			}
		}
	}
	return L
}

func convIpstrToIptChane(ipstr string) string {
	ip := net.ParseIP(ipstr)
	ip = ip.To4()
	if ip == nil {
		return "error"
	}
	ipbytes := []byte(ip)

	return fmt.Sprintf("%x.%x.%x.%x", ipbytes[0], ipbytes[1], ipbytes[2], ipbytes[3])

}

func CoreTrafficMetrics() (L []*model.MetricValue) {
	ipfs, nets := getIpFaces()
	if ipfs == nil || len(ipfs) == 0 {
		log.Println("get faces error")
		return
	}
	myip, myipconf := g.ConfigIp()
	iptag := "ip=" + myip
	if len(ipfs) != 1 && myipconf {
		myipconf = false
	}
	conf := g.Config()

	for ip, iface := range ipfs {
		iptstr := convIpstrToIptChane(ip)
		wi, li, wo, lo, err := getTraffic(iptstr)
		ctime := time.Now().Unix()
		if !myipconf {
			iptag = "ip=" + ip
		}
		ifsub := strings.Split(iface, ".")
		ifacetag := "iface=" + ifsub[0]
		if err != nil {
			nip := net.ParseIP(ip)
			cnt := 0
			for k, n := range nets {
				if n != nil && n.Contains(nip) {
					cnt++
					if cnt == 1 {
						nets[k] = nil
					}
				}
			}
			if cnt == 1 {
				pfc.Report("SWCLGetTrafErr", ip)
				log.Println("get taffic falure. ", err)
			}
			//L = append(L, GaugeValue("traffic.collect.status", 1, ifacetag, iptag))
			continue
		}
		L = append(L, GaugeValue("traffic.collect.status", 0, ifacetag, iptag))
		if !conf.Rate {
			L = append(L, CounterValue("traffic.wan.in", wi, ifacetag, iptag))
			L = append(L, CounterValue("traffic.lan.in", li, ifacetag, iptag))
			L = append(L, CounterValue("traffic.wan.out", wo, ifacetag, iptag))
			L = append(L, CounterValue("traffic.lan.out", lo, ifacetag, iptag))
		} else {

			wir, ts, err := g.Rate(ip, ifsub[0], "wi", wi, ctime)
			if err == nil {
				L = append(L, GaugeValueSliceTS("waninrate", ts, wir, ifacetag, iptag)...)
			}
			wor, ts, err := g.Rate(ip, ifsub[0], "wo", wo, ctime)
			if err == nil {
				L = append(L, GaugeValueSliceTS("wanoutrate", ts, wor, ifacetag, iptag)...)
			}
			lir, ts, err := g.Rate(ip, ifsub[0], "li", li, ctime)
			if err == nil {
				L = append(L, GaugeValueSliceTS("laninrate", ts, lir, ifacetag, iptag)...)
			}
			lor, ts, err := g.Rate(ip, ifsub[0], "lo", lo, ctime)
			if err == nil {
				L = append(L, GaugeValueSliceTS("lanoutrate", ts, lor, ifacetag, iptag)...)
			}
		}
	}
	return L
}
