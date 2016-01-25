package packet

import (
	"fmt"
	"testing"
)

// Should produce a parser error:
// c0 loops back to header start
func TestXH(t *testing.T) {
	na := &Namelabel{[]string{"com", ""}}
	nb := &Namelabel{[]string{"example", "com", ""}}

	if na.IsChildOf(nb) == true {
		panic(fmt.Errorf("Should not be true"))
	}

	if nb.IsChildOf(na) == false {
		panic(fmt.Errorf("Should not be false"))
	}

	if na.IsChildOf(na) == true {
		panic(fmt.Errorf("Same level should not be a child"))
	}

}
