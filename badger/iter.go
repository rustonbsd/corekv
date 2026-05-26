package badger

import (
	"bytes"

	"github.com/dgraph-io/badger/v4"

	"github.com/sourcenetwork/corekv"
)

type iteratorCloser interface {
	corekv.Iterator
	withCloser(func() error)
}

type iterator struct {
	txn      *bTxn
	i        *badger.Iterator
	start    []byte
	end      []byte
	reverse  bool
	keysOnly bool
	// closer releases the implicit read-only transaction used for store-level iteration.
	// reset indicates whether the next Next call should restart iteration.
	reset  bool
	closer func() error
}

func newPrefixIterator(txn *bTxn, prefix []byte, reverse, keysOnly bool) *iterator {
	opt := badger.DefaultIteratorOptions
	opt.Reverse = reverse
	opt.Prefix = prefix
	opt.PrefetchValues = !keysOnly

	return &iterator{
		txn:      txn,
		i:        txn.t.NewIterator(opt),
		start:    prefix,
		end:      bytesPrefixEnd(prefix),
		reverse:  reverse,
		keysOnly: keysOnly,
		reset:    true,
	}
}

func newRangeIterator(txn *bTxn, start, end []byte, reverse, keysOnly bool) *iterator {
	opt := badger.DefaultIteratorOptions
	opt.Reverse = reverse
	opt.PrefetchValues = !keysOnly

	return &iterator{
		txn:      txn,
		i:        txn.t.NewIterator(opt),
		start:    start,
		end:      end,
		reverse:  reverse,
		keysOnly: keysOnly,
		reset:    true,
	}
}

func (it *iterator) Reset() {
	it.reset = true
}

func (it *iterator) Next() (bool, error) {
	it.txn.d.closeLk.RLock()
	defer it.txn.d.closeLk.RUnlock()
	if it.txn.d.closed {
		return false, corekv.ErrDBClosed
	}

	return it.next()
}

func (it *iterator) Key() []byte {
	return it.i.Item().KeyCopy(nil)
}

func (it *iterator) Value() ([]byte, error) {
	if it.keysOnly {
		return nil, nil
	}

	return it.i.Item().ValueCopy(nil)
}

func (it *iterator) Seek(key []byte) (bool, error) {
	it.txn.d.closeLk.RLock()
	defer it.txn.d.closeLk.RUnlock()
	if it.txn.d.closed {
		return false, corekv.ErrDBClosed
	}

	return it.seek(key)
}

func (it *iterator) Close() error {
	it.i.Close()
	if it.closer != nil {
		return it.closer()
	}
	return nil
}

func (it *iterator) withCloser(closer func() error) {
	it.closer = closer
}

func (it *iterator) restart() (bool, error) {
	it.reset = false
	if it.reverse {
		return it.seek(it.end)
	}
	return it.seek(it.start)
}

func (it *iterator) valid() bool {
	if !it.i.Valid() {
		return false
	}
	if len(it.start) > 0 && lt(it.i.Item().Key(), it.start) {
		return false
	}
	if len(it.end) > 0 && gte(it.i.Item().Key(), it.end) {
		return false
	}
	return true
}

func (it *iterator) next() (bool, error) {
	if it.reset {
		return it.restart()
	}
	if !it.i.Valid() {
		return false, nil
	}

	it.i.Next()
	return it.valid(), nil
}

func (it *iterator) seek(key []byte) (bool, error) {
	it.reset = false

	var target []byte
	if it.reverse {
		if it.end != nil && lt(it.end, key) {
			target = it.end
		} else {
			target = key
		}
	} else {
		if it.start != nil && lt(key, it.start) {
			target = it.start
		} else {
			target = key
		}
	}

	it.i.Seek(target)
	if !it.valid() {
		return it.next()
	}

	return true, nil
}

func bytesPrefixEnd(b []byte) []byte {
	end := make([]byte, len(b))
	copy(end, b)
	for i := len(end) - 1; i >= 0; i-- {
		end[i] = end[i] + 1
		if end[i] != 0 {
			return end[:i+1]
		}
	}
	return b
}

func lt(a, b []byte) bool {
	return bytes.Compare(a, b) == -1
}

func gte(a, b []byte) bool {
	return bytes.Compare(a, b) > -1
}
