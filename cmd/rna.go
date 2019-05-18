package main

import (
	"flag"
	"fmt"
	"github.com/adrian-bl/rna/lib/cache"
	"github.com/adrian-bl/rna/lib/constants"
	l "github.com/adrian-bl/rna/lib/log"
	"github.com/adrian-bl/rna/lib/packet"
	"github.com/adrian-bl/rna/lib/queue"
	"net"
)

var listenPort = flag.Int("port", 53, "Bind to this port, defaults to 53")

func main() {
	flag.Parse()

	listenStr := fmt.Sprintf(":%d", *listenPort)
	l.Info("Starting up, listening on %s", listenStr)

	listenAddr, err := net.ResolveUDPAddr("udp", listenStr)
	if err != nil {
		l.Panic("ResolveUDPAddr failed: %v", err)
	}
	rconn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		l.Panic("listen failed: %v", err)
	}

	nc := cache.NewNameCache()
	sq := queue.NewServerQueue(nc)
	cq := queue.NewClientQueue(rconn, nc, sq)
	readClient(cq, rconn)
}

func readClient(cq *queue.Cq, conn *net.UDPConn) {
	buf := make([]byte, constants.MAX_SIZE_UDP) // Upper limit as defined by RFC 1035 2.3.4
	for {
		nread, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil || nread < constants.FIX_SIZE_HEADER {
			l.Debug("%v dropping malformed datagram. Size=%d, err=%v", remoteAddr, nread, err)
			continue
		}

		p, err := packet.Parse(buf[0:nread])
		if err != nil {
			l.Debug("%v failed to parse datagram, err=%v", remoteAddr, err)
			continue
		}

		if p.Header.Response == false && p.Header.Opcode == constants.OP_QUERY && p.Header.RecDesired {
			// This is a query, requesting recursion
			cq.AddClientRequest(p, remoteAddr)
		} else {
			// DOES NOT COMPUTE.
			l.Info("!!! %v dropped packet", remoteAddr)
		}

	}
}
