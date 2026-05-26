package badger_ffi

/*
#cgo LDFLAGS: -L${SRCDIR}/lib -Wl,-rpath,${SRCDIR}/lib -lbadgerffi
#include <stdint.h>
#include <stdlib.h>

void badger_ffi_free(void* ptr);

int32_t badger_ffi_db_open(
	char* dir,
	char* valueDir,
	uint8_t inMemory,
	uint8_t* encryptionKey,
	size_t encryptionKeyLen,
	long long indexCacheSize,
	long long valueLogFileSize,
	uint64_t* outHandle,
	char** errOut
);
int32_t badger_ffi_db_close(uint64_t handle, char** errOut);
int32_t badger_ffi_db_drop_all(uint64_t handle, char** errOut);
int32_t badger_ffi_db_new_txn(uint64_t handle, uint8_t readOnly, uint64_t* outHandle, char** errOut);

int32_t badger_ffi_txn_get(uint64_t handle, uint8_t* key, size_t keyLen, uint8_t** outValue, size_t* outValueLen, char** errOut);
int32_t badger_ffi_txn_has(uint64_t handle, uint8_t* key, size_t keyLen, uint8_t* outHas, char** errOut);
int32_t badger_ffi_txn_set(uint64_t handle, uint8_t* key, size_t keyLen, uint8_t* value, size_t valueLen, char** errOut);
int32_t badger_ffi_txn_delete(uint64_t handle, uint8_t* key, size_t keyLen, char** errOut);
int32_t badger_ffi_txn_commit(uint64_t handle, char** errOut);
void badger_ffi_txn_discard(uint64_t handle);
int32_t badger_ffi_txn_iterator(
	uint64_t handle,
	uint8_t* prefix,
	size_t prefixLen,
	uint8_t* start,
	size_t startLen,
	uint8_t* end,
	size_t endLen,
	uint8_t reverse,
	uint8_t keysOnly,
	uint64_t* outHandle,
	char** errOut
);

int32_t badger_ffi_iter_next(uint64_t handle, uint8_t* outHas, char** errOut);
int32_t badger_ffi_iter_key(uint64_t handle, uint8_t** outKey, size_t* outKeyLen, char** errOut);
int32_t badger_ffi_iter_value(uint64_t handle, uint8_t** outValue, size_t* outValueLen, char** errOut);
int32_t badger_ffi_iter_seek(uint64_t handle, uint8_t* key, size_t keyLen, uint8_t* outHas, char** errOut);
int32_t badger_ffi_iter_reset(uint64_t handle, char** errOut);
int32_t badger_ffi_iter_close(uint64_t handle, char** errOut);
*/
import "C"

import (
	"errors"
	"unsafe"

	badgerds "github.com/dgraph-io/badger/v4"
	"github.com/sourcenetwork/corekv"
)

type ffiOpenOptions struct {
	InMemory         bool
	EncryptionKey    []byte
	IndexCacheSize   int64
	ValueLogFileSize int64
}

type Options = badgerds.Options

type ffiStatus int32

const (
	ffiStatusOK ffiStatus = iota
	ffiStatusInvalidArgument
	ffiStatusInvalidHandle
	ffiStatusNotFound
	ffiStatusEmptyKey
	ffiStatusDiscardedTxn
	ffiStatusDBClosed
	ffiStatusTxnConflict
	ffiStatusReadOnlyTxn
	ffiStatusInvalidEncryptionKey
	ffiStatusInvalidValueLogSize
	ffiStatusUnknown
)

func ffiOpen(path string, opts ffiOpenOptions) (uint64, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	keyPtr, keyLen := ffiBytesArg(opts.EncryptionKey)
	var handle C.uint64_t
	var errPtr *C.char

	status := ffiStatus(C.badger_ffi_db_open(
		cPath,
		cPath,
		ffiBool(opts.InMemory),
		keyPtr,
		keyLen,
		C.longlong(opts.IndexCacheSize),
		C.longlong(opts.ValueLogFileSize),
		&handle,
		&errPtr,
	))
	if err := ffiStatusErr(status, errPtr); err != nil {
		return 0, err
	}

	return uint64(handle), nil
}

