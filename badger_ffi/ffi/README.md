# Badger FFI Surface

This directory adds the smallest C ABI needed to build a Rust clone of corekv on top of Badger directly.

## Why this lives here

The Rust side needs Badger as the storage engine, but it also needs the same iterator and transaction behavior that the Go corekv Badger adapter relies on.

The only wrapper logic included here is the logic required to preserve those semantics when calling through FFI:

- Creating an iterator after the DB has closed returns `ErrDBClosed` instead of letting Badger panic.
- Iterator `Next`, `Seek`, `Reset`, prefix iteration, and range iteration match the Go corekv adapter behavior.
- In-memory open ignores non-empty directories so Rust can keep the same `if in_memory { ignore path }` behavior without tripping Badger's path validation.

Everything else stays at the Badger level.

## Minimal ABI

The shared library exports exactly these concepts:

- DB lifecycle: `badger_ffi_db_open`, `badger_ffi_db_close`, `badger_ffi_db_drop_all`
- Explicit transactions: `badger_ffi_db_new_txn`, `badger_ffi_txn_get`, `badger_ffi_txn_has`, `badger_ffi_txn_set`, `badger_ffi_txn_delete`, `badger_ffi_txn_commit`, `badger_ffi_txn_discard`
- Iterators from transactions: `badger_ffi_txn_iterator`, `badger_ffi_iter_next`, `badger_ffi_iter_key`, `badger_ffi_iter_value`, `badger_ffi_iter_seek`, `badger_ffi_iter_reset`, `badger_ffi_iter_close`
- Memory release for returned buffers and strings: `badger_ffi_free`

There are intentionally no DB-level `Get`, `Has`, `Set`, `Delete`, or `Iterator` exports. The Rust corekv clone can build those the same way the Go adapter does:

- store `Get` and `Has`: create a read-only transaction, call txn method, discard txn
- store `Set` and `Delete`: create a writable transaction, mutate, commit, discard txn
- store `Iterator`: create a read-only transaction, create iterator from that txn, and let Rust own both handles

That keeps the ABI smaller while still allowing a functionally identical Rust implementation.

## Open options exposed through FFI

`badger_ffi_db_open` exposes only the options you asked for:

- `dir`
- `value_dir`
- `in_memory`
- `encryption_key` with validation limited to `0` or `32` bytes
- `index_cache_size`
- `value_log_file_size`
- logger forced to `nil`

The Rust side should keep policy decisions there. For example, if encryption is enabled and you want the same performance policy as the Go caller example, Rust should set `index_cache_size = 100 << 20` before calling FFI.

## Status codes

Every exported function returns an integer status code plus an optional allocated error string.

Known statuses are mapped for the cases your Rust clone needs to reason about directly:

- `StatusOK`
- `StatusInvalidArgument`
- `StatusInvalidHandle`
- `StatusNotFound`
- `StatusEmptyKey`
- `StatusDiscardedTxn`
- `StatusDBClosed`
- `StatusTxnConflict`
- `StatusReadOnlyTxn`
- `StatusInvalidEncryptionKey`
- `StatusInvalidValueLogSize`
- `StatusUnknown`

The exact error text is also returned so Rust can surface Badger failures verbatim when needed.

## Build

Build the shared library and generated C header from the corekv repo root with:

```bash
./tools/scripts/build-badger-ffi.sh
```

That emits both files directly into `badger_ffi/lib`:

- `badger_ffi/lib/libbadgerffi.so`
- `badger_ffi/lib/libbadgerffi.h`

## Rust ownership model

The exported API uses opaque integer handles.

- A DB handle owns the underlying Badger DB.
- A txn handle owns a Badger transaction until `badger_ffi_txn_discard`.
- An iterator handle owns a Badger iterator until `badger_ffi_iter_close`.
- Buffers returned from `get`, `key`, and `value`, and error strings returned through `errOut`, must be released with `badger_ffi_free`.

The iterator does not own the transaction handle. Rust should keep the transaction alive for at least as long as the iterator exists.