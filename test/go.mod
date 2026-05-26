module github.com/rustonbsd/corekv/test

go 1.25.0

require (
	github.com/rustonbsd/corekv v0.0.0
	github.com/rustonbsd/corekv/badger v0.0.0
	github.com/rustonbsd/corekv/chunk v0.0.0
	github.com/rustonbsd/corekv/leveldb v0.0.0
	github.com/rustonbsd/corekv/memory v0.0.0
	github.com/rustonbsd/corekv/namespace v0.0.0
	github.com/sourcenetwork/immutable v0.3.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/sourcenetwork/goleveldb v0.0.0-20251217012629-27249d06b81b // indirect
	github.com/tidwall/btree v1.8.1 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/rustonbsd/corekv v0.0.0 => ./..
	github.com/rustonbsd/corekv/badger v0.0.0 => ../badger
	github.com/rustonbsd/corekv/chunk v0.0.0 => ../chunk
	github.com/rustonbsd/corekv/leveldb v0.0.0 => ../leveldb
	github.com/rustonbsd/corekv/memory v0.0.0 => ../memory
	github.com/rustonbsd/corekv/namespace v0.0.0 => ../namespace
)
