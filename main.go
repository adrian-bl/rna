package main

import (
	"net"
	"rna/cache"
	"rna/constants"
	l "rna/log"
	"rna/packet"
	"rna/queue"
)

func main() {
	l.Info("Starting up")

	listenAddr, err := net.ResolveUDPAddr("udp", ":53")
	if err != nil {
		l.Panic("ResolveUDPAddr failed: %v", err)
	}

	conn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		l.Panic("listen failed: %v", err)
	}

	buf := make([]byte, constants.MAX_SIZE_UDP) // Upper limit as defined by RFC 1035 2.3.4
	nc := cache.NewNameCache()

	cq := queue.NewClientQueue(conn, nc)

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
		} else if p.Header.Response == true && p.Header.Opcode == constants.OP_QUERY {
			// A reply, try to put it into our cache
			nc.Put(p)
		} else {
			// DOES NOT COMPUTE.
			l.Info("%v dropped packet", remoteAddr)
		}

	}
}
