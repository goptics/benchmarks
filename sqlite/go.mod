module github.com/goptics/benchmarks/sqlite

go 1.26.0

require (
	github.com/goptics/sqliteq v0.3.0
	github.com/goptics/varmq v1.10.0
	github.com/mattn/go-sqlite3 v1.14.28
	github.com/mikestefanello/backlite v0.8.0
	github.com/riverqueue/river v0.49.0
	github.com/riverqueue/river/riverdriver/riversqlite v0.49.0
	maragu.dev/goqite v0.4.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/riverqueue/river/riverdriver v0.49.0 // indirect
	github.com/riverqueue/river/rivershared v0.49.0 // indirect
	github.com/riverqueue/river/rivertype v0.49.0 // indirect
	github.com/tidwall/gjson v1.19.0 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	golang.org/x/sync v0.23.0 // indirect
)

replace github.com/goptics/varmq => ../../varmq
