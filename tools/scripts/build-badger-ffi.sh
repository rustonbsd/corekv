#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/../.." && pwd)"
output_dir="$repo_root/badger_ffi/lib"

mkdir -p "$output_dir"

cd "$repo_root"
CGO_ENABLED=1 go build -buildmode=c-shared -o "$output_dir/libbadgerffi.so" ./badger_ffi/ffi/cshared

printf 'built %s\n' "$output_dir/libbadgerffi.so"
printf 'built %s\n' "$output_dir/libbadgerffi.h"