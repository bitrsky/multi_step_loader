package core

import (
	"context"
	"sync"
	"time"

	"github.com/BitrSKy/multi_step_loader/model"
)

// LoadManager loader管理器
type LoadManager struct {
	parallelLoaders []*ParallelLoaders
}

// NewLoadManager 构造loader管理器
func NewLoadManager(parallelLoaders ...*ParallelLoaders) *LoadManager {
	return &LoadManager{
		parallelLoaders: parallelLoaders,
	}
}

func (loadmgr *LoadManager) LoadData(ctx context.Context, items []*model.Item) error {

	for _, pLoaders := range loadmgr.parallelLoaders {
		pLoaders.LoadItemsData(ctx, items)
	}
	return nil
}

type LoaderDataInterface interface {
	// 开始加载数据
	StartLoadData(context.Context, []*model.Item) error
	// 把加载的数据赋值给Item
	SetDataToItems(context.Context, []*model.Item) error
	// 数据是否加载完成
	IsReady() bool
	// Loader 的名字
	Name() string
}

// ParallelLoaders 同一批次loader
type ParallelLoaders struct {
	loaders []LoaderDataInterface
	timeout time.Duration
}

// NewParallelLoaders 构造并发loader
func NewParallelLoaders(timeout time.Duration, loaders ...LoaderDataInterface) *ParallelLoaders {

	return &ParallelLoaders{
		loaders: loaders,
		timeout: timeout,
	}
}

// AppendLoader 增加并发loader列表中的loader
func (pLoaders *ParallelLoaders) AppendLoader(loader LoaderDataInterface) error {

	if loader != nil {
		pLoaders.loaders = append(pLoaders.loaders, loader)
	}

	return nil
}

// LoadItemsData 并发调用loader，获取文章等数据，并设置item中的属性
func (pLoaders *ParallelLoaders) LoadItemsData(ctx context.Context, items []*model.Item) error {

	var wg sync.WaitGroup
	// 并发执行同一批次loader
	for _, loader := range pLoaders.loaders {
		wg.Add(1)
		go func(loader LoaderDataInterface) {
			defer wg.Done()
			loadWithTimeout(ctx, loader, pLoaders.timeout, items)
		}(loader)
	}

	wg.Wait()
	// 串行回写数据到item中，避免并发读写map
	for _, loader := range pLoaders.loaders {
		if loader.IsReady() {
			loader.SetDataToItems(ctx, items)
		}
	}
	return nil
}

// loadWithTimeout loader超时控制
func loadWithTimeout(ctx context.Context, loader LoaderDataInterface,
	timeout time.Duration, items []*model.Item) (err error) {

	var (
		errCh = make(chan error, 1)
	)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	go func() {
		errCh <- loader.StartLoadData(ctx, items)
	}()

	select {
	case err = <-errCh:
	case <-ctx.Done():
		err = ctx.Err()
	}

	return err
}
