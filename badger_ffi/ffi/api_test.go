package ffi

import (
	"testing"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
)

func TestOpenInMemoryIgnoresPaths(t *testing.T) {
	db, err := Open(OpenOptions{
		Dir:              "/tmp/ignored-dir",
		ValueDir:         "/tmp/ignored-value-dir",
		InMemory:         true,
		ValueLogFileSize: 1 << 20,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	txn := db.NewTxn(false)
	require.NoError(t, txn.Set([]byte("k"), []byte("v")))
	require.NoError(t, txn.Commit())
	txn.Discard()

	readTxn := db.NewTxn(true)
	defer readTxn.Discard()

	value, err := readTxn.Get([]byte("k"))
	require.NoError(t, err)
	require.Equal(t, []byte("v"), value)
}

func TestReadOnlyTxnSetReturnsBadgerError(t *testing.T) {
	db, err := Open(OpenOptions{InMemory: true})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	txn := db.NewTxn(true)
	defer txn.Discard()

	err = txn.Set([]byte("k"), []byte("v"))
	require.ErrorIs(t, err, badger.ErrReadOnlyTxn)
	require.Equal(t, StatusReadOnlyTxn, StatusFromError(err))
}

func TestNilValueRoundTripsAsNil(t *testing.T) {
	db, err := Open(OpenOptions{InMemory: true})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	txn := db.NewTxn(false)
	require.NoError(t, txn.Set([]byte("k"), nil))
	require.NoError(t, txn.Commit())
	txn.Discard()

	readTxn := db.NewTxn(true)
	defer readTxn.Discard()

	value, err := readTxn.Get([]byte("k"))
	require.NoError(t, err)
	require.Nil(t, value)

	iter, err := readTxn.Iterator(IteratorOptions{})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, iter.Close())
	}()

	ok, err := iter.Next()
	require.NoError(t, err)
	require.True(t, ok)

	iterValue, err := iter.Value()
	require.NoError(t, err)
	require.Nil(t, iterValue)
}

func TestIteratorReturnsDBClosedAfterClose(t *testing.T) {
	db, err := Open(OpenOptions{InMemory: true})
	require.NoError(t, err)

	txn := db.NewTxn(true)
	defer txn.Discard()

	iter, err := txn.Iterator(IteratorOptions{})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, iter.Close())
	}()

	require.NoError(t, db.Close())

	_, err = txn.Iterator(IteratorOptions{})
	require.ErrorIs(t, err, badger.ErrDBClosed)

	_, err = iter.Next()
	require.ErrorIs(t, err, badger.ErrDBClosed)

	_, err = iter.Seek([]byte("k"))
	require.ErrorIs(t, err, badger.ErrDBClosed)
}

func TestIteratorMatchesPrefixRangeBehavior(t *testing.T) {
	db, err := Open(OpenOptions{InMemory: true})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	txn := db.NewTxn(false)
	for _, key := range []string{"aa1", "aa2", "ab1"} {
		require.NoError(t, txn.Set([]byte(key), []byte(key+"-value")))
	}
	require.NoError(t, txn.Commit())
	txn.Discard()

	readTxn := db.NewTxn(true)
	defer readTxn.Discard()

	iter, err := readTxn.Iterator(IteratorOptions{Prefix: []byte("aa")})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, iter.Close())
	}()

	ok, err := iter.Next()
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []byte("aa1"), iter.Key())

	ok, err = iter.Next()
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []byte("aa2"), iter.Key())

	ok, err = iter.Next()
	require.NoError(t, err)
	require.False(t, ok)

	iter.Reset()
	ok, err = iter.Seek([]byte("aa2"))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []byte("aa2"), iter.Key())
}