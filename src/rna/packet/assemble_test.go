package packet

import (
	"fmt"
	"testing"
)

func TestRFC2181Sect52(t *testing.T) {
	pp := &ParsedPacket{}

	label, _ := ParseName([]byte{0})

	pp.Answers = append(pp.Answers, ResourceRecordFormat{Name: label, Ttl: 30})
	pp.Answers = append(pp.Answers, ResourceRecordFormat{Name: label, Ttl: 40})

	raw := Assemble(pp)
	parsed, err := Parse(raw)

	if err != nil {
		panic(err)
	}

	if parsed.Answers[0].Ttl != parsed.Answers[1].Ttl {
		panic(fmt.Errorf("RRSet with different TTLs are forbidden"))
	}
}
