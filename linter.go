package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/rhysd/actionlint"
)

type LintErr struct {
	message string
}

func (e LintErr) Error() string {
	return e.message
}

func NewLintErr(s string) LintErr {
	return LintErr{s}
}

type Linter struct {
	config   *Config
	workflow *actionlint.Workflow
	errs     []LintErr
}

func newLinter(config *Config, workflow *actionlint.Workflow) *Linter {
	return &Linter{
		config:   config,
		workflow: workflow,
		errs:     []LintErr{},
	}
}

func (l *Linter) Conclude() bool {
	if len(l.errs) > 0 {
		for _, e := range l.errs {
			fmt.Fprint(os.Stderr, e)
		}

		return false
	}

	return true
}

func (l *Linter) Lint() error {
	if l.workflow != nil && l.workflow.Jobs != nil {
		for _, job := range l.workflow.Jobs {
			if err := l.lintJob(job); err != nil {
				return err
			}
		}
	}

	return nil
}

func (l *Linter) lintJob(job *actionlint.Job) error {
	if job.WorkflowCall != nil && job.WorkflowCall.Uses != nil {
		if err := l.lintWorkflowCall(job, job.WorkflowCall.Uses.Value); err != nil {
			return err
		}
	}

	currentPermissions := NewJobPermissions()
	if job.Permissions != nil && job.Permissions.Scopes != nil {
		currentPermissions = LoadJobPermissions(job.Permissions.Scopes)
	}

	actions := l.collectUsedActions(job)
	derivedPermissions := l.deriveJobPermissions(actions)

	excessivePermissions := currentPermissions.ExcessivePermissions(*derivedPermissions)
	if len(excessivePermissions) > 0 {
		for k, p := range excessivePermissions {
			l.errs = append(l.errs, NewLintErr(fmt.Sprintf("excessive permission (job:%s): %s:%s\n", job.ID.Value, k, p)))
		}
	}
	insufficientPermissions := currentPermissions.InsufficientPermissions(*derivedPermissions)
	if len(insufficientPermissions) > 0 {
		for k, p := range insufficientPermissions {
			l.errs = append(l.errs, NewLintErr(fmt.Sprintf("insufficient permission (job:%s): %s:%s\n", job.ID.Value, k, p)))
		}
	}

	return nil
}

func (l *Linter) lintWorkflowCall(job *actionlint.Job, reusableWorkflow string) error {
	f, err := os.Open(reusableWorkflow)
	if err != nil {
		return err
	}
	defer f.Close()

	wf, err := parse(f)
	if err != nil {
		return err
	}

	reusableWorkflowLinter := newLinter(l.config, wf)
	if err := reusableWorkflowLinter.Lint(); err != nil {
		return err
	}

	l.errs = append(l.errs, reusableWorkflowLinter.errs...)

	return nil
}

func (l *Linter) collectUsedActions(job *actionlint.Job) []StepAction {
	var actions []StepAction

	for _, step := range job.Steps {
		execAction, ok := step.Exec.(*actionlint.ExecAction)
		if !ok {
			continue
		}

		actions = append(actions, StepAction{value: execAction.Uses.Value, pos: execAction.Uses.Pos})
	}

	return actions
}

func (l *Linter) deriveJobPermissions(stepActions []StepAction) *JobPermissions {
	derivedPermissions := NewJobPermissions()

	for _, action := range stepActions {
		c := l.config.GetRepositoryConfig(action.ActionName())
		if c != nil {
			for k, v := range *c.Permissions {
				derivedPermissions.Add(k, c.Repository, NewJobPermissionsFromString(v))
			}
		}
	}

	return derivedPermissions
}

// https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#jobsjob_idstepsuses
type StepAction struct {
	value string
	pos   *actionlint.Pos
}

func (a StepAction) String() string {
	return string(a.value)
}

func (a StepAction) ActionName() string {
	return strings.Split(a.value, "@")[0]
}

type JobPermissions map[string]JobPermission

type JobPermission struct {
	scope      JobPermissionScope
	repository string
}

func (p JobPermission) String() string {
	if p.repository != "" {
		return fmt.Sprintf("%s (%s)", p.scope, p.repository)
	}

	return p.scope.String()
}

func NewJobPermissions() *JobPermissions {
	p := make(JobPermissions)
	return &p
}

func LoadJobPermissions(scopes map[string]*actionlint.PermissionScope) *JobPermissions {
	p := make(JobPermissions)

	for _, scope := range scopes {
		p[scope.Name.Value] = JobPermission{NewJobPermissionsFromString(scope.Value.Value), ""}
	}

	return &p
}

func (p JobPermissions) Add(key, repository string, scope JobPermissionScope) {
	curr, ok := p[key]
	if !ok {
		p[key] = JobPermission{scope, repository}
	}

	if curr.scope.LowerThan(scope) {
		p[key] = JobPermission{scope, repository}
	}
}

func (p JobPermissions) EqualsTo(q JobPermissions) bool {
	if len(p) != len(q) {
		return false
	}

	for k, v := range p {
		qv, ok := q[k]
		if !ok {
			return false
		}
		if v != qv {
			return false
		}
	}

	for k, v := range q {
		pv, ok := p[k]
		if !ok {
			return false
		}

		if v != pv {
			return false
		}
	}

	return true
}

func (p JobPermissions) ExcessivePermissions(q JobPermissions) JobPermissions {
	r := make(JobPermissions)

	for k, v := range p {
		_, ok := q[k]
		if !ok {
			r.Add(k, v.repository, v.scope)
		}
	}

	for k, v := range q {
		pv, ok := p[k]
		if !ok {
			continue
		}

		if v.scope.LowerThan(pv.scope) {
			r.Add(k, pv.repository, pv.scope)
		}
	}

	return r
}

func (p JobPermissions) InsufficientPermissions(q JobPermissions) JobPermissions {
	r := make(JobPermissions)

	for k, v := range q {
		_, ok := p[k]
		if !ok {
			r.Add(k, v.repository, v.scope)
		}
	}

	for k, v := range p {
		qv, ok := q[k]
		if !ok {
			continue
		}

		if v.scope.LowerThan(qv.scope) {
			r.Add(k, qv.repository, qv.scope)
		}
	}

	return r
}

type JobPermissionScope int

const (
	JobPermissionScopeNone  = iota
	JobPermissionScopeRead  = iota
	JobPermissionScopeWrite = iota
)

func NewJobPermissionsFromString(s string) JobPermissionScope {
	switch s {
	case NoScope:
		return JobPermissionScopeNone
	case ReadScope:
		return JobPermissionScopeRead
	case WriteScope:
		return JobPermissionScopeWrite
	}

	return JobPermissionScopeNone
}

func (s JobPermissionScope) String() string {
	switch s {
	case JobPermissionScopeNone:
		return NoScope
	case JobPermissionScopeRead:
		return ReadScope
	case JobPermissionScopeWrite:
		return WriteScope
	}

	return NoScope
}

func (s JobPermissionScope) LowerThan(other JobPermissionScope) bool {
	return int(s) < int(other)
}
