module pocket

go 1.25.5

require github.com/fredrikaverpil/pocket v0.0.0-20260118134119-04aa2e71d825

require golang.org/x/sync v0.19.0 // indirect

// Use local pocket code for dogfooding during development.
// This is specific to the pocket repo itself - user projects don't have this.
replace github.com/fredrikaverpil/pocket => ../
