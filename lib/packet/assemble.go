package packet

import (
	"github.com/adrian-bl/rna/lib/constants"
)

// Assemble returns binary payload for a ParsedPacket
func Assemble(p *ParsedPacket) []byte {
	buf := make([]byte, 0)

	for _, q := range p.Questions {
		buf = append(buf, assembleQuestion(q)...)
	}
	p.Header.QuestionCount = uint16(len(p.Questions))

	for _, rr := range p.Answers {
		buf = append(buf, assembleResourceRecord(rr)...)
	}
	p.Header.AnswerCount = uint16(len(p.Answers))

	for _, rr := range p.Nameservers {
		buf = append(buf, assembleResourceRecord(rr)...)
	}
	p.Header.NameserverCount = uint16(len(p.Nameservers))

	for _, rr := range p.Additionals {
		buf = append(buf, assembleResourceRecord(rr)...)
	}
	p.Header.AdditionalCount = uint16(len(p.Additionals))

	payload := assembleHeader(p.Header)
	payload = append(payload, buf...)

	return payload
}

// Returns on-wire representation of an uint32
func getU32Int(v uint32) []byte {
	b := make([]byte, 4)
	setU16Int(b[0:], uint16(v>>16))
	setU16Int(b[2:], uint16(v&0xFFFF))
	return b
}

// Returns on-wire representation of an uint16
func getU16Int(v uint16) []byte {
	b := make([]byte, 2)
	setU16Int(b[0:], v)
	return b
}

// Handy function to set a []byte to an uint16 value
func setU16Int(b []byte, val uint16) {
	b[0] = byte(val >> 8)
	b[1] = byte(val & 0xFF)
}

// Handy function to set a bit (aka flag) in a byte
func setFlag(b *byte, condition bool, val byte) {
	if condition == true {
		*b |= val
	}
}

// assembleQuestion transformas a QuestionFormat struct into
// the on-wire format
func assembleQuestion(q QuestionFormat) []byte {
	buf := make([]byte, 0)
	buf = append(buf, EncodeName(q.Name)...)
	buf = append(buf, getU16Int(q.Type)...)
	buf = append(buf, getU16Int(q.Class)...)
	return buf
}

// assembleResourceRecord transforms a ResourceRecordFormat struct
// into the on-wire format
func assembleResourceRecord(rr ResourceRecordFormat) []byte {
	buf := make([]byte, 0)
	buf = append(buf, EncodeName(rr.Name)...)
	buf = append(buf, getU16Int(rr.Type)...)
	buf = append(buf, getU16Int(rr.Class)...)
	buf = append(buf, getU32Int(rr.Ttl)...)
	buf = append(buf, getU16Int(uint16(len(rr.Data)))...)
	buf = append(buf, rr.Data...)
	return buf
}

// assembleHeader returns the on-wire format
// of given ParsedPacketHeader
func assembleHeader(h ParsedPacketHeader) []byte {
	buf := make([]byte, constants.FIX_SIZE_HEADER)

	setU16Int(buf[0:], h.Id) // the 16 bit ID of this query

	// set various flags in the header
	setFlag(&buf[2], h.Response, 1<<7)
	buf[2] |= (h.Opcode & 0xF) << 3
	setFlag(&buf[2], h.Authoritative, 1<<2)
	setFlag(&buf[2], h.Truncated, 1<<1)
	setFlag(&buf[2], h.RecDesired, 1<<0)

	setFlag(&buf[3], h.RecAvailable, 1<<7)
	buf[3] |= (h.ResponseCode) & 0xF

	// and append information about how many items to expect in the 'body'
	setU16Int(buf[4:], h.QuestionCount)
	setU16Int(buf[6:], h.AnswerCount)
	setU16Int(buf[8:], h.NameserverCount)
	setU16Int(buf[10:], h.AdditionalCount)
	return buf
}

// EncodeName encodes a namelabel into the
// DNS on-wire format: <strlen1><rawstr1><strlen2><rawstr2>
func EncodeName(n Namelabel) (payload []byte) {
	for _, str := range n.name {
		payload = append(payload, uint8(len(str)))
		payload = append(payload, str...)
	}
	return
}
