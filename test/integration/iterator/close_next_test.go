package iterator

import (
	"testing"

	"github.com/rustonbsd/corekv"
	"github.com/rustonbsd/corekv/test/action"
	"github.com/rustonbsd/corekv/test/integration"
)

func TestIteratorCloseNext(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			&action.Iterator{
				ChildActions: []action.IteratorAction{
					action.CloseRoot(),
					action.NextE(corekv.ErrDBClosed.Error()),
				},
			},
		},
	}

	test.Execute(t)
}
