package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"syscall"

	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
)

type networkOpt struct {
	name      string
	IP        string
	hwAddress net.HardwareAddr
}

var _ net.Addr = &Addr{}

// Addr is an implement for raw.Addr
type Addr struct {
	HardwareAddr net.HardwareAddr
}

// Network returns the address's network name, "raw".
func (a *Addr) Network() string {
	return "raw"
}

// String returns the address's hardware address.
func (a *Addr) String() string {
	return a.HardwareAddr.String()
}

const (
	defaultPort = ":9999"
	etherType   = 0xcccc
)

var (
	txCon *net.IPConn
	rxCon *net.IPConn
	IPMap = map[string]networkOpt{}
)

func check(msg string, err error) {
	if err != nil {
		fmt.Println(msg)
		log.Fatal(err)
	}
}

// for darwin / win32
// mac or ipv6 will be the form of "fe80:3c4e:...."
// so use "." to identify
func getInterfaces() []string {
	IPMap = map[string]networkOpt{}
	result := []string{}
	infs, err := net.Interfaces()
	check("inf", err)
	for _, i := range infs {
		fmt.Println(i.HardwareAddr)

		addrs, err := i.Addrs()
		check("addr", err)
		for _, a := range addrs {
			if strings.Contains(a.String(), ".") && !strings.Contains(a.String(), "169.254.") && !strings.Contains(a.String(), "127.0.0.1") {
				name := i.Name
				ip := strings.Split(a.String(), "/")[0]
				result = append(result, name+ip)
				IPMap[name+"-"+ip] = networkOpt{name: name, IP: ip, hwAddress: i.HardwareAddr}
			}
		}
	}
	return result
}

func start(ifName string) {
	// Open a raw socket on the specified interface, and configure it to accept
	// traffic with etherecho's EtherType.
	fmt.Println("ifname", ifName)
	ifi, err := net.InterfaceByName(ifName)
	if err != nil {
		log.Fatalf("failed to find interface %q: %v", ifName, err)
	}

	c, err := raw.ListenPacket(ifi, etherType, nil)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Default message to system's hostname if empty.
	msg := "hello world"
	if msg == "" {
		msg, err = os.Hostname()
		if err != nil {
			log.Fatalf("failed to retrieve hostname: %v", err)
		}
	}

	// Send messages in one goroutine, receive messages in another.
	go sendMessages(ifi, msg)
	go receiveMessages(c, ifi.MTU)

}

func sendMessages(source net.HardwareAddr, msg string) {
	// // Message is broadcast to all machines in same network segment.
	f := &ethernet.Frame{
		Destination: ethernet.Broadcast,
		Source:      source,
		EtherType:   etherType,
		Payload:     []byte(msg),
	}

	b, err := f.MarshalBinary()
	if err != nil {
		log.Fatalf("failed to marshal ethernet frame: %v", err)
	}

	// // Required by Linux, even though the Ethernet frame has a destination.
	// // Unused by BSD.
	// addr := &Addr{
	// 	HardwareAddr: ethernet.Broadcast,
	// }

	// // Send message forever.
	// t := time.NewTicker(1 * time.Second)
	// for range t.C {
	// 	if _, err := c.WriteTo(b, addr); err != nil {
	// 		log.Fatalf("failed to send message: %v", err)
	// 	}
	// }
	fmt.Println("source", source)
	fmt.Println("msg", msg)
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, syscall.ETH_P_ALL)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		return
	}
	fmt.Println("Obtained fd ", fd)
	defer syscall.Close(fd)

}

// receiveMessages continuously receives messages over a connection. The messages
// may be up to the interface's MTU in size.
func receiveMessages(c net.PacketConn, mtu int) {
	var f ethernet.Frame
	b := make([]byte, mtu)

	// Keep receiving messages forever.
	for {
		n, addr, err := c.ReadFrom(b)
		if err != nil {
			log.Fatalf("failed to receive message: %v", err)
		}

		// Unpack Ethernet II frame into Go representation.
		if err := (&f).UnmarshalBinary(b[:n]); err != nil {
			log.Fatalf("failed to unmarshal ethernet frame: %v", err)
		}

		// Display source of message and message itself.
		log.Printf("[%s] %s", addr.String(), string(f.Payload))
	}
}

func main() {
	opts := getInterfaces()
	fmt.Println(opts)
	fmt.Println(IPMap)

	start(IPMap["en7-192.168.15.10"].name)

	start(IPMap["en8-192.168.15.20"].name)

	stop := make(chan bool)
	stop <- true
}
