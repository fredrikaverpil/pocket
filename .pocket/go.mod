module pocket

go 1.25.5

require github.com/fredrikaverpil/pocket v0.0.0


// Use local pocket code for dogfooding during development.
// This is specific to the pocket repo itself - user projects don't have this.
replace github.com/fredrikaverpil/pocket => ../
