package main

/*
#include <stdint.h>

extern int mainccc(int argc, char **argv);
*/
import "C"
import (
	"encoding/hex"
	"fmt"
	"gopp"
	"log"
	"net"
	"strings"
	"time"
	"unsafe"
)

var err error

func main() {
	go chkNodeProc()
	go enqueNodeProc()

	for {
		C.mainccc(0, nil)
		log.Println("Exit and restart...")
		if true {
			return
		}
		time.Sleep(5 * time.Second)
	}
}

//export dump_node_handler
func dump_node_handler(cipstr *C.char, cport C.uint16_t, cpubkeyb *C.uint8_t) {
	ipstr := C.GoString(cipstr)
	pubkeyb := C.GoBytes(unsafe.Pointer(cpubkeyb), 32)
	pubkeyh := strings.ToUpper(hex.EncodeToString(pubkeyb))
	port := uint16(cport)
	port = (port&0x00FF)<<8 | (port&0xFF00)>>8 // simple ntohs

	dump_node_handler_impl(ipstr, port, pubkeyh)
}

func dump_node_handler_impl(ipstr string, port uint16, pubkey string) {
	if false {
		log.Println("node:", ipstr, port, len(pubkey), pubkey)
	}
	err = putDHT(pubkey, ipstr, port)
	if err != nil && !strings.HasPrefix(err.Error(), "UNIQUE constraint failed:") {
		gopp.ErrPrint(err)
	}

	nodeQueueRt <- dhtnode{ipstr, port, pubkey}
}

type dhtnode struct {
	ipstr  string
	port   uint16
	pubkey string
}

var nodeQueueRt = make(chan dhtnode, 300)
var nodeQueueBg = make(chan dhtnode, 300000)

// should block
func enqueNodeProc() {
	for node := range nodeQueueRt {
		nodeQueueBg <- node
	}
	log.Println("done")
}

// should block
func chkNodeProc() {
	for i := 0; i < 16; i++ {
		go func(idx int) {
			for node := range nodeQueueBg {
				chkNodeTcpProcImpl(node, idx)
				// chkNodeUdpProcImpl(node, idx)
			}
		}(i)
	}
	if true {
		select {}
	}
	log.Println("done")
}

func chkNodeTcpProcImpl(node dhtnode, idx int) {
	tov := 5 * time.Second
	log.Println("tcp checking...", idx, node.ipstr, node.port, "left:", len(nodeQueueBg))
	c, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", node.ipstr, node.port), tov)
	if err == nil {
		c.Close()
		log.Printf("%+v, tcp opened:\n", node)
		updateDHT(node.pubkey, TCP_STATE_OPEN)
	} else {
		switch {
		case strings.Contains(err.Error(), "connection refused"):
			updateDHT(node.pubkey, TCP_STATE_REFUSE)
		case strings.Contains(err.Error(), "i/o timeout"):
			updateDHT(node.pubkey, TCP_STATE_TIMEOUT)
		case strings.Contains(err.Error(), "no route to host"):
			updateDHT(node.pubkey, TCP_STATE_NOROUTE)
		case strings.Contains(err.Error(), "network is unreachable"):
			updateDHT(node.pubkey, TCP_STATE_UNREACH)
		default:
			gopp.ErrPrint(err, node)
		}
	}
}

func chkNodeUdpProcImpl(node dhtnode, idx int) {
	log.Println("udp checking...", idx, node.ipstr, node.port, "left:", len(nodeQueueBg))
	// nc -vzu 145.178.118.123 33444
	res, err := gopp.RunCmdCout("nc", "-vzu", node.ipstr, fmt.Sprintf("%d", node.port))
	if err != nil && strings.Contains(err.Error(), "exit status 1") {
		log.Println("why udp port can not connect:", node)
	} else {
		gopp.ErrPrint(err, node)
	}
	if err == nil {
		res = strings.TrimSpace(res)
		// log.Println(res)
	}
}
