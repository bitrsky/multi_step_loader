package loader

import (
	"context"
	"fmt"
	"time"

	"github.com/BitrSKy/multi_step_loader/model"
)

var Err error = fmt.Errorf("run err")

type WaitLoader struct {
	wait    int
	isReady bool
}

func NewWaitLoader(wait int) *WaitLoader {
	return &WaitLoader{
		wait: wait,
	}
}

func (loader *WaitLoader) StartLoadData(context.Context, []*model.Item) error {
	if loader.wait < 0 {
		return Err
	}
	time.Sleep(time.Duration(loader.wait) * time.Millisecond)
	return nil
}

func (loader *WaitLoader) SetDataToItems(context.Context, []*model.Item) error {
	loader.isReady = true
	return nil
}

func (loader *WaitLoader) IsReady() bool {
	return loader.isReady
}

func (loader *WaitLoader) Name() string {
	return fmt.Sprintf("WaitLoader:%d", loader.wait)
}
