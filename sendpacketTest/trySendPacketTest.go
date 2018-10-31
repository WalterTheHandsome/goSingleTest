package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"runtime"
	"strings"
	"time"

	"github.com/google/gopacket/pcap"
	"github.com/mdlayher/ethernet"
)

type networkOpt struct {
	Idx       int
	NameUI    string
	NameSys   string
	IP        string
	hwAddress net.HardwareAddr
}

const (
	defaultPort = ":9999"
	etherType   = 0xcccc
)

var (
	txCon        *net.IPConn
	rxCon        *net.IPConn
	interfaceMap = map[string]networkOpt{}
)

func check(msg string, err error) {
	if err != nil {
		fmt.Println(msg+":", err)
	}
}

// This function parse the interface from pcap and net and parse them into 1 structure
func getInterfaces() {
	interfaceMap = map[string]networkOpt{}

	infs, err := net.Interfaces()
	check("inf", err)
	devs, err := pcap.FindAllDevs()
	check("find all devs error", err)
	for _, i := range infs {
		addrs, err := i.Addrs()
		check("addr", err)
		for _, a := range addrs {
			if strings.Contains(a.String(), ".") && !strings.Contains(a.String(), "169.254.") && !strings.Contains(a.String(), "127.0.0.1") {
				name := i.Name
				ip := strings.Split(a.String(), "/")[0]
				for _, d := range devs {
					for _, addr := range d.Addresses {
						if ip == addr.IP.String() {
							interfaceMap[ip] = networkOpt{NameUI: name, IP: ip, hwAddress: i.HardwareAddr, NameSys: d.Name}
						}
					}
				}
			}
		}
	}

}

func startSendMessages(from, to networkOpt, msg string) {
	fmt.Println("start send")
	// // Message is broadcast to all machines in same network segment.
	fromIfi, err := net.InterfaceByName(from.NameUI)
	toIfi, err := net.InterfaceByName(to.NameUI)

	fmt.Println("msg", msg)

	f := &ethernet.Frame{
		Destination: toIfi.HardwareAddr,
		Source:      fromIfi.HardwareAddr,
		EtherType:   etherType,
		Payload:     []byte(msg),
	}

	b, err := f.MarshalBinary()
	if err != nil {
		log.Fatalf("failed to marshal ethernet frame: %v", err)
	}

	name := ""
	if runtime.GOOS == "windows" {
		name = from.NameSys
	} else {
		name = from.NameUI
	}
	fmt.Println("openlive => ", name)
	handle, err := pcap.OpenLive(name, 1600, false, pcap.BlockForever)
	fmt.Println("send handler is", handle)
	check("openlive send", err)
	for {
		time.Sleep(time.Second)
		fmt.Println("send")
		handle.WritePacketData(b)
	}
}

func startReceiveMessages(from networkOpt) {
	fmt.Println("start receive")
	name := ""
	if runtime.GOOS == "windows" {
		name = from.NameSys
	} else {
		name = from.NameUI
	}
	fmt.Println("openLive name is", name)
	handle, err := pcap.OpenLive(name, 1600, false, pcap.BlockForever)
	fmt.Println("receive handler is ", handle)
	check("openlive receive", err)

	for {
		fmt.Println("read")
		data, _, err := handle.ReadPacketData()
		check("read err", err)
		r := &ethernet.Frame{}
		r.UnmarshalBinary(data)
		fmt.Println("data ", string(r.Payload))
	}
}

func main() {
	getInterfaces()
	fmt.Println(interfaceMap)

	from := flag.String("from", "none", "source interface IP address")
	to := flag.String("to", "none", "destination interface IP address")
	msg := flag.String("m", "walter", "msg to send")
	flag.Parse()

	go startReceiveMessages(interfaceMap[*to])
	go startSendMessages(interfaceMap[*from], interfaceMap[*to], *msg)

	stop := make(chan bool)
	stop <- true
}
