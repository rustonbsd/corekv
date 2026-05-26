package ffi

import (
	"bytes"
	"errors"
	"strings"
	"sync"

	badger "github.com/dgraph-io/badger/v4"
)

type OpenOptions struct {
	Dir              string
	ValueDir         string
	InMemory         bool
	EncryptionKey    []byte
	IndexCacheSize   int64
	ValueLogFileSize int64
}

type DB struct {
	db *badger.DB

	// Badger panics when creating a new iterator on a closed DB. The FFI surface
	// must return an error instead so Rust can preserve corekv's behavior.
	closed  bool
	closeLk sync.RWMutex
}

type Txn struct {
	t  *badger.Txn
	db *DB
}

type IteratorOptions struct {
	Prefix   []byte
	Start    []byte
	End      []byte
	Reverse  bool
	KeysOnly bool
}

type Iterator struct {
	txn      *Txn
	i        *badger.Iterator
	start    []byte
	end      []byte
	reverse  bool
	keysOnly bool
	reset    bool
}

func NewFrom(db *badger.DB) *DB {
	return &DB{db: db}
}

func Open(opts OpenOptions) (*DB, error) {
	if len(opts.EncryptionKey) != 0 && len(opts.EncryptionKey) != 32 {
		return nil, badger.ErrInvalidEncryptionKey
	}

	dir := opts.Dir
	valueDir := opts.ValueDir
	if opts.InMemory {
		// Badger rejects non-empty paths in in-memory mode.
		dir = ""
		valueDir = ""
	}

	badgerOpts := badger.DefaultOptions(dir)
	badgerOpts.ValueDir = valueDir
	badgerOpts.Logger = nil
	badgerOpts.InMemory = opts.InMemory
	badgerOpts.EncryptionKey = cloneBytes(opts.EncryptionKey)
	if opts.IndexCacheSize > 0 {
		badgerOpts.IndexCacheSize = opts.IndexCacheSize
	}
	if opts.ValueLogFileSize > 0 {
		badgerOpts.ValueLogFileSize = opts.ValueLogFileSize
	}

	db, err := badger.Open(badgerOpts)
	if err != nil {
		return nil, err
	}

	return &DB{db: db}, nil
}

func (db *DB) Close() error {
	db.closeLk.Lock()
	defer db.closeLk.Unlock()
	db.closed = true

	return db.db.Close()
}

func (db *DB) DropAll() error {
	return db.db.DropAll()
}

func (db *DB) NewTxn(readOnly bool) *Txn {
	return &Txn{
		t:  db.db.NewTransaction(!readOnly),
		db: db,
	}
}

func (txn *Txn) Get(key []byte) ([]byte, error) {
	item, err := txn.t.Get(key)
	if err != nil {
		return nil, err
	}

	value, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	return normalizeValue(value), nil
}

func (txn *Txn) Has(key []byte) (bool, error) {
	_, err := txn.t.Get(key)
	switch {
	case errors.Is(err, badger.ErrKeyNotFound):
		return false, nil
	case err == nil:
		return true, nil
	default:
		return false, err
	}
}

func (txn *Txn) Set(key, value []byte) error {
	return txn.t.Set(key, value)
}

func (txn *Txn) Delete(key []byte) error {
	return txn.t.Delete(key)
}

func (txn *Txn) Commit() error {
	return txn.t.Commit()
}

func (txn *Txn) Discard() {
	txn.t.Discard()
}

func (txn *Txn) Iterator(opts IteratorOptions) (*Iterator, error) {
	txn.db.closeLk.RLock()
	defer txn.db.closeLk.RUnlock()
	if txn.db.closed {
		return nil, badger.ErrDBClosed
	}

	if opts.Prefix != nil {
		return newPrefixIterator(txn, opts.Prefix, opts.Reverse, opts.KeysOnly), nil
	}

	return newRangeIterator(txn, opts.Start, opts.End, opts.Reverse, opts.KeysOnly), nil
}

func newPrefixIterator(txn *Txn, prefix []byte, reverse, keysOnly bool) *Iterator {
	opt := badger.DefaultIteratorOptions
	opt.Reverse = reverse
	opt.Prefix = cloneBytes(prefix)
	opt.PrefetchValues = !keysOnly

	return &Iterator{
		txn:      txn,
		i:        txn.t.NewIterator(opt),
		start:    cloneBytes(prefix),
		end:      bytesPrefixEnd(prefix),
		reverse:  reverse,
		keysOnly: keysOnly,
		reset:    true,
	}
}

