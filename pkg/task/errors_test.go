package task

import (
	"github.com/data-preservation-programs/RetrievalBot/pkg/requesterror"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestResolveError(t *testing.T) {
	err := errors.New("cannot dial")
	err = requesterror.CannotConnectError{Err: err}
	err = errors.Wrap(err, "failed to check if provider is boost")
	result := resolveErrorResult(err)
	assert.NotNil(t, result)
}
