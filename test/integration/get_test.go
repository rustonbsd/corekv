package integration

import (
	"testing"

	"github.com/rustonbsd/corekv"
	"github.com/rustonbsd/corekv/test/action"
)

func TestGet_NoneExistantKey_Errors(t *testing.T) {
	test := &Test{
		Actions: []action.Action{
			action.GetE([]byte("does not exist"), corekv.ErrNotFound.Error()),
		},
	}

	test.Execute(t)
}
