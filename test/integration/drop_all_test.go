package integration

import (
	"testing"

	"github.com/rustonbsd/corekv/test/action"
)

func TestDropAll(t *testing.T) {
	test := &Test{
		Actions: []action.Action{
			action.New(),
			action.DropAll(),
		},
	}

	test.Execute(t)
}
