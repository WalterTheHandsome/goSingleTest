package main

import (
	"fmt"
	"net/rpc"
	"runtime/debug"
)

var (
	clientEx                *rpc.Client
	err                     error
	exIP                    = "192.168.17.147"
	typeLED1gOn             = "led1gOn"
	typeLED1gOff            = "led1gOff"
	typeLED10gOn            = "led10gOn"
	typeLED10gOff           = "led10gOff"
	typeLED100gOn           = "led100gOn"
	typeLED100gOff          = "led100gOff"
	typeLEDSWPartnerOn      = "ledSWPartnerOn"
	typeLEDSWPartnerOff     = "ledSWPartnerOff"
	typeLEDSWPartnerDisable = "ledSWPartnerDisable"
	typeLEDSWStatusOn       = "ledSWStatusOn"
	typeLEDSWStatusOff      = "ledSWStatusOff"
)

func connect() *rpc.Client {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("AGX5000 connection recover", r, string(debug.Stack()))
		}
	}()

	client, err := rpc.Dial("tcp", exIP+":1337")
	if err != nil {
		fmt.Println(err)
		return nil
	}

	fmt.Println("AGX5000 connected : "+exIP, client)
	return client
}

func main() {
	clientEx = connect()
	if clientEx == nil {
		return
	}

	// APIDiagnostic.GetSysInfo
	var reply int
	var args map[string]string
	err := clientEx.Call("APIDiagnostic.GetSysInfo", &reply, &args)
	if err != nil {
		fmt.Println("call err =", err)
	}
	fmt.Println("args ", args)
	fmt.Println("reply", reply)

	// // APIDiagnostic.SetLEDTest
	// var reply int
	// var args = typeLED10gOff
	// err := clientEx.Call("APIDiagnostic.ForceTestLED", &args, &reply)
	// if err != nil {
	// 	fmt.Println("call err =", err)
	// }
	// fmt.Println("args ", args)
	// fmt.Println("reply", reply)

	// // APIDiagnostic.SetPortTest
	// var reply int
	// var args = 2
	// err := clientEx.Call("APIDiagnostic.SetPortTest", &args, &reply)
	// if err != nil {
	// 	fmt.Println("call err =", err)
	// }
	// fmt.Println("args ", args)
	// fmt.Println("reply", reply)

	// // APIDiagnostic.GetMcuFanStatus
	// var reply int
	// var args []api.McuFan
	// err := clientEx.Call("APIDiagnostic.GetMcuFanStatus", &reply, &args)
	// if err != nil {
	// 	fmt.Println("call err =", err)
	// }
	// fmt.Println("args ", args)
	// fmt.Println("reply", reply)

	// // APIDiagnostic.GetMcuPsuStatus
	// var reply int
	// var args []api.McuPsu
	// err := clientEx.Call("APIDiagnostic.GetMcuPsuStatus", &reply, &args)
	// if err != nil {
	// 	fmt.Println("call err =", err)
	// }
	// fmt.Println("args ", args)
	// fmt.Println("reply", reply)
}
