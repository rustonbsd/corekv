package badger

import (
	"context"

	"github.com/dgraph-io/badger/v4"
	badgerffi "github.com/dgraph-io/badger/v4/ffi"

	"github.com/sourcenetwork/corekv"
)

type Datastore struct {
	db *badgerffi.DB
}

var _ corekv.TxnStore = (*Datastore)(nil)
var _ corekv.Dropable = (*Datastore)(nil)

func NewDatastore(path string, opts badger.Options) (*Datastore, error) {
	store, err := badgerffi.Open(badgerffi.OpenOptions{
		Dir:              path,
		ValueDir:         path,
		InMemory:         opts.InMemory,
		EncryptionKey:    opts.EncryptionKey,
		IndexCacheSize:   opts.IndexCacheSize,
		ValueLogFileSize: opts.ValueLogFileSize,
	})
	if err != nil {
		return nil, err
	}

	return &Datastore{db: store}, nil
}

func NewDatastoreFrom(db *badger.DB) *Datastore {
	return &Datastore{
		db: badgerffi.NewFrom(db),
	}
}

func (b *Datastore) Get(ctx context.Context, key []byte) ([]byte, error) {
	txn, ok := corekv.TryGetCtxTxnG[*bTxn](ctx)
	if ok {
		return txn.Get(ctx, key)
	}

	txn = b.newTxn(true)
	defer txn.Discard()

	return txn.Get(ctx, key)
}

func (b *Datastore) Has(ctx context.Context, key []byte) (bool, error) {
	txn, ok := corekv.TryGetCtxTxnG[*bTxn](ctx)
	if ok {
		return txn.Has(ctx, key)
	}

	txn = b.newTxn(true)
	defer txn.Discard()

	return txn.Has(ctx, key)
}

func (b *Datastore) Set(ctx context.Context, key []byte, value []byte) error {
	txn, ok := corekv.TryGetCtxTxnG[*bTxn](ctx)
	if ok {
		return txn.Set(ctx, key, value)
	}

	txn = b.newTxn(false)
	defer txn.Discard()

	err := txn.Set(ctx, key, value)
	if err != nil {
		return err
	}

	return txn.Commit()
}

func (b *Datastore) Delete(ctx context.Context, key []byte) error {
	txn, ok := corekv.TryGetCtxTxnG[*bTxn](ctx)
	if ok {
		return txn.Delete(ctx, key)
	}

	txn = b.newTxn(false)
	defer txn.Discard()

	err := txn.Delete(ctx, key)
	if err != nil {
		return err
	}

	return txn.Commit()
}

func (b *Datastore) Close() error {
	return b.db.Close()
}

func (b *Datastore) Iterator(ctx context.Context, iterOpts corekv.IterOptions) (corekv.Iterator, error) {
	txn, ok := corekv.TryGetCtxTxnG[*bTxn](ctx)
	if ok {
		return txn.iterator(iterOpts)
	}
	txn = b.newTxn(true)

	it, err := txn.iterator(iterOpts)
	if err != nil {
		return nil, err
	}

	// closer for discarding implicit txn
	// so that the txn is discarded when the
	// iterator is closed
	it.withCloser(func() error {
		txn.Discard()
		return nil
	})

	return it, nil
}

func (d *Datastore) DropAll() error {
	return d.db.DropAll()
}

func (b *Datastore) NewTxn(readonly bool) corekv.Txn {
	return b.newTxn(readonly)
}

func (b *Datastore) newTxn(readonly bool) *bTxn {
	return &bTxn{
		t: b.db.NewTxn(readonly),
		d: b,
	}
}

type bTxn struct {
	t *badgerffi.Txn
	d *Datastore
}

func (txn *bTxn) Get(ctx context.Context, key []byte) ([]byte, error) {
	value, err := txn.t.Get(key)
	return value, badgerErrToKVErr(err)
}

func (txn *bTxn) Has(ctx context.Context, key []byte) (bool, error) {
	has, err := txn.t.Has(key)
	return has, badgerErrToKVErr(err)
}

func (txn *bTxn) Iterator(ctx context.Context, iterOpts corekv.IterOptions) (corekv.Iterator, error) {
	return txn.iterator(iterOpts)
}

func (txn *bTxn) iterator(iopts corekv.IterOptions) (iteratorCloser, error) {
	it, err := txn.t.Iterator(badgerffi.IteratorOptions{
		Prefix:   iopts.Prefix,
		Start:    iopts.Start,
		End:      iopts.End,
		Reverse:  iopts.Reverse,
		KeysOnly: iopts.KeysOnly,
	})
	if err != nil {
		return nil, badgerErrToKVErr(err)
	}

	return &iterator{txn: txn, i: it}, nil
}

func (txn *bTxn) Set(ctx context.Context, key []byte, value []byte) error {
	return badgerErrToKVErr(txn.t.Set(key, value))
}

func (txn *bTxn) Delete(ctx context.Context, key []byte) error {
	return badgerErrToKVErr(txn.t.Delete(key))
}

func (txn *bTxn) Commit() error {
	return badgerErrToKVErr(txn.t.Commit())
}

func (txn *bTxn) Discard() {
	txn.t.Discard()
}

func badgerErrToKVErr(err error) error {
	if err == nil {
		return nil
	}

	switch badgerffi.StatusFromError(err) {
	case badgerffi.StatusEmptyKey:
		return corekv.ErrEmptyKey
	case badgerffi.StatusNotFound:
		return corekv.ErrNotFound
	case badgerffi.StatusDiscardedTxn:
		return corekv.ErrDiscardedTxn
	case badgerffi.StatusDBClosed:
		return corekv.ErrDBClosed
	case badgerffi.StatusTxnConflict:
		return corekv.ErrTxnConflict
	case badgerffi.StatusReadOnlyTxn:
		return corekv.ErrReadOnlyTxn
	default:
		return err
	}
}
