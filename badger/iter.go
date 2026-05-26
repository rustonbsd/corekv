package badger

import (
	badgerffi "github.com/dgraph-io/badger/v4/ffi"

	"github.com/sourcenetwork/corekv"
)

type iteratorCloser interface {
	corekv.Iterator
	withCloser(func() error)
}

type iterator struct {
	txn *bTxn
	i   *badgerffi.Iterator
	// closer releases the implicit read-only transaction used for store-level iteration.
	reset  bool
	closer func() error
}

func (it *iterator) Reset() {
	it.i.Reset()
}

func (it *iterator) Next() (bool, error) {
	ok, err := it.i.Next()
	return ok, badgerErrToKVErr(err)
}

func (it *iterator) Key() []byte {
	return it.i.Key()
}

func (it *iterator) Value() ([]byte, error) {
	value, err := it.i.Value()
	return value, badgerErrToKVErr(err)
}

func (it *iterator) Seek(key []byte) (bool, error) {
	ok, err := it.i.Seek(key)
	return ok, badgerErrToKVErr(err)
}

func (it *iterator) Close() error {
	err := badgerErrToKVErr(it.i.Close())
	if it.closer != nil {
		closerErr := it.closer()
		if err != nil {
			return err
		}
		return closerErr
	}
	return err
}

func (it *iterator) withCloser(closer func() error) {
	it.closer = closer
}
