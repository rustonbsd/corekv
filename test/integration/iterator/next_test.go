package iterator

import (
	"testing"

	"github.com/rustonbsd/corekv/test/action"
	"github.com/rustonbsd/corekv/test/integration"
)

func TestIteratorNext(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			action.Set([]byte("k1"), []byte("v1")),
			&action.Iterator{
				ChildActions: []action.IteratorAction{
					action.Next(true),
				},
			},
		},
	}

	test.Execute(t)
}