func compatOptions(opts Options) ffiOpenOptions {
	return ffiOpenOptions{
		InMemory:         opts.InMemory,
		EncryptionKey:    opts.EncryptionKey,
		IndexCacheSize:   opts.IndexCacheSize,
		ValueLogFileSize: opts.ValueLogFileSize,
	}
}

func ffiDBClose(handle uint64) error {
	var errPtr *C.char
	status := ffiStatus(C.badger_ffi_db_close(C.uint64_t(handle), &errPtr))
	return ffiStatusErr(status, errPtr)
}

func ffiDBDropAll(handle uint64) error {
	var errPtr *C.char
	status := ffiStatus(C.badger_ffi_db_drop_all(C.uint64_t(handle), &errPtr))
	return ffiStatusErr(status, errPtr)
}

func ffiDBNewTxn(handle uint64, readOnly bool) (uint64, error) {
	var txnHandle C.uint64_t
	var errPtr *C.char
	status := ffiStatus(C.badger_ffi_db_new_txn(C.uint64_t(handle), ffiBool(readOnly), &txnHandle, &errPtr))
	if err := ffiStatusErr(status, errPtr); err != nil {
		return 0, err
	}

	return uint64(txnHandle), nil
}

func ffiTxnGet(handle uint64, key []byte) ([]byte, error) {
	keyPtr, keyLen := ffiBytesArg(key)
	var valuePtr *C.uint8_t
	var valueLen C.size_t
	var errPtr *C.char
	status := ffiStatus(C.badger_ffi_txn_get(C.uint64_t(handle), keyPtr, keyLen, &valuePtr, &valueLen, &errPtr))
	if err := ffiStatusErr(status, errPtr); err != nil {
		return nil, err
	}

	return ffiBytesResult(valuePtr, valueLen), nil
}

func ffiTxnHas(handle uint64, key []byte) (bool, error) {
	keyPtr, keyLen := ffiBytesArg(key)
	var has C.uint8_t
	var errPtr *C.char
	status := ffiStatus(C.badger_ffi_txn_has(C.uint64_t(handle), keyPtr, keyLen, &has, &errPtr))
	if err := ffiStatusErr(status, errPtr); err != nil {
		return false, err
	}

	return has != 0, nil
}

func ffiTxnSet(handle uint64, key, value []byte) error {
	keyPtr, keyLen := ffiBytesArg(key)
	valuePtr, valueLen := ffiBytesArg(value)
	var errPtr *C.char
	status := ffiStatus(C.badger_ffi_txn_set(C.uint64_t(handle), keyPtr, keyLen, valuePtr, valueLen, &errPtr))
	return ffiStatusErr(status, errPtr)
}

func ffiTxnDelete(handle uint64, key []byte) error {
	keyPtr, keyLen := ffiBytesArg(key)
	var errPtr *C.char
	status := ffiStatus(C.badger_ffi_txn_delete(C.uint64_t(handle), keyPtr, keyLen, &errPtr))
	return ffiStatusErr(status, errPtr)
}

func ffiTxnCommit(handle uint64) error {
	var errPtr *C.char
	status := ffiStatus(C.badger_ffi_txn_commit(C.uint64_t(handle), &errPtr))
	return ffiStatusErr(status, errPtr)
}

func ffiTxnDiscard(handle uint64) {
	if handle == 0 {
		return
	}
	C.badger_ffi_txn_discard(C.uint64_t(handle))
}

func ffiTxnIterator(handle uint64, opts corekv.IterOptions) (uint64, error) {
	prefixPtr, prefixLen := ffiBytesArg(opts.Prefix)
	startPtr, startLen := ffiBytesArg(opts.Start)
	endPtr, endLen := ffiBytesArg(opts.End)
	var iterHandle C.uint64_t
	var errPtr *C.char
	status := ffiStatus(C.badger_ffi_txn_iterator(
		C.uint64_t(handle),
		prefixPtr,
		prefixLen,
		startPtr,
		startLen,
		endPtr,
		endLen,
		ffiBool(opts.Reverse),
		ffiBool(opts.KeysOnly),
		&iterHandle,
		&errPtr,
	))
	if err := ffiStatusErr(status, errPtr); err != nil {
		return 0, err
	}

	return uint64(iterHandle), nil
}

