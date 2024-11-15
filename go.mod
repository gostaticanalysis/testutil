module github.com/gostaticanalysis/testutil

go 1.22.9

require (
	github.com/google/go-cmp v0.6.0
	github.com/josharian/txtarfs v0.0.0-20210218200122-0702f000015a
	github.com/otiai10/copy v1.14.0
	github.com/tenntenn/modver v1.0.1
	github.com/tenntenn/text/transform v0.0.0-20200319021203-7eef512accb3
	golang.org/x/mod v0.22.0
	golang.org/x/text v0.20.0
	golang.org/x/tools v0.27.0
)

require (
	github.com/hashicorp/go-version v1.7.0 // indirect
	golang.org/x/sync v0.9.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
)

retract v0.5.1 // it has line comment bug
