package packet

import "strings"

// The parsed representation of a DNS header
type ParsedPacketHeader struct {
	Id              uint16 // Id of this query
	Response        bool   // `true' if this is a response (qr)
	Opcode          uint8  // The RFC1035 opcode of this query (usually OP_QUERY)
	Authoritative   bool   // `true' if we have authorative data in the reply
	Truncated       bool   // `true' if the query was truncated and might be re-done using TCP or EDNS
	RecDesired      bool   // `true' if the client asked us to resolve recursively
	RecAvailable    bool   // indicates if the host generating this reply is willing to do recursive queries
	ResponseCode    uint8  // RFC1035 response code, such as NXDOMAIN
	QuestionCount   uint16 // Number of questions in this packet
	AnswerCount     uint16 // Number of items in the answer section
	NameserverCount uint16 // Number of items in the NS section
	AdditionalCount uint16 // Number of items in the additional section
}

// String array typed to describe DNS labels
type Namelabel struct {
	name []string
}

// Returns a string version of given Namelabel reference
func (l *Namelabel) ToKey() string {
	return strings.ToUpper(l.ToCaseSensitiveKey())
}

// Returns a CASE SENSITIVE key.
// Use this if you want to verify a ShuffleCases()'ed namelabel
func (l *Namelabel) ToCaseSensitiveKey() string {
	return strings.Join(l.name, "/") + ";"
}

// Pops a label from the label list, walking the hierarchy
func (l *Namelabel) PoppedLabel(n int) *Namelabel {
	var result []string
	for i := n; i < len(l.name); i++ {
		result = append(result, l.name[i])
	}
	return &Namelabel{result}
}

// Returs the number of labels in the list (1 == .)
func (l *Namelabel) Len() int {
	return len(l.name)
}

// Returns true if given namelabel is a child of *parent
func (l *Namelabel) IsChildOf(parent *Namelabel) bool {
	lc := l.Len()
	lp := parent.Len()

	if lp > lc {
		return false
	}

	for i := 1; i <= lp; i++ {
		if strings.ToUpper(l.name[lc-i]) != strings.ToUpper(parent.name[lp-i]) {
			return false
		}
	}
	return true
}

// Returns a copy of this namelabel but with
// shuffled cases
func (l *Namelabel) ShuffleCases() *Namelabel {
	var result []string
	for _, v := range l.name {
		// fixme: use something random, not Title()
		result = append(result, strings.Title(v))
	}
	return &Namelabel{result}
}

// A fully parsed DNS packet
type ParsedPacket struct {
	Header      ParsedPacketHeader
	Questions   []QuestionFormat
	Answers     []ResourceRecordFormat
	Nameservers []ResourceRecordFormat
	Additionals []ResourceRecordFormat
}

// Question section of the DNS packet
type QuestionFormat struct {
	Name  Namelabel
	Type  uint16
	Class uint16
}

// RR section of a DNS packet
type ResourceRecordFormat struct {
	Name  Namelabel
	Type  uint16
	Class uint16
	Ttl   uint32
	Data  []byte // note: this is RAW data!
}
