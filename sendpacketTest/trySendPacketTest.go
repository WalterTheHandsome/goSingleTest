package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/gopacket/pcap"
	"github.com/mdlayher/ethernet"
)

var (
	interfaceMap = map[string]networkOpt{}
	devNameMap   = map[string]string{}

	txInterface *infInstance
	rxInterface *infInstance

	// StartTx :
	StartTx = make(chan int, 1)
	// StartRx :
	StartRx = make(chan int, 1)
)

const (
	etherType            = 0xcccc
	defaultTxMSg         = "intrising tx test, port "
	defaultRxMsg         = "intrising rx test, port "
	windowGOOS           = "windows"
	basicPortTestTimeout = 30
	packetAmount         = 1
)

type networkOpt struct {
	NameUI    string
	NameSys   string
	IP        string
	HwAddress net.HardwareAddr
}

type infInstance struct {
	localIP     string
	remoteIP    string
	handle      *pcap.Handle
	counter     int
	role        string
	currentPort int
}

func (i *infInstance) send(port int) {
	// reset counter
	fmt.Println("start send ", i.role)
	i.counter = 0
	i.currentPort = port

	if i.handle == nil {
		fmt.Println("infInstance needs init")
		return
	}

	msg := ""
	if i.role == "tx" {
		msg += defaultTxMSg
	} else {
		msg += defaultRxMsg
	}
	msg += strconv.Itoa(port)
	fmt.Println("msg is ", msg)
	f := &ethernet.Frame{
		Destination: interfaceMap[i.remoteIP].HwAddress,
		Source:      interfaceMap[i.localIP].HwAddress,
		EtherType:   etherType,
		Payload:     []byte(msg),
	}

	b, _ := f.MarshalBinary()

	for count := 0; count < packetAmount; count++ {
		err := i.handle.WritePacketData(b)
		if err != nil {
			check("interface read error:", err)
		}
	}
}

func (i *infInstance) close() {
	if i.handle == nil {
		return
	}
	i.handle.Close()
	i.handle = nil
}

// init the instance and close the handle when parent is canceled
func (i *infInstance) init(parent context.Context, srcIP, dstIP, role string) {
	i.localIP = srcIP
	i.remoteIP = dstIP
	i.counter = 0
	i.role = role

	// need name to open a interface handle
	sourceIfi := interfaceMap[srcIP]
	name := ""
	if runtime.GOOS == windowGOOS {
		name = sourceIfi.NameSys
	} else {
		name = sourceIfi.NameUI
	}
	handle, err := pcap.OpenLive(name, 1600, false, pcap.BlockForever)

	if err != nil {
		check(srcIP+" init error:", err)
	}
	i.handle = handle
	fmt.Println("init ", i.role, "done")
	fmt.Println("start read")
	for {
		select {
		case <-parent.Done():
			fmt.Println("init return")
			return
		default:
			data, _, err := i.handle.ReadPacketData()
			if err != nil {
				check(i.localIP+" read packet err: ", err)
			}

			r := new(ethernet.Frame)
			r.UnmarshalBinary(data)
			// rm the 0
			result := strings.Trim(string(r.Payload), string([]byte{0}))

			if i.role == "rx" && result == (defaultTxMSg+strconv.Itoa(i.currentPort)) {
				i.counter++
				fmt.Println("rx counter", i.counter)
			}

			if i.role == "tx" && result == (defaultRxMsg+strconv.Itoa(i.currentPort)) {
				i.counter++
				fmt.Println("tx couter", i.counter)
			}

		}
	}
}

func check(msg string, err error) {
	if err != nil {
		fmt.Println(msg, err.Error())
	}
}

// This function parse the interface from pcap and net and parse them into 1 structure
func updateInterfacesOpts() {
	interfaceMap = map[string]networkOpt{}

	infs, err := net.Interfaces()
	check("update interface error: ", err)
	devs, err := pcap.FindAllDevs()
	check("find devicess error: ", err)

	// parse pcap devName map by ip for windows
	for _, d := range devs {
		for _, addr := range d.Addresses {
			ip := addr.IP.String()
			if strings.Contains(ip, ".") && !strings.Contains(ip, "169.254.") && !strings.Contains(ip, "127.0.0.1") {
				devNameMap[ip] = d.Name
			}
		}
	}

	// parse interfaceMap by ip
	for _, i := range infs {
		addrs, err := i.Addrs()
		check("addr err: ", err)
		for _, a := range addrs {
			ip := strings.Split(a.String(), "/")[0] // 192.168.15.10/24 => 192.168.15.10
			name := i.Name
			// filter off the local and 169.254...
			if strings.Contains(ip, ".") && !strings.Contains(ip, "169.254.") && !strings.Contains(ip, "127.0.0.1") {
				interfaceMap[ip] = networkOpt{NameUI: name, NameSys: devNameMap[ip], IP: ip, HwAddress: i.HardwareAddr}
			}
		}
	}
}

func listenSignalAndSend(parent context.Context) {
	for {
		select {
		case <-parent.Done():
			fmt.Println("listen return")
			return
		case port := <-StartTx:
			txInterface.send(port)
			break
		case port := <-StartRx:
			rxInterface.send(port)
			break
		}
	}
}

// StartTrafficTest :
func StartTrafficTest(srcIP, dstIP string) {
	fmt.Println("traffic start")
	updateInterfacesOpts()

	// Create instance
	txInterface = new(infInstance)
	rxInterface = new(infInstance)

	trafficCtrl, stopTrafficTest := context.WithCancel(context.Background())
	defer func() {
		stopTrafficTest()
		txInterface.close()
		rxInterface.close()
	}()

	go txInterface.init(trafficCtrl, srcIP, dstIP, "tx")
	go rxInterface.init(trafficCtrl, dstIP, srcIP, "rx")

	go listenSignalAndSend(trafficCtrl)

	time.Sleep(5 * time.Second)

	// result := DiagResultVer2{State: stateUpdate, Type: typeBasicPort}

	if txInterface.counter == packetAmount && rxInterface.counter == packetAmount {
		fmt.Println("passed", txInterface.counter)
		fmt.Println("passed", rxInterface.counter)
	} else {
		fmt.Println("failsed", txInterface.counter)
		fmt.Println("failed", rxInterface.counter)
	}
}

func main() {

	from := flag.String("from", "none", "source interface IP address")
	to := flag.String("to", "none", "destination interface IP address")
	flag.Parse()

	go func() {
		go func() {
			time.Sleep(time.Second)
			StartRx <- 1
		}()

		go func() {
			time.Sleep(time.Second)
			StartTx <- 1
		}()

		StartTrafficTest(*from, *to)
	}()

	go func() {
		time.Sleep(10 * time.Second)
		go func() {
			time.Sleep(time.Second)
			StartRx <- 2
		}()

		go func() {
			time.Sleep(time.Second)
			StartTx <- 2
		}()

		StartTrafficTest(*from, *to)

	}()

	stop := make(chan bool)
	stop <- true
}
