package queue

import (
	"fmt"
)

type putCbItem struct {
	Key  string
	Type uint16
}

func (r *putCbItem) ToString() string {
	return fmt.Sprintf("%d->%s", r.Type, r.Key)
}
