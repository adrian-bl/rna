package packet

import (
	"fmt"
	"github.com/adrian-bl/rna/lib/constants"
)

func Parse(buf []byte) (p *ParsedPacket, perr error) {
	msglen := len(buf) // not really the received length, but the allocated memory

	if msglen < constants.FIX_SIZE_HEADER {
		perr = fmt.Errorf("Packet too short")
		return
	}

	p = &ParsedPacket{}
	p.Header = ParsedPacketHeader{}

	// Parse the fixed packet header
	h := &p.Header
	h.Id = nUint16(buf[0:])
	h.Response = (buf[2]&(1<<7) != 0)
	h.Opcode = buf[2] >> 3 & 0xF
	h.Authoritative = (buf[2]&(1<<2) != 0)
	h.Truncated = (buf[2]&(1<<1) != 0)
	h.RecDesired = (buf[2]&(1<<0) != 0)
	h.RecAvailable = (buf[3]&(1<<7) != 0)
	h.ResponseCode = uint8(buf[3]) & 0x0F
	h.QuestionCount = nUint16(buf[4:])
	h.AnswerCount = nUint16(buf[6:])
	h.NameserverCount = nUint16(buf[8:])
	h.AdditionalCount = nUint16(buf[10:])

	// Ok, time to parse the variable-sized sections
	// by jumping at the end of the header
	c := constants.FIX_SIZE_HEADER

	// Parse question section
	for i := uint16(0); i < h.QuestionCount && perr == nil; i++ {
		var qf QuestionFormat
		qf, c, perr = parseQuestion(buf, c)
		if perr == nil {
			p.Questions = append(p.Questions, qf)
		}
	}
	// make sure that this matches what we parsed
	h.QuestionCount = uint16(len(p.Questions))

	// Parse answer section
	for i := uint16(0); i < h.AnswerCount && perr == nil; i++ {
		var rr ResourceRecordFormat
		rr, c, perr = parseResourceRecord(buf, c)
		if perr == nil {
			p.Answers = append(p.Answers, rr)
		}
	}
	h.AnswerCount = uint16(len(p.Answers))

	// Parse nameserver section
	for i := uint16(0); i < h.NameserverCount && perr == nil; i++ {
		var rr ResourceRecordFormat
		rr, c, perr = parseResourceRecord(buf, c)
		if perr == nil {
			p.Nameservers = append(p.Nameservers, rr)
		}
	}
	h.NameserverCount = uint16(len(p.Nameservers))

	// Parse additional section
	for i := uint16(0); i < h.AdditionalCount && perr == nil; i++ {
		var rr ResourceRecordFormat
		rr, c, perr = parseResourceRecord(buf, c)
		if perr == nil {
			p.Additionals = append(p.Additionals, rr)
		}
	}
	h.AdditionalCount = uint16(len(p.Additionals))

	return
}

func nUint16(buf []byte) uint16 {
	return uint16(buf[0])<<8 + uint16(buf[1])
}
func nUint32(buf []byte) uint32 {
	return uint32(nUint16(buf[0:]))<<16 + uint32(nUint16(buf[2:]))
}

func parseQuestion(buf []byte, spos int) (qf QuestionFormat, c int, err error) {
	var name Namelabel
	name, c, err = parseName(buf, spos)
	if c+4 > len(buf) { // fixme?
		err = fmt.Errorf("Short datagram")
	}

	if err == nil {
		q_type := nUint16(buf[c:])
		c += 2
		q_class := nUint16(buf[c:])
		c += 2
		// fill qf
		qf.Name = name
		qf.Type = q_type
		qf.Class = q_class
	}

	return
}

func parseResourceRecord(buf []byte, spos int) (rr ResourceRecordFormat, c int, err error) {
	var qf QuestionFormat
	qf, c, err = parseQuestion(buf, spos)

	if c+6 > len(buf) { // fixme?
		err = fmt.Errorf("Short datagram: %d >= %d", c+6, len(buf))
	}

	if err == nil {
		q_ttl := nUint32(buf[c:])
		c += 4
		q_rdlen := int(nUint16(buf[c:]))
		c += 2

		rr.Name = qf.Name
		rr.Type = qf.Type
		rr.Class = qf.Class
		rr.Ttl = q_ttl
		label := Namelabel{}
		if c+q_rdlen <= len(buf) { // fixme!
			switch rr.Type {
			case constants.TYPE_NS:
				fallthrough
			case constants.TYPE_CNAME:
				// This might be compressed: we uncompress this label
				label, _, err = parseName(buf, c)
				if err == nil {
					rr.Data = EncodeName(label)
				}
			case constants.TYPE_MX:
				if len(buf) > c+2 { // enough data for <uint16><char>
					label, _, err = parseName(buf, c+2)
					if err == nil {
						// copy MX priority and add expanded label:
						rr.Data = append([]byte{buf[c], buf[c+1]}, EncodeName(label)...)
					}
				} else {
					err = fmt.Errorf("Invalid MX record")
				}
			default:
				rr.Data = buf[c:(c + q_rdlen)]
			}
			c += q_rdlen
		} else {
			err = fmt.Errorf("Invalid rdlen")
		}
	}

	return
}

func ParseName(buf []byte) (Namelabel, error) {
	n, _, err := parseName(buf, 0)
	return n, err
}

func ParseSoaTtl(buf []byte) uint32 {
	l := len(buf)
	return nUint32(buf[l-4:])
}

func parseName(buf []byte, spos int) (n Namelabel, c int, err error) {
	buflen := len(buf)
	c = spos // start pos

	err = fmt.Errorf("Malformed packet at %d", spos)
	for c < buflen {
		l_len := int(buf[c])
		l_end := c + int(buf[c]) + 1 // +1 because we skip this byte
		c++                          // skip length
		if l_len&0xc0 != 0 && c < buflen {
			p_pos := int(nUint16(buf[c-1:]) ^ 0xC000) // last 14 bits specify offset
			c++
			if p_pos < spos {
				p_labels, _, p_err := parseName(buf, p_pos)
				err = p_err
				n.name = append(n.name, p_labels.name...)
			}
			break
		} else if l_end == c {
			err = nil
			n.name = append(n.name, "")
			break
		} else if l_end >= buflen {
			break
		} else {
			n.name = append(n.name, string(buf[c:l_end]))
			c = l_end
		}
	}
	return
}
