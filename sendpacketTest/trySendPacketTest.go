package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/mdlayher/ethernet"
)

type networkOpt struct {
	Idx       int
	name      string
	IP        string
	hwAddress net.HardwareAddr
}

const (
	defaultPort = ":9999"
	etherType   = 0xcccc
)

var (
	txCon *net.IPConn
	rxCon *net.IPConn
	IPMap = map[int]networkOpt{}
)

func check(msg string, err error) {
	if err != nil {
		fmt.Println(msg+":", err)
	}
}

// for darwin / win32
// mac or ipv6 will be the form of "fe80:3c4e:...."
// so use "." to identify
func getInterfaces() {
	IPMap = map[int]networkOpt{}
	infs, err := net.Interfaces()
	check("inf", err)
	for _, i := range infs {
		fmt.Println("interface is", i)
		fmt.Println(i.HardwareAddr)
		addrs, err := i.Addrs()
		check("addr", err)
		for _, a := range addrs {
			if strings.Contains(a.String(), ".") && !strings.Contains(a.String(), "169.254.") && !strings.Contains(a.String(), "127.0.0.1") {
				name := i.Name
				idx := i.Index
				ip := strings.Split(a.String(), "/")[0]
				IPMap[idx] = networkOpt{name: name, IP: ip, hwAddress: i.HardwareAddr}
			}
		}
	}
}

func startSendMessages(from, to, msg string) {
	fmt.Println("start send")
	// // Message is broadcast to all machines in same network segment.
	fmt.Println("fromIfi name", from)
	fromIfi, err := net.InterfaceByName(from)
	check("from ifi error", err)
	fmt.Println("toIfi name", to)
	toIfi, err := net.InterfaceByName(to)
	check("to ifi err", err)

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

	fmt.Println("openlive => ", from)

	handle, err := pcap.OpenLive(from, 1600, false, 100)
	fmt.Println("send handler is", handle)
	check("openlive send", err)
	for {
		time.Sleep(time.Second)
		fmt.Println("send")
		handle.WritePacketData(b)
	}
}

func startReceiveMessages(ifName string) {
	fmt.Println("start receive")
	handle, err := pcap.OpenLive(ifName, 1600, false, 100)
	fmt.Println("receive handler is ", handle)
	check("openlive receive", err)

	fmt.Println("read")
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		// Process packet here
		log.Println(packet)
	}
}

func main() {
	getInterfaces()
	fmt.Println(IPMap)

	from := flag.Int("from", -1, "source interface index")
	to := flag.Int("to", -1, "destination interface index")
	fromName := flag.String("fName", "none", "source interface name")
	toName := flag.String("toName", "none", "destination interface name")
	msg := flag.String("m", "walter", "msg to send")
	flag.Parse()

	if *fromName != "none" && *toName != "none" {
		go startReceiveMessages(*toName)
		go startSendMessages(*fromName, *toName, *msg)
	} else {
		go startReceiveMessages(IPMap[*to].name)
		go startSendMessages(IPMap[*from].name, IPMap[*to].name, *msg)
	}

	stop := make(chan bool)
	stop <- true
}
