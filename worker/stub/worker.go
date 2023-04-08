package stub

import (
	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	"math/rand"
	"time"
)

type Worker struct{}

func (e Worker) DoWork(_ task.Task) (*task.RetrievalResult, error) {
	//nolint: gosec
	return task.NewSuccessfulRetrievalResult(
		time.Duration(rand.Int31()),
		int64(rand.Int31()),
		time.Duration(rand.Int31())), nil
}
