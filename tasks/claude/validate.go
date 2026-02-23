package claude

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/skillvalidator"
)

// Flag names for the SkillValidator task.
const (
	FlagSkillValidatorPath = "path"
)

// SkillValidator validates Agent Skill packages using skill-validator.
var SkillValidator = &pk.Task{
	Name:  "claude-skill-validator",
	Usage: "validate Claude skill packages",
	Flags: map[string]pk.FlagDef{
		FlagSkillValidatorPath: {Default: ".claude/skills", Usage: "path to skills directory"},
	},
	Body: pk.Serial(skillvalidator.Install, skillValidatorCmd()),
}

func skillValidatorCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		path := pk.GetFlag[string](ctx, FlagSkillValidatorPath)
		return pk.Exec(ctx, skillvalidator.Name, "check", path)
	})
}
