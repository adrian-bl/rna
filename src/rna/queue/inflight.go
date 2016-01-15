package queue

import (
	"fmt"
	"net"
	"rna/cache"
	"rna/packet"
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
	conn     *net.UDPConn
	cache    *cache.Cache
	inflight map[string][]chan bool
}

func NewClientQueue(conn *net.UDPConn, cache *cache.Cache) *Cq {
	cq := &Cq{conn: conn, cache: cache, inflight: make(map[string][]chan bool, 0)}
	cache.RegisterPutCallback(cq.handlePutCallback)
	return cq
}

func (cq *Cq) blockForQuery(pp *packet.ParsedPacket) bool {
	cbi := &putCbItem{Key: pp.Questions[0].Name.ToKey(), Type: pp.Questions[0].Type}
	key := cbi.ToString()

	c := make(chan bool)
	if cq.inflight[key] == nil {
		cq.inflight[key] = make([]chan bool, 0)
	}
	cq.inflight[key] = append(cq.inflight[key], c)
	fmt.Printf("Blocking for progress on %s\n", key)

	select {
	case <-c:
		fmt.Printf("%s made progress\n", key)
		return true
	case <-time.After(time.Second * 2):
		fmt.Printf("%s TIMED OUT!\n", key)
		return false
	}
}

func (cq *Cq) handlePutCallback(isrc *cache.InjectSource) {
	cbi := &putCbItem{Key: isrc.Name.ToKey(), Type: isrc.Type}
	key := cbi.ToString()

	if cq.inflight[key] != nil {
		for _, c := range cq.inflight[key] {
			fmt.Printf("Notify about progress on %s\n", key)
			c <- true
		}
		cq.inflight[key] = nil
	}
}
