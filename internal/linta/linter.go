package linta

import (
	"fmt"
	"strings"

	"github.com/rhysd/actionlint"
)

type LintErr struct {
	Message  string `json:"message"`
	Filepath string `json:"filepath,omitempty"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

func (e LintErr) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.Filepath, e.Line, e.Column, e.Message)
}

func newLintErr(message, path string, pos *actionlint.Pos) LintErr {
	e := LintErr{
		Message:  message,
		Filepath: path,
	}
	if pos != nil {
		e.Line = pos.Line
		e.Column = pos.Col
	}

	return e
}

type linter struct {
	config  *Config
	checker *checker
}

func newLinter(config *Config) *linter {
	return &linter{
		config:  config,
		checker: newChecker(config),
	}
}

func (l *linter) Errors() []LintErr {
	return l.checker.errs
}

func (l *linter) Lint(path string) error {
	walker := newWorkflowWalker()
	if err := walker.walk(path, l.checker); err != nil {
		return err
	}

	return nil
}

type checker struct {
	config *Config
	errs   []LintErr
}

func newChecker(config *Config) *checker {
	return &checker{
		config: config,
		errs:   []LintErr{},
	}
}

func (c *checker) onJob(path string, job *actionlint.Job) error {
	currentPermissions := newJobPermissions()
	if job.Permissions != nil && job.Permissions.Scopes != nil {
		currentPermissions = loadJobPermissions(job.Permissions.Scopes)
	}

	actions := c.collectUsedActions(job)
	derivedPermissions := c.deriveJobPermissions(actions)

	excessivePermissions := currentPermissions.excessivePermissions(*derivedPermissions)
	if len(excessivePermissions) > 0 {
		for k, p := range excessivePermissions {
			c.errs = append(c.errs, newLintErr(fmt.Sprintf("job %s has excessive permission: %s:%s", job.ID.Value, k, p), path, p.pos))
		}
	}

	if currentPermissions.isEmpty() {
		currentPermissions.setDefault()
	}
	insufficientPermissions := currentPermissions.insufficientPermissions(*derivedPermissions)
	if len(insufficientPermissions) > 0 {
		for k, p := range insufficientPermissions {
			c.errs = append(c.errs, newLintErr(fmt.Sprintf("job %s has insufficient permission: %s:%s", job.ID.Value, k, p), path, p.pos))
		}
	}

	return nil
}

func (c *checker) collectUsedActions(job *actionlint.Job) []stepAction {
	var actions []stepAction

	for _, step := range job.Steps {
		execAction, ok := step.Exec.(*actionlint.ExecAction)
		if !ok {
			continue
		}

		actions = append(actions, stepAction{value: execAction.Uses.Value, pos: execAction.Uses.Pos})
	}

	return actions
}

func (c *checker) deriveJobPermissions(stepActions []stepAction) *jobPermissions {
	derivedPermissions := newJobPermissions()

	for _, action := range stepActions {
		c := c.config.getPermissionConfig(action.actionName())
		if c != nil {
			for k, v := range *c {
				derivedPermissions.add(k, action.actionName(), newJobPermissionsFromString(v), action.pos)
			}
		}
	}

	return derivedPermissions
}

// https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#jobsjob_idstepsuses
type stepAction struct {
	value string
	pos   *actionlint.Pos
}

func (a stepAction) String() string {
	return string(a.value)
}

func (a stepAction) actionName() string {
	return strings.Split(a.value, "@")[0]
}

type jobPermissions map[string]jobPermission

func (p jobPermissions) isEmpty() bool {
	return len(p) == 0
}

func (p jobPermissions) setDefault() {
	p["contents"] = jobPermission{JobPermissionScopeRead, "", nil}
}

type jobPermission struct {
	scope      jobPermissionScope
	repository string
	pos        *actionlint.Pos
}

func (p jobPermission) String() string {
	if p.repository != "" {
		return fmt.Sprintf("%s (required by %s)", p.scope, p.repository)
	}

	return p.scope.String()
}

func newJobPermissions() *jobPermissions {
	p := make(jobPermissions)
	return &p
}

func loadJobPermissions(scopes map[string]*actionlint.PermissionScope) *jobPermissions {
	p := make(jobPermissions)

	for _, scope := range scopes {
		p[scope.Name.Value] = jobPermission{newJobPermissionsFromString(scope.Value.Value), "", scope.Value.Pos}
	}

	return &p
}

func (p jobPermissions) add(key, repository string, scope jobPermissionScope, pos *actionlint.Pos) {
	curr, ok := p[key]
	if !ok {
		p[key] = jobPermission{scope, repository, pos}
	}

	if curr.scope.lowerThan(scope) {
		p[key] = jobPermission{scope, repository, pos}
	}
}

func (p jobPermissions) excessivePermissions(q jobPermissions) jobPermissions {
	r := make(jobPermissions)

	for k, v := range p {
		_, ok := q[k]
		if !ok {
			r.add(k, v.repository, v.scope, v.pos)
		}
	}

	for k, v := range q {
		pv, ok := p[k]
		if !ok {
			continue
		}

		if v.scope.lowerThan(pv.scope) {
			r.add(k, pv.repository, pv.scope, pv.pos)
		}
	}

	return r
}

func (p jobPermissions) insufficientPermissions(q jobPermissions) jobPermissions {
	r := make(jobPermissions)

	for k, v := range q {
		_, ok := p[k]
		if !ok {
			r.add(k, v.repository, v.scope, v.pos)
		}
	}

	for k, v := range p {
		qv, ok := q[k]
		if !ok {
			continue
		}

		if v.scope.lowerThan(qv.scope) {
			r.add(k, qv.repository, qv.scope, qv.pos)
		}
	}

	return r
}

type jobPermissionScope int

const (
	JobPermissionScopeNone  = iota
	JobPermissionScopeRead  = iota
	JobPermissionScopeWrite = iota
)

func newJobPermissionsFromString(s string) jobPermissionScope {
	switch s {
	case noneScope:
		return JobPermissionScopeNone
	case readScope:
		return JobPermissionScopeRead
	case writeScope:
		return JobPermissionScopeWrite
	}

	return JobPermissionScopeNone
}

func (s jobPermissionScope) String() string {
	switch s {
	case JobPermissionScopeNone:
		return noneScope
	case JobPermissionScopeRead:
		return readScope
	case JobPermissionScopeWrite:
		return writeScope
	}

	return noneScope
}

func (s jobPermissionScope) lowerThan(other jobPermissionScope) bool {
	return int(s) < int(other)
}
