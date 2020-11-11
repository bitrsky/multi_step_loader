package core

import (
	"context"
	"testing"
	"time"

	"github.com/BitrSKy/multi_step_loader/loader"
	"github.com/BitrSKy/multi_step_loader/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ParallelLoaderManagerSuite struct {
	suite.Suite
}

func (s *ParallelLoaderManagerSuite) SetupSuite() {

}

func (s ParallelLoaderManagerSuite) TestParallelLoaderManager() {
	parallelLoader0 := NewParallelLoaders(time.Millisecond * 500)
	parallelLoader0.AppendLoader(loader.NewWaitLoader(5))
	parallelLoader1 := NewParallelLoaders(time.Millisecond * 500)
	parallelLoader1.AppendLoader(loader.NewWaitLoader(20))
	parallelLoader1.AppendLoader(loader.NewWaitLoader(40))
	parallelLoader2 := NewParallelLoaders(time.Millisecond * 500)
	parallelLoader2.AppendLoader(loader.NewWaitLoader(30))
	parallelLoader2.AppendLoader(loader.NewWaitLoader(10))

	loaderMgr := NewLoadManager(parallelLoader0, parallelLoader1, parallelLoader2)
	err := loaderMgr.LoadData(context.Background(), []*model.Item{})
	assert.Equal(s.T(), loader.Err, err)
}

func TestUnitParallelLoaderManager(t *testing.T) {
	s := new(ParallelLoaderManagerSuite)
	suite.Run(t, s)
}
