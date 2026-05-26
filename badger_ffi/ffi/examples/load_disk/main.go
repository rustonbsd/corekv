package main

import (
	"fmt"
	"os"

	"github.com/rustonbsd/corekv/badger_ffi/ffi"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "load_disk failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	path := "/tmp/badger-go-example"
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

	txn := db.NewTxn(true)
	defer txn.Discard()

	iter, err := txn.Iterator(ffi.IteratorOptions{
		Prefix: []byte("cross/"),
	})
	if err != nil {
		return err
	}
	defer iter.Close()

	fmt.Printf("loaded database at %s\n", path)
	for {
		hasNext, err := iter.Next()
		if err != nil {
			return err
		}
		if !hasNext {
			break
		}

		key := string(iter.Key())
		value, err := iter.Value()
		if err != nil {
			return err
		}

		if value == nil {
			fmt.Printf("%s=<nil>\n", key)
			continue
		}

		fmt.Printf("%s=%s\n", key, string(value))
	}

	return nil
}
