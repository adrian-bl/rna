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
	nc := &Namelabel{[]string{"xeample", "com", ""}}
	nd := &Namelabel{[]string{"eXample", "COM", ""}}

	if na.IsChildOf(nb) == true {
		panic(fmt.Errorf("Should not be true"))
	}

	if nb.IsChildOf(na) == false {
		panic(fmt.Errorf("Should not be false"))
	}

	if na.IsChildOf(na) == false {
		panic(fmt.Errorf("Same level should be a child"))
	}

	if nc.IsChildOf(nb) == true {
		panic(fmt.Errorf("This should be false"))
	}

	if nd.IsChildOf(nb) == false {
		panic(fmt.Errorf("Test should be case INSENSITIVE"))
	}
}
