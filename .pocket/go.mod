module pocket

go 1.25.6

require github.com/fredrikaverpil/pocket v0.0.0

require (
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/term v0.39.0 // indirect
)

// Use local pocket code for dogfooding during development.
// This is specific to the pocket repo itself - user projects don't have this.
replace github.com/fredrikaverpil/pocket => ../
