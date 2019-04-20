package queue

import (
	"context"
	"fmt"
	"github.com/adrian-bl/rna/lib/cache"
	"github.com/adrian-bl/rna/lib/constants"
	l "github.com/adrian-bl/rna/lib/log"
	"github.com/adrian-bl/rna/lib/packet"
	"math/rand"
	"net"
	"time"
)

// A (parsed) request sent by a client
type clientRequest struct {
	Query      *packet.ParsedPacket
	RemoteAddr *net.UDPAddr
}

// The result of a lookup operation
type lookupRes struct {
	cres   *cache.CacheResult
	status int
}

const (
	LR_POSITIVE = 0
	LR_NEGATIVE = 1
	LR_TIMEOUT  = 2
)

type qCtx struct {
	context context.Context
	cancel  context.CancelFunc
}

// Starts the lookup of a new client request
func (cq *Cq) AddClientRequest(query *packet.ParsedPacket, remote *net.UDPAddr) {
	d := time.Now().Add(1250 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	qctx := &qCtx{context: ctx, cancel: cancel}
	go cq.clientLookup(&clientRequest{Query: query, RemoteAddr: remote}, qctx)
}

func (cq *Cq) clientLookup(cr *clientRequest, qctx *qCtx) {
	// Ensure that this query makes some sense
	if len(cr.Query.Questions) == 1 {
		q := cr.Query.Questions[0]
		c := make(chan *lookupRes)
		go cq.collapsedLookup(q, c, qctx)
		lres := <-c
		qctx.cancel()

		l.Debug("final lookup reply -> %v", lres)
		if lres != nil { // fixme: error
			cres := lres.cres
			p := &packet.ParsedPacket{}
			p.Header.Id = cr.Query.Header.Id
			p.Header.Response = true
			p.Header.ResponseCode = cres.ResponseCode
			p.Questions = cr.Query.Questions
			switch lres.status {
			case LR_POSITIVE:
				p.Answers = append(p.Answers, cres.ResourceRecord...)
			case LR_NEGATIVE:
				p.Nameservers = append(p.Nameservers, cres.ResourceRecord...)
			default:
				// nil
			}
			cq.rconn.WriteToUDP(packet.Assemble(p), cr.RemoteAddr)
		} else {
			l.Info("Lookup returned an error, should send it back to client (fixme): %+v", lres)
		}
	} else {
		l.Info("Dropping nonsense query")
	}
}

// Our shiny lookup loop
func (cq *Cq) collapsedLookup(q packet.QuestionFormat, c chan *lookupRes, qctx *qCtx) {

	for i := 0; i < 5; {
		if qctx.context.Err() != nil {
			c <- &lookupRes{&cache.CacheResult{}, LR_TIMEOUT}
			break
		}

		cres, cerr := cq.cache.Lookup(q.Name, q.Type)
		if cres != nil {
			c <- &lookupRes{cres, LR_POSITIVE}
			break
		}
		if cerr != nil {
			c <- &lookupRes{cerr, LR_NEGATIVE}
			break
		}

		// No such type exists in cache, but maybe we got a CNAME... horray.
		cres, _ = cq.cache.Lookup(q.Name, constants.TYPE_CNAME)
		if cres != nil {
			if len(cres.ResourceRecord) == 1 {
				// We do not support weird multi-record cnames
				target_label, err := packet.ParseName(cres.ResourceRecord[0].Data)
				if err == nil {
					// Restart query with cname label but inherit types of original query.
					target_chan := make(chan *lookupRes)
					target_q := packet.QuestionFormat{Name: target_label, Type: q.Type, Class: q.Class}
					go cq.collapsedLookup(target_q, target_chan, qctx)
					target_res := <-target_chan
					if target_res != nil {
						// not a dead cname: we got the requested record -> append it to original cache reply
						cres.ResourceRecord = append(cres.ResourceRecord, target_res.cres.ResourceRecord...)
					}
					c <- &lookupRes{cres, LR_POSITIVE}
				}
			}
			break
		}

		pp := cq.advanceCache(q, qctx)
		progress := cq.blockForQuery(pp, qctx)
		if progress == false {
			i++
		}
	}
	// return pseudio-nil if we give up
	close(c)
}

func (cq *Cq) advanceCache(q packet.QuestionFormat, qctx *qCtx) *packet.ParsedPacket {
	// our hardcoded, not so redundant slist
	targetNS := "192.5.5.241:53"
	targetXH := &packet.Namelabel{}
	targetQT := q.Type

POP_LOOP:
	for i := 0; ; i++ {
		label := q.Name.PoppedLabel(i) // removes 'i' labels from the label list
		nsrec, _ := cq.cache.Lookup(*label, constants.TYPE_NS)

		if nsrec != nil { // we got an NS cache entry for this level
			var candidate_cres *cache.CacheResult
			var candidate_label packet.Namelabel

			// loop trough all NS servers for this record
			for _, candidate_data := range nsrec.ResourceRecord {
				name, err := packet.ParseName(candidate_data.Data)
				if err == nil {
					l.Debug("NS %v handles %v", name, label)
					cres, _ := cq.cache.Lookup(name, constants.TYPE_A)
					if cres != nil {
						candidate_cres = cres
					} else {
						candidate_label = name
					}
				}
			}

			// Fixme: We should try to resolve (yet unknown) nameservers
			// even if we got a candidate_res as the one we are contacting
			// might fail for some reason.

			if candidate_cres == nil && candidate_label.Len() > 0 {
				l.Debug("Looking up IP of known candidate: %v", candidate_label)
				c := make(chan *lookupRes)
				go cq.collapsedLookup(packet.QuestionFormat{Type: constants.TYPE_A, Class: constants.CLASS_IN, Name: candidate_label}, c, qctx)
				lres := <-c
				if lres != nil && lres.status == LR_POSITIVE {
					candidate_cres = lres.cres
				}
			}
			if candidate_cres != nil {
				l.Debug("We got an RR: %v", candidate_cres)
				for _, v := range candidate_cres.ResourceRecord {
					if v.Type != constants.TYPE_A {
						l.Panic("Not an A type: %v", v)
					}
					targetNS = fmt.Sprintf("%d.%d.%d.%d:53", v.Data[0], v.Data[1], v.Data[2], v.Data[3])
					targetXH = label
					break POP_LOOP
				}
			}
		}
		if label.Len() == 1 {
			break
		}
	}

	pp := &packet.ParsedPacket{}
	pp.Header.Id = uint16(rand.Uint32()) // will simply overflow
	pp.Header.Opcode = constants.OP_QUERY
	pp.Header.QuestionCount = 1
	pp.Questions = []packet.QuestionFormat{{Name: *q.Name.ShuffleCases(), Class: constants.CLASS_IN, Type: targetQT}}
	remoteNs, err := net.ResolveUDPAddr("udp", targetNS)

	if err == nil {
		l.Info("+ op=query, remote=%s, type=%d, id=%d, name=%v", targetNS, targetQT, pp.Header.Id, pp.Questions[0].Name)
		cq.sconn.WriteToUDP(packet.Assemble(pp), remoteNs)
		cq.sq.registerQuery(pp.Questions[0], remoteNs, targetXH)
	}

	return pp
}
