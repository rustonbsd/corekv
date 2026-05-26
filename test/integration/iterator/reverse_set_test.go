package iterator

import (
	"testing"

	"github.com/rustonbsd/corekv"
	"github.com/rustonbsd/corekv/test/action"
	"github.com/rustonbsd/corekv/test/integration"
)

func TestIteratorReverseSet(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			action.Set([]byte("k1"), []byte("v1")),
			action.Set([]byte("k1"), []byte("v1.1")),
			&action.Iterate{
				IterOptions: corekv.IterOptions{
					Reverse: true,
				},
				Expected: []action.KeyValue{
					{Key: []byte("k1"), Value: []byte("v1.1")},
				},
			},
		},
	}

	test.Execute(t)
}
