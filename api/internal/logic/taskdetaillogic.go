package logic

import (
	"context"

	"tower/api/internal/logic/common"
	"tower/api/internal/svc"
	"tower/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MainTaskDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMainTaskDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MainTaskDetailLogic {
	return &MainTaskDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MainTaskDetailLogic) MainTaskDetail(req *types.MainTaskDetailReq) (resp interface{}, err error) {
	if req.Id == "" {
		return &types.BaseResp{Code: 400, Msg: "缺少任务ID"}, nil
	}

	wsIds := common.GetWorkspaceIds(l.ctx, l.svcCtx, "all")

	for _, wsId := range wsIds {
		taskModel := l.svcCtx.GetMainTaskModel(wsId)
		task, err := taskModel.FindById(l.ctx, req.Id)
		if err != nil {
			continue
		}
		if task != nil {
			return map[string]interface{}{
				"code": 0,
				"data": task,
			}, nil
		}
	}

	return &types.BaseResp{Code: 404, Msg: "任务不存在"}, nil
}
