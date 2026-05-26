package badger_ffi

import (
	"github.com/rustonbsd/corekv
)

type iteratorCloser interface {
	corekv.Iterator
	withCloser(func() error)
}

type iterator struct {
	txn    *bTxn
	handle uint64
	// closer releases the implicit read-only transaction used for store-level iteration.
	closer func() error
}

func (it *iterator) Reset() {
	if err := ffiIterReset(it.handle); err != nil {
		panic(err)
	}
}

func (it *iterator) Next() (bool, error) {
	return ffiIterNext(it.handle)
}

func (it *iterator) Key() []byte {
	key, err := ffiIterKey(it.handle)
	if err != nil {
		panic(err)
	}
	return key
}

func (it *iterator) Value() ([]byte, error) {
	return ffiIterValue(it.handle)
}

func (it *iterator) Seek(key []byte) (bool, error) {
	return ffiIterSeek(it.handle, key)
}

func (it *iterator) Close() error {
	err := ffiIterClose(it.handle)
	if err == nil {
		it.handle = 0
	}
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
