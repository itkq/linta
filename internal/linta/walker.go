package linta

import (
	"fmt"
	"os"

	"github.com/rhysd/actionlint"
)

type workflowProcessor interface {
	onJob(string, *actionlint.Job) error
}

type workflowWalker struct {
	workflowCallRecords map[string]struct{}
}

func newWorkflowWalker() *workflowWalker {
	return &workflowWalker{
		workflowCallRecords: make(map[string]struct{}),
	}
}

func (r workflowWalker) shouldWalkPath(path string) bool {
	_, ok := r.workflowCallRecords[path]
	return !ok
}

func (r workflowWalker) shouldWalkWorkflowCall(c *actionlint.WorkflowCall) bool {
	if c.Uses == nil {
		return false
	}

	_, ok := r.workflowCallRecords[c.Uses.Value]
	return !ok
}

func (r workflowWalker) recordWorkflowCall(c *actionlint.WorkflowCall) error {
	if c.Uses == nil {
		return fmt.Errorf("cannot record workflow call: uses is nil")
	}

	r.workflowCallRecords[c.Uses.Value] = struct{}{}
	return nil
}

func (r workflowWalker) walk(path string, p workflowProcessor) error {
	if !r.shouldWalkPath(path) {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w, err := parse(f)
	if err != nil {
		return err
	}

	if w != nil && w.Jobs != nil {
		for _, job := range w.Jobs {
			if err := p.onJob(path, job); err != nil {
				return err
			}

			wfc := job.WorkflowCall
			if wfc != nil && r.shouldWalkWorkflowCall(wfc) {
				_, err := os.Stat(wfc.Uses.Value)
				if err != nil {
					return nil
				}
				if err := r.walk(job.WorkflowCall.Uses.Value, p); err != nil {
					return err
				}
				if err := r.recordWorkflowCall(wfc); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
