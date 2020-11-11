package core

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/BitrSKy/multi_step_loader/model"
)

type LoaderRunner struct {
	status int32
	// = 0: 所有父任务执行完成，当前任务待执行
	// > 0: 表示还有父任务未执行
	// < 0: 当前任务在执行中

	seq    int32
	loader ILoader
	childs []*LoaderRunner
}

func (runner *LoaderRunner) IsReady() bool {
	return runner.status == 0
}

func (runner *LoaderRunner) AddChild(child *LoaderRunner) {
	if runner.childs == nil {
		runner.childs = []*LoaderRunner{}
	}
	runner.childs = append(runner.childs, child)
}

func (runner *LoaderRunner) updateStatus(num int32) {
	atomic.AddInt32(&runner.status, num)
}

func (runner *LoaderRunner) Run(ctx context.Context, items []*model.Item) (err error) {

	if err = runner.loader.StartLoadData(ctx, items); err != nil {
		return
	}
	if err = runner.loader.SetDataToItems(ctx, items); err != nil {
		return
	}
	for _, child := range runner.childs {
		child.updateStatus(runner.seq * -1)
	}
	return nil
}

var over error = fmt.Errorf("load manage run over")

type LoaderManager struct {
	loaderRunners map[ILoader]*LoaderRunner
	runners       chan *LoaderRunner
	timeOut       int64

	errMonitor chan error
	err        error
}

func NewLoaderManager(timeOut int64) *LoaderManager {
	return &LoaderManager{
		timeOut: timeOut,
	}
}

func (loadmgr *LoaderManager) run(runner *LoaderRunner, ctx context.Context, items []*model.Item) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("recover:%v", r)
			fmt.Println(err)
		}
	}()

	errCh := make(chan error)
	defer close(errCh)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("recover:%v", r)
				fmt.Println(err)
			}
		}()
		errCh <- runner.Run(ctx, items)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			loadmgr.errMonitor <- err
			break
		}
		loadmgr.runners <- runner
	case <-ctx.Done():
		fmt.Println("loader:", ctx.Err())
	}
}

func (loadmgr *LoaderManager) loop(ctx context.Context, items []*model.Item) {
	counter := len(loadmgr.loaderRunners)
	for runner := range loadmgr.runners {
		if counter--; counter == 0 {
			loadmgr.errMonitor <- over
			break
		}
		for _, child := range runner.childs {
			if child.IsReady() {
				child.updateStatus(-1)
				go loadmgr.run(child, ctx, items)
			}
		}
	}
}

func (loadmgr *LoaderManager) LoadData(ctx context.Context, items []*model.Item) error {

	tctx, tcancel := context.WithTimeout(ctx, time.Duration(loadmgr.timeOut)*time.Millisecond)
	defer tcancel()

	loadmgr.errMonitor = make(chan error)
	defer close(loadmgr.errMonitor)

	loadmgr.runners = make(chan *LoaderRunner)
	defer close(loadmgr.runners)

	// 根据完成的任务，执行其子任务
	go loadmgr.loop(tctx, items)

	// 首次将头部任务开始执行
	for _, runner := range loadmgr.loaderRunners {
		if runner.IsReady() {
			runner.updateStatus(-1)
			go loadmgr.run(runner, tctx, items)
		}
	}

	select {
	case err, isOpen := <-loadmgr.errMonitor:
		if isOpen && err != nil && err != over {
			loadmgr.err = err
		}
	case <-tctx.Done():
		loadmgr.err = tctx.Err()
	case <-ctx.Done():
		fmt.Println("ctx:", ctx.Err())
	}

	return loadmgr.err
}

func (loadmgr *LoaderManager) AddLoaders(loaders ...ILoader) {
	if loadmgr.loaderRunners == nil {
		loadmgr.loaderRunners = make(map[ILoader]*LoaderRunner)
	}
	seq := int32(len(loadmgr.loaderRunners) + 1)
	for _, loader := range loaders {
		loadmgr.loaderRunners[loader] = &LoaderRunner{loader: loader, seq: seq}
		seq++
	}
}

func (loadmgr *LoaderManager) Link(loader ILoader, childs ...ILoader) {
	father, exist := loadmgr.loaderRunners[loader]
	if !exist {
		return
	}

	for _, child := range childs {
		runner, exist := loadmgr.loaderRunners[child]
		if !exist {
			return
		}
		father.AddChild(runner)
		runner.updateStatus(father.seq)
	}
}
