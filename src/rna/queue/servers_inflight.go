package queue

import (
	"fmt"
	"net"
	"rna/cache"
	"rna/packet"
	"sync"
)

type Sq struct {
	sync.Mutex
	q [200]string // keep up to 200 outstanding replies
	c int         // cursor
}

func NewServerQueue(nc *cache.Cache) *Sq {
	sq := &Sq{}
	nc.RegisterVeritfyCallback(sq.handleVerifyCallback)
	return sq
}

func (sq *Sq) registerQuery(q packet.QuestionFormat, ns *net.UDPAddr) {
	sq.Lock()
	sq.q[sq.c] = sq.toKey(q, ns)
	sq.Unlock()
	sq.c++
	if sq.c == len(sq.q) {
		sq.c = 0
	}
}

func (sq *Sq) handleVerifyCallback(q packet.QuestionFormat, ns *net.UDPAddr) bool {
	key := sq.toKey(q, ns)
	sq.Lock()
	defer sq.Unlock()
	for i, k := range sq.q {
		if k == key {
			sq.q[i] = ""
			return true
		}
	}
	return false
}

func (sq *Sq) toKey(q packet.QuestionFormat, ns *net.UDPAddr) string {
	return fmt.Sprintf("ns=%s, q=%s, t=%d, c=%d ", ns, q.Name.ToKey(), q.Type, q.Class)
}
