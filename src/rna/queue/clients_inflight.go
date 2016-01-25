package queue

import (
	"fmt"
	"net"
	"rna/cache"
	l "rna/log"
	"rna/packet"
	"sync"
	"time"
)

type putCbItem struct {
	Key  string
	Type uint16
}

func (r *putCbItem) ToString() string {
	return fmt.Sprintf("%d->%s", r.Type, r.Key)
}

type Cq struct {
	sync.RWMutex
	conn     *net.UDPConn
	cache    *cache.Cache
	sq       *Sq
	inflight map[string][]chan bool
}

func NewClientQueue(conn *net.UDPConn, cache *cache.Cache, sq *Sq) *Cq {
	cq := &Cq{conn: conn, cache: cache, sq: sq, inflight: make(map[string][]chan bool, 0)}
	cache.RegisterPutCallback(cq.handlePutCallback)
	return cq
}

func (cq *Cq) blockForQuery(pp *packet.ParsedPacket) bool {
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
		return false
	}
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
