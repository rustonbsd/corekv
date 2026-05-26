package iterator

import (
	"testing"

	"github.com/rustonbsd/corekv"
	"github.com/rustonbsd/corekv/test/action"
	"github.com/rustonbsd/corekv/test/integration"
)

func TestIteratorReverseEndNext(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			&action.Iterator{
				IterOptions: corekv.IterOptions{
					Reverse: true,
					End:     []byte("k4"),
				},
				ChildActions: []action.IteratorAction{
					action.Next(false),
				},
			},
		},
	}

	test.Execute(t)
}
