package integration

import (
	"testing"

	"github.com/rustonbsd/corekv/test/action"
)

func TestCancel(t *testing.T) {
	test := &Test{
		Actions: []action.Action{
			action.New(),
			action.Cancel(),
		},
	}

	test.Execute(t)
}
