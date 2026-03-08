package claude

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/skillvalidator"
)

// SkillValidatorFlags holds flags for the SkillValidator task.
type SkillValidatorFlags struct {
	Path string `flag:"path" usage:"path to skills directory"`
}

// SkillValidator validates Agent Skill packages using skill-validator.
var SkillValidator = &pk.Task{
	Name:  "claude-skill-validator",
	Usage: "validate Claude skill packages",
	Flags: SkillValidatorFlags{Path: ".claude/skills"},
	Body:  pk.Serial(skillvalidator.Install, skillValidatorCmd()),
}

func skillValidatorCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		f := pk.GetFlags[SkillValidatorFlags](ctx)
		return pk.Exec(ctx, skillvalidator.Name, "check", f.Path)
	})
}
