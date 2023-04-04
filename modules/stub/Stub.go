package stub

import (
	"github.com/data-preservation-programs/RetrievalBot/common"
	"math/rand"
)

type Stub struct{}

func (e Stub) DoWork(_ common.Task) (common.RetrievalResult, error) {
	//nolint: gosec
	return common.RetrievalResult{
		Success:    true,
		TTFB:       rand.Int31(),
		Speed:      rand.Float32(),
		Duration:   rand.Int31(),
		Downloaded: rand.Int63(),
	}, nil
}
