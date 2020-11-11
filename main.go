package main

import (
	"context"
	"fmt"
	"time"

	"github.com/BitrSKy/multi_step_loader/core"
	"github.com/BitrSKy/multi_step_loader/loader"
	"github.com/BitrSKy/multi_step_loader/model"
)

func main() {
	stepFlow()
	stepParallel()
}

func stepFlow() {

	/*
			   	5
		    	      /   \
			     20   40
			    /       \
			   30       10
	*/
	loader5 := loader.NewWaitLoader(5)
	loader10 := loader.NewWaitLoader(10)
	loader20 := loader.NewWaitLoader(20)
	loader30 := loader.NewWaitLoader(30)
	loader40 := loader.NewWaitLoader(40)

	loaderManager := core.NewLoaderManager(100)
	loaderManager.AddLoaders(loader5, loader10, loader20, loader30, loader40)

	loaderManager.Link(loader5, loader20, loader40)
	loaderManager.Link(loader20, loader30)
	loaderManager.Link(loader40, loader10)
	fmt.Println("flow start:", time.Now().Nanosecond()/1000000)
	if err := loaderManager.LoadData(context.Background(), []*model.Item{}); err != nil {
		fmt.Println(err)
	}
	fmt.Println("flow start:", time.Now().Nanosecond()/1000000)
}

func stepParallel() {

	/*
				5
		    	      /   \
			     20   40
			    /       \
			   30       10
	*/
	parallelLoader0 := core.NewParallelLoaders(time.Millisecond * 500)
	parallelLoader0.AppendLoader(loader.NewWaitLoader(5))
	parallelLoader1 := core.NewParallelLoaders(time.Millisecond * 500)
	parallelLoader1.AppendLoader(loader.NewWaitLoader(20))
	parallelLoader1.AppendLoader(loader.NewWaitLoader(40))
	parallelLoader2 := core.NewParallelLoaders(time.Millisecond * 500)
	parallelLoader2.AppendLoader(loader.NewWaitLoader(30))
	parallelLoader2.AppendLoader(loader.NewWaitLoader(10))

	loaderMgr := core.NewLoadManager(parallelLoader0, parallelLoader1, parallelLoader2)
	fmt.Println("parallel start:", time.Now().Nanosecond()/1000000)
	if err := loaderMgr.LoadData(context.Background(), []*model.Item{}); err != nil {
		fmt.Println(err)
	}
	fmt.Println("parallel end:", time.Now().Nanosecond()/1000000)
}