func ffiIterNext(handle uint64) (bool, error) {
	var has C.uint8_t
	var errPtr *C.char
	status := ffiStatus(C.badger_ffi_iter_next(C.uint64_t(handle), &has, &errPtr))
	if err := ffiStatusErr(status, errPtr); err != nil {
		return false, err
	}

	return has != 0, nil
}

func ffiIterKey(handle uint64) ([]byte, error) {
	var keyPtr *C.uint8_t
	var keyLen C.size_t
	var errPtr *C.char
	status := ffiStatus(C.badger_ffi_iter_key(C.uint64_t(handle), &keyPtr, &keyLen, &errPtr))
	if err := ffiStatusErr(status, errPtr); err != nil {
		return nil, err
	}

	return ffiBytesResult(keyPtr, keyLen), nil
}

func ffiIterValue(handle uint64) ([]byte, error) {
	var valuePtr *C.uint8_t
	var valueLen C.size_t
	var errPtr *C.char
	status := ffiStatus(C.badger_ffi_iter_value(C.uint64_t(handle), &valuePtr, &valueLen, &errPtr))
	if err := ffiStatusErr(status, errPtr); err != nil {
		return nil, err
	}

	return ffiBytesResult(valuePtr, valueLen), nil
}

func ffiIterSeek(handle uint64, key []byte) (bool, error) {
	keyPtr, keyLen := ffiBytesArg(key)
	var has C.uint8_t
	var errPtr *C.char
	status := ffiStatus(C.badger_ffi_iter_seek(C.uint64_t(handle), keyPtr, keyLen, &has, &errPtr))
	if err := ffiStatusErr(status, errPtr); err != nil {
		return false, err
	}

	return has != 0, nil
}

func ffiIterReset(handle uint64) error {
	var errPtr *C.char
	status := ffiStatus(C.badger_ffi_iter_reset(C.uint64_t(handle), &errPtr))
	return ffiStatusErr(status, errPtr)
}

func ffiIterClose(handle uint64) error {
	if handle == 0 {
		return nil
	}

	var errPtr *C.char
	status := ffiStatus(C.badger_ffi_iter_close(C.uint64_t(handle), &errPtr))
	if status == ffiStatusInvalidHandle {
		_ = ffiCString(errPtr)
		return nil
	}
	return ffiStatusErr(status, errPtr)
}

func ffiStatusErr(status ffiStatus, errPtr *C.char) error {
	message := ffiCString(errPtr)
	if status == ffiStatusOK {
		return nil
	}

	switch status {
	case ffiStatusNotFound:
		return corekv.ErrNotFound
	case ffiStatusEmptyKey:
		return corekv.ErrEmptyKey
	case ffiStatusDiscardedTxn:
		return corekv.ErrDiscardedTxn
	case ffiStatusDBClosed:
		return corekv.ErrDBClosed
	case ffiStatusTxnConflict:
		return corekv.ErrTxnConflict
	case ffiStatusReadOnlyTxn:
		return corekv.ErrReadOnlyTxn
	case ffiStatusInvalidEncryptionKey, ffiStatusInvalidValueLogSize, ffiStatusInvalidArgument, ffiStatusInvalidHandle, ffiStatusUnknown:
		if message == "" {
			return errors.New("badger ffi error")
		}
		return errors.New(message)
	default:
		if message == "" {
			return errors.New("unknown badger ffi status")
		}
		return errors.New(message)
	}
}

func ffiCString(ptr *C.char) string {
	if ptr == nil {
		return ""
	}
	defer C.badger_ffi_free(unsafe.Pointer(ptr))
	return C.GoString(ptr)
}

func ffiBytesArg(data []byte) (*C.uint8_t, C.size_t) {
	if len(data) == 0 {
		return nil, 0
	}

	return (*C.uint8_t)(unsafe.Pointer(unsafe.SliceData(data))), C.size_t(len(data))
}

func ffiBytesResult(ptr *C.uint8_t, n C.size_t) []byte {
	if ptr == nil || n == 0 {
		return nil
	}
	defer C.badger_ffi_free(unsafe.Pointer(ptr))

	return C.GoBytes(unsafe.Pointer(ptr), C.int(n))
}

func ffiBool(value bool) C.uint8_t {
	if value {
		return 1
	}
	return 0
}
