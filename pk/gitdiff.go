package pk

import (
	"context"
	"regexp"
)

// runGitDiff runs `git diff --exit-code` after task execution.
// Git diff only runs when the -g flag is passed (checked via context).
// When enabled, GitDiffConfig can specify rules to skip specific tasks.
// Returns nil if git diff passes or is skipped, error if there are uncommitted changes.
func runGitDiff(ctx context.Context, cfg *GitDiffConfig, tracker *executionTracker) error {
	// Only run if -g flag was passed.
	if !gitDiffEnabledFromContext(ctx) {
		return nil
	}

	if shouldSkipGitDiff(cfg, tracker) {
		return nil
	}

	// Run git diff --exit-code (fails if there are uncommitted changes).
	Printf(ctx, ":: git-diff\n")
	return Exec(ctx, "git", "diff", "--exit-code")
}

// shouldSkipGitDiff determines if the git diff check should be skipped.
// The behavior depends on DisableByDefault:
//
//   - DisableByDefault=false (default): Git diff runs for all tasks.
//     Rules specify tasks to SKIP. Skip if ALL executed tasks match a rule.
//
//   - DisableByDefault=true: Git diff is disabled for all tasks.
//     Rules specify tasks to INCLUDE. Run if ANY executed task matches a rule.
func shouldSkipGitDiff(cfg *GitDiffConfig, tracker *executionTracker) bool {
	// If no tracker or no executions, don't skip (run git diff).
	if tracker == nil {
		return false
	}

	executed := tracker.executed()
	if len(executed) == 0 {
		return false
	}

	// If no config, use default behavior (run git diff for all).
	if cfg == nil {
		return false
	}

	// Handle DisableByDefault mode.
	if cfg.DisableByDefault {
		// Git diff disabled by default. Rules specify tasks to INCLUDE.
		// Run git diff if ANY executed task matches a rule.
		if len(cfg.Rules) == 0 {
			// No include rules = skip git diff for everything.
			return true
		}

		for _, exec := range executed {
			if matchesRule(exec, cfg.Rules) {
				// At least one task wants git diff.
				return false
			}
		}
		// No executed tasks matched include rules.
		return true
	}

	// Default mode: Git diff enabled for all. Rules specify tasks to SKIP.
	if len(cfg.Rules) == 0 {
		// No skip rules = run git diff.
		return false
	}

	// Check if ALL executed combinations match a skip rule.
	// If any execution doesn't match, we should run git diff.
	for _, exec := range executed {
		if !matchesRule(exec, cfg.Rules) {
			return false
		}
	}

	// All executed combinations matched skip rules.
	return true
}

// matchesRule checks if an executed task:path matches any rule.
func matchesRule(exec executedTaskPath, rules []GitDiffRule) bool {
	for _, rule := range rules {
		if rule.Task == nil {
			continue
		}

		// Check task name match.
		if rule.Task.Name() != exec.TaskName {
			continue
		}

		// If no paths specified, rule matches all paths.
		if len(rule.Paths) == 0 {
			return true
		}

		// Check if execution path matches any pattern.
		for _, pattern := range rule.Paths {
			matched, err := regexp.MatchString("^"+pattern, exec.Path)
			if err != nil {
				// Invalid pattern, skip this pattern.
				continue
			}
			if matched {
				return true
			}
		}
	}
	return false
}
