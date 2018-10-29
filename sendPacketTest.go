package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/dmichael/go-multicast/multicast"
)

type networkOpt struct {
	name        string
	IP          string
	multicastIP []string
}

const (
	defaultPort = ":9999"
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
		mulAddr, _ := i.MulticastAddrs()
		fmt.Println("inf", mulAddr)

		addrs, err := i.Addrs()
		check("addr", err)
		for _, a := range addrs {
			if strings.Contains(a.String(), ".") && !strings.Contains(a.String(), "169.254.") && !strings.Contains(a.String(), "127.0.0.1") {
				name := i.Name
				ip := strings.Split(a.String(), "/")[0]
				result = append(result, name+ip)
				muls := []string{}
				for _, m := range mulAddr {
					if strings.Contains(m.String(), ".") {
						fmt.Println(net.ParseIP(m.String()).IsLinkLocalMulticast())
						muls = append(muls, m.String())
					}
				}
				IPMap[name+"-"+ip] = networkOpt{name: name, IP: ip, multicastIP: muls}
			}
		}
	}
	return result
}

func startPinger(addr string) {
	ping(addr)
}

func ping(addr string) {
	conn, err := multicast.NewBroadcaster(addr)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			conn.Write([]byte("walter hahaha\n"))
			time.Sleep(1 * time.Second)
		}
	}()
}

func msgHandler(src *net.UDPAddr, n int, b []byte) {
	log.Println(n, "bytes read from", src)
	log.Println(string(b[:n]))
}

func startListener(addr string) {
	go multicast.Listen(addr, msgHandler)
}

func main() {
	opts := getInterfaces()
	fmt.Println(opts)
	fmt.Println(IPMap)

	go func() {
		startPinger(IPMap["en7-192.168.15.10"].multicastIP[0] + defaultPort)
	}()

	go func() {
		startListener(IPMap["en8-192.168.16.20"].multicastIP[0] + defaultPort)
	}()

	stop := make(chan bool)
	stop <- true
}
