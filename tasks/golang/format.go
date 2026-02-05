package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
)

// Format formats Go code using golangci-lint fmt.
var Format = &pk.Task{
	Name:  "go-format",
	Usage: "format Go code",
	Body:  pk.Serial(golangcilint.Install, formatCmd()),
}

func formatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		args := []string{"fmt", "./..."}
		return pk.Exec(ctx, golangcilint.Name, args...)
	})
}
