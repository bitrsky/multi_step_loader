package core

import (
	"context"
	"testing"

	"github.com/BitrSKy/multi_step_loader/loader"
	"github.com/BitrSKy/multi_step_loader/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type LoaderManagerSuite struct {
	suite.Suite
}

func (s *LoaderManagerSuite) SetupSuite() {

}

func (s LoaderManagerSuite) TestLoaderManager() {
	loader5 := loader.NewWaitLoader(5)
	loader10 := loader.NewWaitLoader(10)
	loader20 := loader.NewWaitLoader(20)
	loader30 := loader.NewWaitLoader(30)
	loader40 := loader.NewWaitLoader(40)

	loaderManager := NewLoaderManager(100)
	loaderManager.AddLoaders(loader5, loader10, loader20, loader30, loader40)

	loaderManager.Link(loader5, loader20, loader40)
	loaderManager.Link(loader20, loader30)
	loaderManager.Link(loader40, loader10)

	err := loaderManager.LoadData(context.Background(), []*model.Item{})
	assert.Equal(s.T(), nil, err)
}

func (s LoaderManagerSuite) TestLoaderManagerWithError() {
	loader5 := loader.NewWaitLoader(5)
	loader10 := loader.NewWaitLoader(10)
	loader20 := loader.NewWaitLoader(-1)
	loader30 := loader.NewWaitLoader(30)
	loader40 := loader.NewWaitLoader(-1)

	loaderManager := NewLoaderManager(100)
	loaderManager.AddLoaders(loader5, loader10, loader20, loader30, loader40)

	loaderManager.Link(loader5, loader20, loader40)
	loaderManager.Link(loader20, loader30)
	loaderManager.Link(loader40, loader10)

	err := loaderManager.LoadData(context.Background(), []*model.Item{})
	assert.Equal(s.T(), loader.Err, err)
}

func (s LoaderManagerSuite) TestLoaderManagerWithTimeOut() {
	loader5 := loader.NewWaitLoader(5)
	loader10 := loader.NewWaitLoader(10)
	loader20 := loader.NewWaitLoader(20)
	loader30 := loader.NewWaitLoader(30)
	loader40 := loader.NewWaitLoader(40)

	loaderManager := NewLoaderManager(40)
	loaderManager.AddLoaders(loader5, loader10, loader20, loader30, loader40)

	loaderManager.Link(loader5, loader20, loader40)
	loaderManager.Link(loader20, loader30)
	loaderManager.Link(loader40, loader10)

	err := loaderManager.LoadData(context.Background(), []*model.Item{})
	assert.Equal(s.T(), context.DeadlineExceeded, err)
}

func TestUnitLoaderManager(t *testing.T) {
	s := new(LoaderManagerSuite)
	suite.Run(t, s)
}
