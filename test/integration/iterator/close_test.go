package iterator

import (
	"testing"

	"github.com/rustonbsd/corekv"
	"github.com/rustonbsd/corekv/test/action"
	"github.com/rustonbsd/corekv/test/integration"
)

func TestIteratorClose(t *testing.T) {
	test := &integration.Test{
		Actions: []action.Action{
			action.Close(),
			&action.Iterate{
				ExpectedError: corekv.ErrDBClosed.Error(),
			},
		},
	}

	test.Execute(t)
}
