package main

import (
	"fmt"
	"os"

	"github.com/sourcenetwork/corekv/badger_ffi/ffi"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "iter_bounds_reverse failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	path := "/tmp/badger-iter-example"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	db, err := ffi.Open(ffi.OpenOptions{
		Dir:      path,
		ValueDir: path,
	})
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.DropAll(); err != nil {
		return err
	}
	if err := seed(db); err != nil {
		return err
	}

	txn := db.NewTxn(true)
	defer txn.Discard()

	fmt.Printf("iterating database at %s\n", path)
	if err := printRange(txn, "forward range [cross/015, cross/045)", ffi.IteratorOptions{
		Start: []byte("cross/015"),
		End:   []byte("cross/045"),
	}); err != nil {
		return err
	}
	if err := printRange(txn, "reverse range [cross/015, cross/045)", ffi.IteratorOptions{
		Start:   []byte("cross/015"),
		End:     []byte("cross/045"),
		Reverse: true,
	}); err != nil {
		return err
	}

	return nil
}

func seed(db *ffi.DB) error {
	txn := db.NewTxn(false)
	defer txn.Discard()

	if err := txn.Set([]byte("cross/010"), []byte("ten")); err != nil {
		return err
	}
	if err := txn.Set([]byte("cross/020"), []byte("twenty")); err != nil {
		return err
	}
	if err := txn.Set([]byte("cross/030"), []byte("thirty")); err != nil {
		return err
	}
	if err := txn.Set([]byte("cross/040"), []byte("forty")); err != nil {
		return err
	}
	if err := txn.Set([]byte("cross/050"), []byte("fifty")); err != nil {
		return err
	}

	return txn.Commit()
}

func printRange(txn *ffi.Txn, label string, opts ffi.IteratorOptions) error {
	iter, err := txn.Iterator(opts)
	if err != nil {
		return err
	}
	defer iter.Close()

	fmt.Println(label)
	for {
		hasNext, err := iter.Next()
		if err != nil {
			return err
		}
		if !hasNext {
			return nil
		}

		value, err := iter.Value()
		if err != nil {
			return err
		}
		fmt.Printf("%s=%s\n", string(iter.Key()), string(value))
	}
}
