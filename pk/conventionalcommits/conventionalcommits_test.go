package conventionalcommits

import (
	"strings"
	"testing"
)

func TestValidateMessage(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		wantErr bool
		errMsg  string
	}{
		// Valid messages.
		{"feat", "feat: add new feature", false, ""},
		{"fix", "fix: resolve crash", false, ""},
		{"chore", "chore: update deps", false, ""},
		{"docs", "docs: update readme", false, ""},
		{"build", "build: update makefile", false, ""},
		{"ci", "ci: add workflow", false, ""},
		{"perf", "perf: optimize query", false, ""},
		{"refactor", "refactor: simplify logic", false, ""},
		{"revert", "revert: undo change", false, ""},
		{"style", "style: fix formatting", false, ""},
		{"test", "test: add unit tests", false, ""},
		// Scoped messages.
		{"scoped feat", "feat(api): add endpoint", false, ""},
		{"scoped fix", "fix(auth): resolve token issue", false, ""},
		{"deep scope", "feat(api/v2): add endpoint", false, ""},

		// Breaking change.
		{"breaking feat", "feat!: remove deprecated api", false, ""},
		{"breaking scoped", "feat(api)!: remove endpoint", false, ""},

		// Merge commits (skipped).
		{"merge commit", "Merge branch 'main' into feature", false, ""},
		{"merge pull request", "Merge pull request #42 from org/branch", false, ""},

		// Multiline (only first line validated).
		{"multiline valid", "feat: add feature\n\ndetailed description", false, ""},
		{"multiline invalid", "Add feature\n\nfeat: this is body", true, "type prefix required"},

		// Subject length (72 char limit).
		{"exactly 72 chars", "feat: " + strings.Repeat("a", 66), false, ""},
		{"73 chars", "feat: " + strings.Repeat("a", 67), true, "subject exceeds 72 characters"},

		// Greedy scope rejected.
		{"greedy scope", "feat(a)(b): add thing", true, "invalid format"},

		// Removed types.
		{"wip rejected", "wip: work in progress", true, "type prefix required"},
		{"merge type rejected", "merge: merge branch", true, "type prefix required"},

		// Invalid messages.
		{"empty", "", true, "empty commit message"},
		{"no type", "add new feature", true, "type prefix required"},
		{"uppercase desc", "feat: Add new feature", true, "description must not start with uppercase"},
		{"uppercase fix", "fix: Fix the bug", true, "description must not start with uppercase"},
		{"no description", "feat:", true, "invalid format"},
		{"no space after colon", "feat:no space", true, "invalid format"},
		{"unknown type", "feature: add thing", true, "type prefix required"},
		{"capitalized type", "Feat: add thing", true, "type prefix required"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateMessage(tc.msg)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ValidateMessage(%q) = nil, want error containing %q", tc.msg, tc.errMsg)
				} else if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("ValidateMessage(%q) error = %q, want containing %q", tc.msg, err.Error(), tc.errMsg)
				}
			} else if err != nil {
				t.Errorf("ValidateMessage(%q) = %v, want nil", tc.msg, err)
			}
		})
	}
}

func TestTypes(t *testing.T) {
	// Verify the type list is non-empty and sorted.
	if len(Types) == 0 {
		t.Fatal("Types is empty")
	}
	for i := 1; i < len(Types); i++ {
		if Types[i] <= Types[i-1] {
			t.Errorf(
				"Types not sorted: %q should come before %q",
				Types[i], Types[i-1],
			)
		}
	}
}
