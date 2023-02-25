package linta

import (
	"io"

	"github.com/rhysd/actionlint"
)

func parse(r io.Reader) (*actionlint.Workflow, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	workflow, errs := actionlint.Parse(b)
	if len(errs) > 0 {
		// FIXME: handle all errors
		return nil, errs[0]
	}

	return workflow, nil
}
