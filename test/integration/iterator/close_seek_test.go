package iterator

import (
	"testing"

	"github.com/rustonbsd/corekv"
	"github.com/rustonbsd/corekv/test/action"
	"github.com/rustonbsd/corekv/test/integration"
)

func TestIteratorCloseSeek(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			&action.Iterator{
				ChildActions: []action.IteratorAction{
					action.CloseRoot(),
					action.SeekE([]byte("any key"), corekv.ErrDBClosed.Error()),
				},
			},
		},
	}

	test.Execute(t)
}
