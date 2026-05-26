package badger_ffi

import (
	"context"
	"errors"

	"github.com/rustonbsd/corekv"
)

type Datastore struct {
	handle uint64
	closed bool
}

var errWritesBlocked = errors.New("Writes are blocked, possibly due to DropAll or Close")

var _ corekv.TxnStore = (*Datastore)(nil)
var _ corekv.Dropable = (*Datastore)(nil)

func NewDatastore(path string, opts Options) (*Datastore, error) {
	handle, err := ffiOpen(path, compatOptions(opts))
	if err != nil {
		return nil, err
	}

	return &Datastore{handle: handle}, nil
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
	if b.closed {
		return nil
	}

	err := ffiDBClose(b.handle)
	if err == nil {
		b.closed = true
	}
	return err
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
	return ffiDBDropAll(d.handle)
}

func (b *Datastore) NewTxn(readonly bool) corekv.Txn {
	return b.newTxn(readonly)
}

func (b *Datastore) newTxn(readonly bool) *bTxn {
	if b.closed {
		return &bTxn{
			d:       b,
			initErr: closedTxnErr(readonly),
		}
	}

	handle, err := ffiDBNewTxn(b.handle, readonly)
	return &bTxn{
		handle:  handle,
		d:       b,
		initErr: err,
	}
}

type bTxn struct {
	handle  uint64
	d       *Datastore
	initErr error
}

func (txn *bTxn) Get(ctx context.Context, key []byte) ([]byte, error) {
	if txn.initErr != nil {
		return nil, txn.initErr
	}
	return ffiTxnGet(txn.handle, key)
}

func (txn *bTxn) Has(ctx context.Context, key []byte) (bool, error) {
	if txn.initErr != nil {
		return false, txn.initErr
	}
	return ffiTxnHas(txn.handle, key)
}

func (txn *bTxn) Iterator(ctx context.Context, iterOpts corekv.IterOptions) (corekv.Iterator, error) {
	return txn.iterator(iterOpts)
}

func (txn *bTxn) iterator(iopts corekv.IterOptions) (iteratorCloser, error) {
	if txn.initErr != nil {
		return nil, txn.initErr
	}
	handle, err := ffiTxnIterator(txn.handle, iopts)
	if err != nil {
		return nil, err
	}

	return &iterator{txn: txn, handle: handle}, nil
}

func (txn *bTxn) Set(ctx context.Context, key []byte, value []byte) error {
	if txn.initErr != nil {
		return txn.initErr
	}
	return ffiTxnSet(txn.handle, key, value)
}

func (txn *bTxn) Delete(ctx context.Context, key []byte) error {
	if txn.initErr != nil {
		return txn.initErr
	}
	return ffiTxnDelete(txn.handle, key)
}

func (txn *bTxn) Commit() error {
	if txn.initErr != nil {
		return txn.initErr
	}
	return ffiTxnCommit(txn.handle)
}

func (txn *bTxn) Discard() {
	ffiTxnDiscard(txn.handle)
	txn.handle = 0
}

func closedTxnErr(readonly bool) error {
	if readonly {
		return corekv.ErrDBClosed
	}

	return errWritesBlocked
}
