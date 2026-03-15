package pk

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// ConventionalCommitTypes is the set of allowed conventional commit types.
// Used by the -c builtin and the pr.yml workflow template.
var ConventionalCommitTypes = []string{
	"build", "chore", "ci", "docs", "feat", "fix",
	"merge", "perf", "refactor", "revert", "style", "test", "wip",
}

var (
	commitRegex     *regexp.Regexp
	commitRegexOnce sync.Once
)

func conventionalCommitRegex() *regexp.Regexp {
	commitRegexOnce.Do(func() {
		types := strings.Join(ConventionalCommitTypes, "|")
		pattern := fmt.Sprintf(`^(%s)(\(.+\))?!?: (?:[^A-Z]).+$`, types)
		commitRegex = regexp.MustCompile(pattern)
	})
	return commitRegex
}

// ValidateCommitMessage validates a single commit message first line against
// the conventional commits format. Returns nil if valid.
func ValidateCommitMessage(msg string) error {
	// Use only the first line.
	firstLine, _, _ := strings.Cut(msg, "\n")
	firstLine = strings.TrimSpace(firstLine)

	if firstLine == "" {
		return fmt.Errorf("empty commit message")
	}

	// Skip merge commits.
	if strings.HasPrefix(firstLine, "Merge ") {
		return nil
	}

	if !conventionalCommitRegex().MatchString(firstLine) {
		return commitError(firstLine)
	}

	return nil
}

// commitError returns a descriptive error for an invalid commit message.
func commitError(firstLine string) error {
	// Check if it has a known type prefix at all.
	for _, t := range ConventionalCommitTypes {
		if !strings.HasPrefix(firstLine, t) {
			continue
		}
		// Ensure the type is followed by a valid separator: "(", "!", or ":".
		rest := firstLine[len(t):]
		if len(rest) == 0 || (rest[0] != '(' && rest[0] != '!' && rest[0] != ':') {
			continue
		}
		// Has the right type — check if description starts with uppercase.
		if _, desc, found := strings.Cut(firstLine, ": "); found {
			if len(desc) > 0 && desc[0] >= 'A' && desc[0] <= 'Z' {
				return fmt.Errorf("description must not start with uppercase")
			}
		}
		return fmt.Errorf("invalid format (expected: type[(scope)][!]: description)")
	}
	return fmt.Errorf("type prefix required (e.g. feat, fix, chore, ...)")
}
