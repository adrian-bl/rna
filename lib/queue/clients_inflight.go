package queue

import (
	"github.com/adrian-bl/rna/lib/cache"
	l "github.com/adrian-bl/rna/lib/log"
	"github.com/adrian-bl/rna/lib/packet"
	"net"
	"sync"
	"time"
)

type Cq struct {
	sync.RWMutex
	rconn    *net.UDPConn
	sconn    *net.UDPConn
	cache    *cache.Cache
	sq       *Sq
	inflight map[string][]chan bool
}

func NewClientQueue(rconn *net.UDPConn, sconn *net.UDPConn, cache *cache.Cache, sq *Sq) *Cq {
	cq := &Cq{rconn: rconn, sconn: sconn, cache: cache, sq: sq, inflight: make(map[string][]chan bool, 0)}
	cache.RegisterPutCallback(cq.handlePutCallback)
	return cq
}

func (cq *Cq) blockForQuery(pp *packet.ParsedPacket, qctx *qCtx) bool {
	cbi := &putCbItem{Key: pp.Questions[0].Name.ToKey(), Type: pp.Questions[0].Type}
	key := cbi.ToString()

	cq.Lock()
	c := make(chan bool)
	if cq.inflight[key] == nil {
		cq.inflight[key] = make([]chan bool, 0)
	}
	cq.inflight[key] = append(cq.inflight[key], c)
	cq.Unlock()

	l.Debug("Blocking for progress on %s", key)
	select {
	case <-c:
		l.Debug("%s progressed", key)
		return true
	case <-time.After(time.Second * 2):
		l.Debug("%s timed out!", key)
	case <-qctx.context.Done():
		l.Debug("%s context deadline reached", key)
	}
	return false
}

func (cq *Cq) handlePutCallback(isrc cache.InjectSource) {
	cbi := &putCbItem{Key: isrc.Name.ToKey(), Type: isrc.Type}
	key := cbi.ToString()

	cq.Lock()
	if cq.inflight[key] != nil {
		for _, c := range cq.inflight[key] {
			l.Debug("Broadcasting progress on %s", key)
			close(c)
		}
		cq.inflight[key] = nil
	}
	cq.Unlock()
}
