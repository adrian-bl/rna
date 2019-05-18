package queue

import (
	"github.com/adrian-bl/rna/lib/constants"
	l "github.com/adrian-bl/rna/lib/log"
	"github.com/adrian-bl/rna/lib/packet"
	"net"
)

func (cq *Cq) newServerReader() (*net.UDPConn, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{})
	if err != nil {
		return nil, err
	}

	go func() {
		buf := make([]byte, constants.MAX_SIZE_UDP)
		for {
			nread, remoteAddr, err := conn.ReadFromUDP(buf)
			if err != nil || nread == 0 {
				l.Debug("Shutdown due to closed sock with err %v", err)
				break
			}
			if nread < constants.FIX_SIZE_HEADER {
				l.Debug("Short read: %d\n", nread)
				continue
			}

			p, err := packet.Parse(buf[0:nread])
			if err != nil {
				l.Debug("%v failed to parse datagram, err=%v", remoteAddr, err)
				continue
			}
			if p.Header.Response == true && p.Header.Opcode == constants.OP_QUERY {
				cq.cache.Put(p, remoteAddr)
			} else {
				l.Debug("??? %v dropped strange packet", remoteAddr)
			}
		}
	}()

	return conn, nil
}
