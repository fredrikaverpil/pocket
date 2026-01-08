package pocket

import "github.com/goyek/goyek/v3"

// AutoTaskGroup provides a generic auto-detecting task group wrapper.
// O is the options type (e.g., golang.Options) which must implement ModuleConfig.
type AutoTaskGroup[O ModuleConfig] struct {
	name      string
	detect    func() []string
	defaults  O
	overrides map[string]O
	newGroup  func(modules map[string]O) TaskGroup

	detected map[string]O // Lazily populated.
}

// NewAutoTaskGroup creates a new auto-detecting task group.
func NewAutoTaskGroup[O ModuleConfig](
	name string,
	detect func() []string,
	defaults O,
	overrides map[string]O,
	newGroup func(modules map[string]O) TaskGroup,
) *AutoTaskGroup[O] {
	return &AutoTaskGroup[O]{
		name:      name,
		detect:    detect,
		defaults:  defaults,
		overrides: overrides,
		newGroup:  newGroup,
	}
}

func (tg *AutoTaskGroup[O]) Name() string { return tg.name }

func (tg *AutoTaskGroup[O]) doDetect() map[string]O {
	if tg.detected != nil {
		return tg.detected
	}

	paths := tg.detect()
	modules := make(map[string]O, len(paths))
	for _, p := range paths {
		if opts, ok := tg.overrides[p]; ok {
			modules[p] = opts
		} else {
			modules[p] = tg.defaults
		}
	}

	tg.detected = modules
	return modules
}

func (tg *AutoTaskGroup[O]) Modules() map[string]ModuleConfig {
	detected := tg.doDetect()
	modules := make(map[string]ModuleConfig, len(detected))
	for path, opts := range detected {
		modules[path] = opts
	}
	return modules
}

func (tg *AutoTaskGroup[O]) ForContext(context string) TaskGroup {
	detected := tg.doDetect()
	if context == "." {
		return tg.newGroup(detected)
	}
	if opts, ok := detected[context]; ok {
		return tg.newGroup(map[string]O{context: opts})
	}
	return nil
}

func (tg *AutoTaskGroup[O]) Tasks(cfg Config) []*goyek.DefinedTask {
	detected := tg.doDetect()
	return tg.newGroup(detected).Tasks(cfg)
}
