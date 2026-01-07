package pocket

import "fmt"

// Must panics if err is not nil.
func Must(err error) {
	if err != nil {
		panic(fmt.Sprintf("pocket: %v", err))
	}
}
