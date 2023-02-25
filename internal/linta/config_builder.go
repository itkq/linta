package linta

import (
	"sort"

	"github.com/rhysd/actionlint"
)

type configBuilder struct {
	actions []stepAction
}

func (c *configBuilder) onJob(path string, j *actionlint.Job) error {
	for _, step := range j.Steps {
		execAction, ok := step.Exec.(*actionlint.ExecAction)
		if !ok {
			continue
		}

		c.actions = append(c.actions, stepAction{execAction.Uses.Value, execAction.Uses.Pos})
	}

	return nil
}

func (c *configBuilder) repositories() []string {
	mp := make(map[string]struct{})
	for _, a := range c.actions {
		mp[a.actionName()] = struct{}{}
	}

	var actions []string
	for k := range mp {
		actions = append(actions, k)
	}

	sort.Strings(actions)

	return actions
}

func newConfigBuilder() *configBuilder {
	return &configBuilder{
		actions: []stepAction{},
	}
}

func buildConfig(workflowPaths []string) (*Config, error) {
	builder := newConfigBuilder()
	walker := newWorkflowWalker()

	for _, p := range workflowPaths {
		if err := walker.walk(p, builder); err != nil {
			return nil, err
		}
	}

	config := newConfig()
	for _, r := range builder.repositories() {
		config.updateConfig(r, &PermissionConfig{})
	}

	return config, nil
}
