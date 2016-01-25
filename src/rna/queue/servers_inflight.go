package queue

import (
	"fmt"
	"net"
	"rna/cache"
	"rna/packet"
	"sync"
)

type SqEntry struct {
	key     string
	xhlabel *packet.Namelabel
}

type Sq struct {
	sync.Mutex
	q [200]SqEntry // keep up to 200 outstanding replies
	c int          // cursor
}

func NewServerQueue(nc *cache.Cache) *Sq {
	sq := &Sq{}
	nc.RegisterVeritfyCallback(sq.handleVerifyCallback)
	return sq
}

func (sq *Sq) registerQuery(q packet.QuestionFormat, ns *net.UDPAddr, label *packet.Namelabel) {
	sq.Lock()
	sq.q[sq.c] = SqEntry{key: sq.toKey(q, ns), xhlabel: label}
	sq.Unlock()
	sq.c++
	if sq.c == len(sq.q) {
		sq.c = 0
	}
}

func (sq *Sq) handleVerifyCallback(q packet.QuestionFormat, ns *net.UDPAddr) *packet.Namelabel {
	key := sq.toKey(q, ns)
	sq.Lock()
	defer sq.Unlock()
	for i, e := range sq.q {
		if e.key == key {
			sq.q[i] = SqEntry{}
			return e.xhlabel
		}
	}
	return nil
}

func (sq *Sq) toKey(q packet.QuestionFormat, ns *net.UDPAddr) string {
	return fmt.Sprintf("ns=%s, q=%s, t=%d, c=%d ", ns, q.Name.ToKey(), q.Type, q.Class)
}
