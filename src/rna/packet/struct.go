package packet

import "strings"

// The parsed representation of a DNS header
type ParsedPacketHeader struct {
	Id              uint16
	Response        bool
	Opcode          uint8
	Authoritative   bool
	Truncated       bool
	RecDesired      bool
	RecAvailable    bool
	ResponseCode    uint8
	QuestionCount   uint16
	AnswerCount     uint16
	NameserverCount uint16
	AdditionalCount uint16
}

// String array typed to describe DNS labels
type Namelabel struct {
	name []string
}

// Returns a string version of given Namelabel reference
func (l *Namelabel) ToKey() string {
	return strings.ToUpper(strings.Join(l.name, "/") + ";")
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
		if l.name[lc-i] != parent.name[lp-i] {
			return false
		}
	}
	return true
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
