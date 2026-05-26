package iterator

import (
	"testing"

	"github.com/rustonbsd/corekv/test/action"
	"github.com/rustonbsd/corekv/test/integration"
	"github.com/rustonbsd/corekv/test/multiplier"
)

func TestIteratorCloseNextValue_NoTxn(t *testing.T) {
	test := &integration.Test{
		Excludes: []multiplier.Name{
			// Test behaviour varies a bit with the txn multipliers at the moment,
			// with the stores all failing in slightly different ways.
			// https://github.com/rustonbsd/corekv/issues/68
			multiplier.TxnDiscard,
			multiplier.TxnCommit,
			multiplier.TxnMulti,
		},
		Actions: []action.Action{
			action.Set([]byte("k1"), []byte("v1")),
			&action.Iterator{
				ChildActions: []action.IteratorAction{
					action.Next(true),
					action.CloseRoot(),
					action.Value([]byte("v1")),
				},
			},
		},
	}

	test.Execute(t)
}
