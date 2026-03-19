package pk

import (
	"bytes"

	pkrun "github.com/fredrikaverpil/pocket/pk/run"
)

// bufferedOutput captures output per-goroutine for parallel execution.
// Flushes to parent Output on completion.
type bufferedOutput struct {
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	parent *pkrun.Output
}

func newBufferedOutput(parent *pkrun.Output) *bufferedOutput {
	return &bufferedOutput{
		stdout: new(bytes.Buffer),
		stderr: new(bytes.Buffer),
		parent: parent,
	}
}

func (b *bufferedOutput) output() *pkrun.Output {
	return &pkrun.Output{
		Stdout: b.stdout,
		Stderr: b.stderr,
	}
}

func (b *bufferedOutput) flush() {
	if b.stdout.Len() > 0 {
		_, _ = b.parent.Stdout.Write(b.stdout.Bytes())
	}
	if b.stderr.Len() > 0 {
		_, _ = b.parent.Stderr.Write(b.stderr.Bytes())
	}
}