func newRangeIterator(txn *Txn, start, end []byte, reverse, keysOnly bool) *Iterator {
	opt := badger.DefaultIteratorOptions
	opt.Reverse = reverse
	opt.PrefetchValues = !keysOnly

	return &Iterator{
		txn:      txn,
		i:        txn.t.NewIterator(opt),
		start:    cloneBytes(start),
		end:      cloneBytes(end),
		reverse:  reverse,
		keysOnly: keysOnly,
		reset:    true,
	}
}

func (it *Iterator) Reset() {
	it.reset = true
}

func (it *Iterator) Next() (bool, error) {
	it.txn.db.closeLk.RLock()
	defer it.txn.db.closeLk.RUnlock()
	if it.txn.db.closed {
		return false, badger.ErrDBClosed
	}

	return it.next()
}

func (it *Iterator) Key() []byte {
	return it.i.Item().KeyCopy(nil)
}

func (it *Iterator) Value() ([]byte, error) {
	if it.keysOnly {
		return nil, nil
	}

	value, err := it.i.Item().ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	return normalizeValue(value), nil
}

func (it *Iterator) Seek(key []byte) (bool, error) {
	it.txn.db.closeLk.RLock()
	defer it.txn.db.closeLk.RUnlock()
	if it.txn.db.closed {
		return false, badger.ErrDBClosed
	}

	return it.seek(key)
}

func (it *Iterator) Close() error {
	it.i.Close()
	return nil
}

func (it *Iterator) next() (bool, error) {
	if it.reset {
		return it.restart()
	}

	if !it.i.Valid() {
		return false, nil
	}

	it.i.Next()
	return it.valid(), nil
}

func (it *Iterator) restart() (bool, error) {
	it.reset = false
	if it.reverse {
		return it.seek(it.end)
	}

	return it.seek(it.start)
}

func (it *Iterator) valid() bool {
	if !it.i.Valid() {
		return false
	}

	key := it.i.Item().Key()
	if len(it.start) > 0 && bytes.Compare(key, it.start) < 0 {
		return false
	}
	if len(it.end) > 0 && bytes.Compare(key, it.end) >= 0 {
		return false
	}

	return true
}

func (it *Iterator) seek(key []byte) (bool, error) {
	it.reset = false

	var target []byte
	if it.reverse {
		if it.end != nil && bytes.Compare(it.end, key) < 0 {
			target = it.end
		} else {
			target = key
		}
	} else {
		if it.start != nil && bytes.Compare(key, it.start) < 0 {
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

func bytesPrefixEnd(prefix []byte) []byte {
	end := cloneBytes(prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end[:i+1]
		}
	}

	return cloneBytes(prefix)
}

func cloneBytes(src []byte) []byte {
	if src == nil {
		return nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

func normalizeValue(value []byte) []byte {
	if len(value) == 0 {
		return nil
	}

	return value
}

type Status int32

const (
	StatusOK Status = iota
	StatusInvalidArgument
	StatusInvalidHandle
	StatusNotFound
	StatusEmptyKey
	StatusDiscardedTxn
	StatusDBClosed
	StatusTxnConflict
	StatusReadOnlyTxn
	StatusInvalidEncryptionKey
	StatusInvalidValueLogSize
	StatusUnknown
)

func StatusFromError(err error) Status {
	if err == nil {
		return StatusOK
	}

	switch {
	case errors.Is(err, badger.ErrKeyNotFound):
		return StatusNotFound
	case errors.Is(err, badger.ErrEmptyKey):
		return StatusEmptyKey
	case errors.Is(err, badger.ErrDiscardedTxn):
		return StatusDiscardedTxn
	case errors.Is(err, badger.ErrDBClosed):
		return StatusDBClosed
	case errors.Is(err, badger.ErrConflict):
		return StatusTxnConflict
	case errors.Is(err, badger.ErrReadOnlyTxn):
		return StatusReadOnlyTxn
	case errors.Is(err, badger.ErrInvalidEncryptionKey):
		return StatusInvalidEncryptionKey
	case errors.Is(err, badger.ErrValueLogSize):
		return StatusInvalidValueLogSize
	case strings.Contains(err.Error(), badger.ErrDBClosed.Error()):
		return StatusDBClosed
	case strings.Contains(err.Error(), badger.ErrConflict.Error()):
		return StatusTxnConflict
	default:
		return StatusUnknown
	}
}