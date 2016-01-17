package cache

import (
	"fmt"
	"rna/constants"
	l "rna/log"
	"rna/packet"
	"sync"
	"time"
)

// CacheResult is a struct returned by the Lookup function
type CacheResult struct {
	ResourceRecord []packet.ResourceRecordFormat
	ResponseCode   uint8
}

// citem stores a cache item
type citem struct {
	data     []byte
	deadline time.Time
}
type centry map[string]citem
type cmap map[uint16]centry

// mitem stores a miss item
type mitem struct {
	name packet.Namelabel
	data []byte
	rc   uint8
}
type mmap map[uint16]*mitem

type Cache struct {
	sync.RWMutex
	CacheMap map[string]cmap
	MissMap  map[string]mmap
	Callback func(*InjectSource)
}

type InjectSource struct {
	Name packet.Namelabel
	Type uint16
}

// NewNameCache returns a newly initialized cache reference
func NewNameCache() *Cache {
	c := &Cache{}
	c.CacheMap = make(map[string]cmap, 0)
	c.MissMap = make(map[string]mmap, 0)
	return c
}

// Registers a function to be called on cache inserts
func (c *Cache) RegisterPutCallback(cb func(*InjectSource)) {
	c.Callback = cb
}

// Puts given entry into c's Cache
func (c *Cache) Put(p *packet.ParsedPacket) {
	// FIXME: SHOULD CHECK IF SENDER IS PERMITTED TO TELL US ABOUT THIS!

	var qname packet.Namelabel
	var qtype uint16
	for _, q := range p.Questions {
		l.Debug("QNAME=%v", q.Name)
		qname = q.Name
		qtype = q.Type
	}

	isrc := &InjectSource{Name: qname, Type: qtype}

	if p.Header.Authoritative == true {
		for _, n := range p.Answers {
			if n.Class == constants.CLASS_IN {
				c.inject(isrc, n)
			}
		}
	}

	for _, n := range p.Additionals {
		if n.Class == constants.CLASS_IN {
			c.inject(isrc, n)
		}
	}

	for _, n := range p.Nameservers {
		if n.Type == constants.TYPE_NS && n.Class == constants.CLASS_IN {
			c.inject(isrc, n)
		}
		if p.Header.AnswerCount == 0 && n.Type == constants.TYPE_SOA && n.Class == constants.CLASS_IN {
			c.injectNegativeItem(isrc, p.Header.ResponseCode, n)
		}
	}
}

// Lookup returns the CacheResult of given Namelabel and Type combination
// rr will be nil if there was no positive match
// re will be nil if there was no negative match
// rr == re == nil if the entry is completely unknown
func (c *Cache) Lookup(l packet.Namelabel, t uint16) (rr *CacheResult, re *CacheResult) {
	key := l.ToKey()
	now := time.Now()

	c.Lock()
	defer c.Unlock()

	if c.CacheMap[key] != nil {
		if c.CacheMap[key][t] != nil {
			ent := make([]packet.ResourceRecordFormat, 0)
			for _, item := range c.CacheMap[key][t] {
				if now.Before(item.deadline) {
					ttl := uint32(item.deadline.Sub(now).Seconds())
					ent = append(ent, packet.ResourceRecordFormat{Name: l, Class: constants.CLASS_IN, Type: t, Ttl: ttl, Data: item.data})
				}
			}
			if len(ent) > 0 { // ensure to return a null pointer if ent is empty
				rr = &CacheResult{ResourceRecord: ent, ResponseCode: constants.RC_NO_ERR}
			}
		}
	}

	// rr will be nil on cache miss, check if we have a negative cache entry
	if rr == nil && c.MissMap[key] != nil {
		mtype := t
		if c.MissMap[key][constants.TYPE_SOA] != nil {
			mtype = constants.TYPE_SOA // this marks an NX domain
		}
		if c.MissMap[key][mtype] != nil {
			item := c.MissMap[key][mtype]
			ent := make([]packet.ResourceRecordFormat, 0)
			ent = append(ent, packet.ResourceRecordFormat{Name: item.name, Class: constants.CLASS_IN, Type: constants.TYPE_SOA, Ttl: 1234, Data: item.data})
			re = &CacheResult{ResourceRecord: ent, ResponseCode: item.rc}
		}
	}

	return
}

func (c *Cache) dump() {

	for name, tmap := range c.CacheMap {
		for t, centry := range tmap {
			for plx, _ := range centry {
				fmt.Printf("%-21s [%2d] = %+v\n", name, t, plx)
			}
		}
	}

}

func (c *Cache) notify(isrc *InjectSource) {
	if c.Callback != nil {
		c.Callback(isrc)
	}
}

// injectNegativeItem marks given label as non existing. rc defines the return code
// item is supposed to be a SOA
func (c *Cache) injectNegativeItem(isrc *InjectSource, rc uint8, item packet.ResourceRecordFormat) {
	if item.Type != constants.TYPE_SOA {
		l.Panic("can not inject non-soa type: %v", item)
	}
	key := isrc.Name.ToKey()
	mtype := isrc.Type

	c.Lock()
	defer c.Unlock()

	if c.MissMap[key] == nil {
		c.MissMap[key] = make(mmap, 0)
	}
	if rc == constants.RC_NAME_ERR {
		mtype = constants.TYPE_SOA
	}

	target := make([]byte, len(item.Data))
	copy(target, item.Data)
	c.MissMap[key][mtype] = &mitem{rc: rc, data: target, name: item.Name}
	c.notify(isrc)
}

// inject puts given resource record format item into our positive cache
func (c *Cache) inject(isrc *InjectSource, item packet.ResourceRecordFormat) {
	key := item.Name.ToKey()
	t := item.Type
	data := item.Data
	ttl := item.Ttl

	c.Lock()
	defer c.Unlock()

	if c.CacheMap[key] == nil {
		c.CacheMap[key] = make(cmap, 0)
	}
	if c.CacheMap[key][t] == nil {
		c.CacheMap[key][t] = make(centry, 0)
	}

	target := make([]byte, len(data))
	copy(target, data)
	l.Debug("+ cache inject: %v", item)
	c.CacheMap[key][t][string(data)] = citem{data: target, deadline: time.Now().Add(time.Duration(ttl) * time.Second)}
	c.notify(isrc)
}
