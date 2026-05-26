package iterator

import (
	"testing"

	"github.com/rustonbsd/corekv/test/action"
	"github.com/rustonbsd/corekv/test/integration"
)

func TestIteratorResetSeekNextValue(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			action.Set([]byte("k1"), []byte("v1")),
			action.Set([]byte("k2"), []byte("v2")),
			&action.Iterator{
				ChildActions: []action.IteratorAction{
					action.Reset(),
					action.Seek([]byte("k1"), true),
					action.Value([]byte("v1")),
					action.Next(true),
					action.Value([]byte("v2")),
				},
			},
		},
	}

	test.Execute(t)
}
