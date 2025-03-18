module github.com/gostaticanalysis/testutil

go 1.23.7

toolchain go1.24.1

require (
	github.com/google/go-cmp v0.7.0
	github.com/josharian/txtarfs v0.0.0-20240408113805-5dc76b8fe6bf
	github.com/newmo-oss/gotestingmock v0.1.1
	github.com/otiai10/copy v1.14.1
	github.com/tenntenn/modver v1.0.1
	github.com/tenntenn/text/transform v0.0.0-20200319021203-7eef512accb3
	golang.org/x/mod v0.24.0
	golang.org/x/text v0.23.0
	golang.org/x/tools v0.31.0
)

require (
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/josharian/mapfs v0.0.0-20210615234106-095c008854e6 // indirect
	github.com/otiai10/mint v1.6.3 // indirect
	github.com/tenntenn/golden v0.5.4 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
)

retract v0.5.1 // it has line comment bug
