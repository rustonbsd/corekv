package main

/*
#include <stdint.h>
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"sync"
	"unsafe"

	badgerffi "github.com/rustonbsd/corekv/badger_ffi/ffi"
)

type handleTable[T any] struct {
	mu    sync.RWMutex
	next  uint64
	items map[uint64]T
}

func newHandleTable[T any]() *handleTable[T] {
	return &handleTable[T]{
		next:  1,
		items: make(map[uint64]T),
	}
}

func (t *handleTable[T]) add(value T) uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	handle := t.next
	t.next++
	t.items[handle] = value
	return handle
}

func (t *handleTable[T]) get(handle uint64) (T, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	value, ok := t.items[handle]
	return value, ok
}

func (t *handleTable[T]) delete(handle uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.items, handle)
}

var (
	dbHandles   = newHandleTable[*badgerffi.DB]()
	txnHandles  = newHandleTable[*badgerffi.Txn]()
	iterHandles = newHandleTable[*badgerffi.Iterator]()

	errInvalidHandle = errors.New("invalid handle")
)

func main() {}

//export badger_ffi_free
func badger_ffi_free(ptr unsafe.Pointer) {
	if ptr != nil {
		C.free(ptr)
	}
}

//export badger_ffi_db_open
func badger_ffi_db_open(
	dir *C.char,
	valueDir *C.char,
	inMemory C.uint8_t,
	encryptionKey *C.uint8_t,
	encryptionKeyLen C.size_t,
	indexCacheSize C.longlong,
	valueLogFileSize C.longlong,
	outHandle *C.uint64_t,
	errOut **C.char,
) C.int32_t {
	if outHandle == nil {
		return failStatus(badgerffi.StatusInvalidArgument, errors.New("outHandle is nil"), errOut)
	}

	db, err := badgerffi.Open(badgerffi.OpenOptions{
		Dir:              cString(dir),
		ValueDir:         cString(valueDir),
		InMemory:         inMemory != 0,
		EncryptionKey:    cBytes(encryptionKey, encryptionKeyLen),
		IndexCacheSize:   int64(indexCacheSize),
		ValueLogFileSize: int64(valueLogFileSize),
	})
	if err != nil {
		return fail(err, errOut)
	}

	*outHandle = C.uint64_t(dbHandles.add(db))
	clearError(errOut)
	return C.int32_t(badgerffi.StatusOK)
}

//export badger_ffi_db_close
func badger_ffi_db_close(handle C.uint64_t, errOut **C.char) C.int32_t {
	db, ok := dbHandles.get(uint64(handle))
	if !ok {
		return failStatus(badgerffi.StatusInvalidHandle, errInvalidHandle, errOut)
	}

	err := db.Close()
	if err == nil {
		dbHandles.delete(uint64(handle))
	}
	if err != nil {
		return fail(err, errOut)
	}

	clearError(errOut)
	return C.int32_t(badgerffi.StatusOK)
}

//export badger_ffi_db_drop_all
func badger_ffi_db_drop_all(handle C.uint64_t, errOut **C.char) C.int32_t {
	db, ok := dbHandles.get(uint64(handle))
	if !ok {
		return failStatus(badgerffi.StatusInvalidHandle, errInvalidHandle, errOut)
	}

	err := db.DropAll()
	if err != nil {
		return fail(err, errOut)
	}

	clearError(errOut)
	return C.int32_t(badgerffi.StatusOK)
}

//export badger_ffi_db_new_txn
func badger_ffi_db_new_txn(handle C.uint64_t, readOnly C.uint8_t, outHandle *C.uint64_t, errOut **C.char) C.int32_t {
	if outHandle == nil {
		return failStatus(badgerffi.StatusInvalidArgument, errors.New("outHandle is nil"), errOut)
	}

	db, ok := dbHandles.get(uint64(handle))
	if !ok {
		return failStatus(badgerffi.StatusInvalidHandle, errInvalidHandle, errOut)
	}

	*outHandle = C.uint64_t(txnHandles.add(db.NewTxn(readOnly != 0)))
	clearError(errOut)
	return C.int32_t(badgerffi.StatusOK)
}

//export badger_ffi_txn_get
func badger_ffi_txn_get(handle C.uint64_t, key *C.uint8_t, keyLen C.size_t, outValue **C.uint8_t, outValueLen *C.size_t, errOut **C.char) C.int32_t {
	txn, ok := txnHandles.get(uint64(handle))
	if !ok {
		return failStatus(badgerffi.StatusInvalidHandle, errInvalidHandle, errOut)
	}
	if outValue == nil || outValueLen == nil {
		return failStatus(badgerffi.StatusInvalidArgument, errors.New("outValue or outValueLen is nil"), errOut)
	}

	value, err := txn.Get(cBytes(key, keyLen))
	if err != nil {
		return fail(err, errOut)
	}

	setOutBytes(outValue, outValueLen, value)
	clearError(errOut)
	return C.int32_t(badgerffi.StatusOK)
}

//export badger_ffi_txn_has
func badger_ffi_txn_has(handle C.uint64_t, key *C.uint8_t, keyLen C.size_t, outHas *C.uint8_t, errOut **C.char) C.int32_t {
	txn, ok := txnHandles.get(uint64(handle))
	if !ok {
		return failStatus(badgerffi.StatusInvalidHandle, errInvalidHandle, errOut)
	}
	if outHas == nil {
		return failStatus(badgerffi.StatusInvalidArgument, errors.New("outHas is nil"), errOut)
	}

	has, err := txn.Has(cBytes(key, keyLen))
	if err != nil {
		return fail(err, errOut)
	}

	if has {
		*outHas = 1
	} else {
		*outHas = 0
	}
	clearError(errOut)
	return C.int32_t(badgerffi.StatusOK)
}

//export badger_ffi_txn_set
func badger_ffi_txn_set(handle C.uint64_t, key *C.uint8_t, keyLen C.size_t, value *C.uint8_t, valueLen C.size_t, errOut **C.char) C.int32_t {
	txn, ok := txnHandles.get(uint64(handle))
	if !ok {
		return failStatus(badgerffi.StatusInvalidHandle, errInvalidHandle, errOut)
	}

	err := txn.Set(cBytes(key, keyLen), cBytes(value, valueLen))
	if err != nil {
		return fail(err, errOut)
	}

	clearError(errOut)
	return C.int32_t(badgerffi.StatusOK)
}

//export badger_ffi_txn_delete
func badger_ffi_txn_delete(handle C.uint64_t, key *C.uint8_t, keyLen C.size_t, errOut **C.char) C.int32_t {
	txn, ok := txnHandles.get(uint64(handle))
	if !ok {
		return failStatus(badgerffi.StatusInvalidHandle, errInvalidHandle, errOut)
	}

	err := txn.Delete(cBytes(key, keyLen))
	if err != nil {
		return fail(err, errOut)
	}

	clearError(errOut)
	return C.int32_t(badgerffi.StatusOK)
}

//export badger_ffi_txn_commit
func badger_ffi_txn_commit(handle C.uint64_t, errOut **C.char) C.int32_t {
	txn, ok := txnHandles.get(uint64(handle))
	if !ok {
		return failStatus(badgerffi.StatusInvalidHandle, errInvalidHandle, errOut)
	}

	err := txn.Commit()
	if err != nil {
		return fail(err, errOut)
	}

	clearError(errOut)
	return C.int32_t(badgerffi.StatusOK)
}

//export badger_ffi_txn_discard
func badger_ffi_txn_discard(handle C.uint64_t) {
	txn, ok := txnHandles.get(uint64(handle))
	if !ok {
		return
	}

	txn.Discard()
	txnHandles.delete(uint64(handle))
}

//export badger_ffi_txn_iterator
func badger_ffi_txn_iterator(
	handle C.uint64_t,
	prefix *C.uint8_t,
	prefixLen C.size_t,
	start *C.uint8_t,
	startLen C.size_t,
	end *C.uint8_t,
	endLen C.size_t,
	reverse C.uint8_t,
	keysOnly C.uint8_t,
	outHandle *C.uint64_t,
	errOut **C.char,
) C.int32_t {
	if outHandle == nil {
		return failStatus(badgerffi.StatusInvalidArgument, errors.New("outHandle is nil"), errOut)
	}

	txn, ok := txnHandles.get(uint64(handle))
	if !ok {
		return failStatus(badgerffi.StatusInvalidHandle, errInvalidHandle, errOut)
	}

	iter, err := txn.Iterator(badgerffi.IteratorOptions{
		Prefix:   cBytes(prefix, prefixLen),
		Start:    cBytes(start, startLen),
		End:      cBytes(end, endLen),
		Reverse:  reverse != 0,
		KeysOnly: keysOnly != 0,
	})
	if err != nil {
		return fail(err, errOut)
	}

	*outHandle = C.uint64_t(iterHandles.add(iter))
	clearError(errOut)
	return C.int32_t(badgerffi.StatusOK)
}

//export badger_ffi_iter_next
func badger_ffi_iter_next(handle C.uint64_t, outHas *C.uint8_t, errOut **C.char) C.int32_t {
	iter, ok := iterHandles.get(uint64(handle))
	if !ok {
		return failStatus(badgerffi.StatusInvalidHandle, errInvalidHandle, errOut)
	}
	if outHas == nil {
		return failStatus(badgerffi.StatusInvalidArgument, errors.New("outHas is nil"), errOut)
	}

	has, err := iter.Next()
	if err != nil {
		return fail(err, errOut)
	}

	if has {
		*outHas = 1
	} else {
		*outHas = 0
	}
	clearError(errOut)
	return C.int32_t(badgerffi.StatusOK)
}

//export badger_ffi_iter_key
func badger_ffi_iter_key(handle C.uint64_t, outKey **C.uint8_t, outKeyLen *C.size_t, errOut **C.char) C.int32_t {
	iter, ok := iterHandles.get(uint64(handle))
	if !ok {
		return failStatus(badgerffi.StatusInvalidHandle, errInvalidHandle, errOut)
	}
	if outKey == nil || outKeyLen == nil {
		return failStatus(badgerffi.StatusInvalidArgument, errors.New("outKey or outKeyLen is nil"), errOut)
	}

	setOutBytes(outKey, outKeyLen, iter.Key())
	clearError(errOut)
	return C.int32_t(badgerffi.StatusOK)
}

//export badger_ffi_iter_value
func badger_ffi_iter_value(handle C.uint64_t, outValue **C.uint8_t, outValueLen *C.size_t, errOut **C.char) C.int32_t {
	iter, ok := iterHandles.get(uint64(handle))
	if !ok {
		return failStatus(badgerffi.StatusInvalidHandle, errInvalidHandle, errOut)
	}
	if outValue == nil || outValueLen == nil {
		return failStatus(badgerffi.StatusInvalidArgument, errors.New("outValue or outValueLen is nil"), errOut)
	}

	value, err := iter.Value()
	if err != nil {
		return fail(err, errOut)
	}

	setOutBytes(outValue, outValueLen, value)
	clearError(errOut)
	return C.int32_t(badgerffi.StatusOK)
}

//export badger_ffi_iter_seek
func badger_ffi_iter_seek(handle C.uint64_t, key *C.uint8_t, keyLen C.size_t, outHas *C.uint8_t, errOut **C.char) C.int32_t {
	iter, ok := iterHandles.get(uint64(handle))
	if !ok {
		return failStatus(badgerffi.StatusInvalidHandle, errInvalidHandle, errOut)
	}
	if outHas == nil {
		return failStatus(badgerffi.StatusInvalidArgument, errors.New("outHas is nil"), errOut)
	}

	has, err := iter.Seek(cBytes(key, keyLen))
	if err != nil {
		return fail(err, errOut)
	}

	if has {
		*outHas = 1
	} else {
		*outHas = 0
	}
	clearError(errOut)
	return C.int32_t(badgerffi.StatusOK)
}

//export badger_ffi_iter_reset
func badger_ffi_iter_reset(handle C.uint64_t, errOut **C.char) C.int32_t {
	iter, ok := iterHandles.get(uint64(handle))
	if !ok {
		return failStatus(badgerffi.StatusInvalidHandle, errInvalidHandle, errOut)
	}

	iter.Reset()
	clearError(errOut)
	return C.int32_t(badgerffi.StatusOK)
}

//export badger_ffi_iter_close
func badger_ffi_iter_close(handle C.uint64_t, errOut **C.char) C.int32_t {
	iter, ok := iterHandles.get(uint64(handle))
	if !ok {
		return failStatus(badgerffi.StatusInvalidHandle, errInvalidHandle, errOut)
	}

	err := iter.Close()
	if err == nil {
		iterHandles.delete(uint64(handle))
	}
	if err != nil {
		return fail(err, errOut)
	}

	clearError(errOut)
	return C.int32_t(badgerffi.StatusOK)
}

func cString(value *C.char) string {
	if value == nil {
		return ""
	}
	return C.GoString(value)
}

func cBytes(ptr *C.uint8_t, n C.size_t) []byte {
	if ptr == nil || n == 0 {
		return nil
	}

	return C.GoBytes(unsafe.Pointer(ptr), C.int(n))
}

func setOutBytes(out **C.uint8_t, outLen *C.size_t, data []byte) {
	if len(data) == 0 {
		*out = nil
		*outLen = 0
		return
	}

	ptr := C.CBytes(data)
	*out = (*C.uint8_t)(ptr)
	*outLen = C.size_t(len(data))
}

func fail(err error, errOut **C.char) C.int32_t {
	return failStatus(badgerffi.StatusFromError(err), err, errOut)
}

func failStatus(status badgerffi.Status, err error, errOut **C.char) C.int32_t {
	setError(errOut, err)
	return C.int32_t(status)
}

func clearError(errOut **C.char) {
	if errOut != nil {
		*errOut = nil
	}
}

func setError(errOut **C.char, err error) {
	if errOut == nil {
		return
	}
	if err == nil {
		*errOut = nil
		return
	}
	*errOut = C.CString(err.Error())
}
