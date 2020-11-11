package core

import (
	"context"

	"github.com/BitrSKy/multi_step_loader/model"
)

type ILoader interface {
	StartLoadData(context.Context, []*model.Item) error

	SetDataToItems(context.Context, []*model.Item) error

	Name() string
}

type ILoaderManage interface{}
