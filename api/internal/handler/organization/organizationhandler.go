package organization

import (
	"net/http"

	"tower/api/internal/logic"
	"tower/api/internal/svc"
	"tower/api/internal/types"
	"tower/pkg/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// OrganizationListHandler 组织列表
func OrganizationListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PageReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.OkJson(w, &types.BaseResp{Code: 400, Msg: err.Error()})
			return
		}

		l := logic.NewOrganizationListLogic(r.Context(), svcCtx)
		resp, err := l.OrganizationList(&req)
		if err != nil {
			response.Error(w, err)
			return
		}
		httpx.OkJson(w, resp)
	}
}

// OrganizationSaveHandler 保存组织
func OrganizationSaveHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.OrganizationSaveReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.OkJson(w, &types.BaseResp{Code: 400, Msg: err.Error()})
			return
		}

		l := logic.NewOrganizationSaveLogic(r.Context(), svcCtx)
		resp, err := l.OrganizationSave(&req)
		if err != nil {
			response.Error(w, err)
			return
		}
		httpx.OkJson(w, resp)
	}
}

// OrganizationDeleteHandler 删除组织
func OrganizationDeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.OrganizationDeleteReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.OkJson(w, &types.BaseResp{Code: 400, Msg: err.Error()})
			return
		}

		l := logic.NewOrganizationDeleteLogic(r.Context(), svcCtx)
		resp, err := l.OrganizationDelete(&req)
		if err != nil {
			response.Error(w, err)
			return
		}
		httpx.OkJson(w, resp)
	}
}

// OrganizationUpdateStatusHandler 更新组织状态
func OrganizationUpdateStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.OrganizationUpdateStatusReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.OkJson(w, &types.BaseResp{Code: 400, Msg: err.Error()})
			return
		}

		l := logic.NewOrganizationUpdateStatusLogic(r.Context(), svcCtx)
		resp, err := l.OrganizationUpdateStatus(&req)
		if err != nil {
			response.Error(w, err)
			return
		}
		httpx.OkJson(w, resp)
	}
}
