package main

import (
	"fmt"
	"os"

	"github.com/sourcenetwork/corekv/badger_ffi/ffi"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "save_disk failed: %v\n", err)
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

	txn := db.NewTxn(false)
	defer txn.Discard()

	if err := txn.Set([]byte("cross/alpha"), []byte("one")); err != nil {
		return err
	}
	if err := txn.Set([]byte("cross/beta"), []byte("two")); err != nil {
		return err
	}
	if err := txn.Set([]byte("cross/empty"), nil); err != nil {
		return err
	}
	if err := txn.Commit(); err != nil {
		return err
	}

	fmt.Printf("saved database at %s\n", path)
	fmt.Println("wrote cross/alpha=one")
	fmt.Println("wrote cross/beta=two")
	fmt.Println("wrote cross/empty=<nil>")
	return nil
}
