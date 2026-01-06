module bld-build

go 1.25.5

require (
	github.com/fredrikaverpil/bld v0.0.0
	github.com/goyek/goyek/v3 v3.0.1
	github.com/goyek/x v0.4.0
)

require (
	github.com/fatih/color v1.18.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/mod v0.31.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
)

// Use local bld code for dogfooding during development.
// This is specific to the bld repo itself - user projects don't have this.
replace github.com/fredrikaverpil/bld => ../
