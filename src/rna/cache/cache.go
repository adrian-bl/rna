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
	rcode    uint8
}
type centry map[string]citem
type cmap map[uint16]centry

type Cache struct {
	sync.RWMutex
	CacheMap map[string]cmap
	MissMap  map[string]cmap
	Callback func(InjectSource)
}

type InjectSource struct {
	Name packet.Namelabel
	Type uint16
}

// NewNameCache returns a newly initialized cache reference
func NewNameCache() *Cache {
	c := &Cache{}
	c.CacheMap = make(map[string]cmap, 0)
	c.MissMap = make(map[string]cmap, 0)
	return c
}

// Registers a function to be called on cache inserts
func (c *Cache) RegisterPutCallback(cb func(InjectSource)) {
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

	isrc := InjectSource{Name: qname, Type: qtype}

	if p.Header.Authoritative == true {
		for _, n := range p.Answers {
			if n.Class == constants.CLASS_IN {
				c.injectPositiveItem(isrc, n)
			}
		}
	}

	for _, n := range p.Additionals {
		if n.Class == constants.CLASS_IN {
			c.injectPositiveItem(isrc, n)
		}
	}

	for _, n := range p.Nameservers {
		if n.Type == constants.TYPE_NS && n.Class == constants.CLASS_IN {
			c.injectPositiveItem(isrc, n)
		}
		if p.Header.AnswerCount == 0 && n.Type == constants.TYPE_SOA && n.Class == constants.CLASS_IN {
			c.injectNegativeItem(isrc, n, p.Header.ResponseCode)
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
			// we do have a negative soa entry, so the domain simply does not exist
			// and there is no point in looking up 't'
			mtype = constants.TYPE_SOA
		}
		if c.MissMap[key][mtype] != nil {
			for _, item := range c.MissMap[key][mtype] {
				if now.Before(item.deadline) {
					ttl := uint32(item.deadline.Sub(now).Seconds())
					// unparse fiddled-in soa label
					rend := item.data[0] + 1
					rlabel := item.data[1:rend]
					plabel, _ := packet.ParseName(rlabel)
					ent := make([]packet.ResourceRecordFormat, 0)
					ent = append(ent, packet.ResourceRecordFormat{Name: plabel, Class: constants.CLASS_IN, Type: constants.TYPE_SOA, Ttl: ttl, Data: item.data[rend:]})
					re = &CacheResult{ResourceRecord: ent, ResponseCode: item.rcode}
				}
			}
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

func (c *Cache) notify(isrc InjectSource) {
	if c.Callback != nil {
		c.Callback(isrc)
	}
}

// injectNegativeItem marks given label as non existing. rc defines the return code
// item is supposed to be a SOA
func (c *Cache) injectNegativeItem(isrc InjectSource, item packet.ResourceRecordFormat, rcode uint8) {
	if item.Type != constants.TYPE_SOA {
		panic("Not a SOA!")
	}

	// use SOA.MINTTL as ttl for this negative cache entry
	// this *should* be the same as the TTL set by upstream, but
	// there are some funny DNS servers out there...
	soaTtl := packet.ParseSoaTtl(item.Data)
	item.Ttl = soaTtl
	switch {
		case item.Ttl < 5:
			item.Ttl = 5
		case item.Ttl > 600:
			item.Ttl = 600
	}

	// xxx: The cache key for this entry should not be the response label but the
	// question (isrc) label. However: The response label needs to be preserved
	// so we are prefixing it to the raw data for now (until we have a nicer API)
	rawlabel := packet.EncodeName(item.Name)
	item.Data = append(rawlabel, item.Data...)
	item.Data = append([]byte{byte(len(rawlabel))}, item.Data...)
	item.Name = isrc.Name // use the looked up label as cache key, not the SOA label

	// We shall also use the source TYPE *unless* we are storing an NXDOMAIN entry
	if rcode != constants.RC_NAME_ERR {
		item.Type = isrc.Type
	} else {
		// this was an NXDOMAIN -> a negative SOA entry signals that NO RRs exist
		item.Type = constants.TYPE_SOA
	}

	c.injectInternal(c.MissMap, isrc, item, rcode)
}

// inject puts given resource record format item into our positive cache
func (c *Cache) injectPositiveItem(isrc InjectSource, item packet.ResourceRecordFormat) {
	c.injectInternal(c.CacheMap, isrc, item, 0)
}

// Internal implementation of cache who works on multiple maps
func (c *Cache) injectInternal(m map[string]cmap, isrc InjectSource, item packet.ResourceRecordFormat, rcode uint8) {
	key := item.Name.ToKey()
	t := item.Type
	data := item.Data
	ttl := item.Ttl

	c.Lock()
	defer c.Unlock()

	if m[key] == nil {
		m[key] = make(cmap, 0)
	}
	if m[key][t] == nil {
		m[key][t] = make(centry, 0)
	}

	l.Debug("+ cache inject: %+v", item)

	cpy := make([]byte, len(data))
	copy(cpy, data)
	m[key][t][string(data)] = citem{data: cpy, deadline: time.Now().Add(time.Duration(ttl) * time.Second), rcode: rcode}
	c.notify(isrc)
}
