package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"tower/model"

	"github.com/golang-jwt/jwt/v4"
)

type ContextKey string

const (
	UserIdKey      ContextKey = "userId"
	UsernameKey    ContextKey = "username"
	RoleKey        ContextKey = "role"
	WorkspaceIdKey ContextKey = "workspaceId"

	// 特殊 workspaceId 值：跨工作空间聚合（仅管理员）
	WorkspaceAll = ""
)

// userWorkspaceCache 缓存 userId -> 用户访问的 workspaceIds，TTL 60s
type userWorkspaceCache struct {
	mu    sync.RWMutex
	items map[string]userWorkspaceEntry
}

type userWorkspaceEntry struct {
	wsIds  map[string]struct{}
	role   string
	expire time.Time
}

func (c *userWorkspaceCache) get(uid string) (userWorkspaceEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.items[uid]
	if !ok || time.Now().After(e.expire) {
		return userWorkspaceEntry{}, false
	}
	return e, true
}

func (c *userWorkspaceCache) set(uid string, e userWorkspaceEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[uid] = e
}

type AuthMiddleware struct {
	AccessSecret string
	UserModel    *model.UserModel
	cache        *userWorkspaceCache
	// allowTokenInQuery 控制是否允许通过 ?token= 传递 JWT（用于 SSE/WS）
	allowTokenInQuery func(*http.Request) bool
}

// NewAuthMiddleware 创建认证中间件
// userModel 必传：用于校验 workspaceId 归属（防止越权）
func NewAuthMiddleware(accessSecret string, userModel *model.UserModel) *AuthMiddleware {
	return &AuthMiddleware{
		AccessSecret: accessSecret,
		UserModel:    userModel,
		cache:        &userWorkspaceCache{items: make(map[string]userWorkspaceEntry)},
		allowTokenInQuery: func(r *http.Request) bool {
			// 仅 SSE/WebSocket 等无法自定义 Header 的端点允许 query token
			p := r.URL.Path
			return strings.HasSuffix(p, "/stream") || strings.HasSuffix(p, "/ws") || strings.HasSuffix(p, "/sse")
		},
	}
}

func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractToken(r, m.allowTokenInQuery)
		if tokenStr == "" {
			unauthorized(w, "未提供认证信息")
			return
		}

		// 验证Token（强制校验签名算法为HMAC，防止算法混淆攻击）
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(m.AccessSecret), nil
		})
		if err != nil || !token.Valid {
			unauthorized(w, "Token无效或已过期")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			unauthorized(w, "Token解析失败")
			return
		}

		// 必备字段校验：缺失或类型不符即拒绝（修复 #4）
		userId, ok := claims["userId"].(string)
		if !ok || userId == "" {
			unauthorized(w, "Token缺少userId")
			return
		}
		username, _ := claims["username"].(string)
		role, _ := claims["role"].(string)
		if role == "" {
			role = "user"
		}

		// 工作空间归属校验（修复 #1）
		reqWs := strings.TrimSpace(r.Header.Get("X-Workspace-Id"))
		// 前端历史值 "all" 视为跨工作空间请求
		if reqWs == "all" {
			reqWs = WorkspaceAll
		}
		effectiveWs, allowed := m.resolveWorkspace(r.Context(), userId, role, reqWs)
		if !allowed {
			forbidden(w, "无权访问该工作空间")
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIdKey, userId)
		ctx = context.WithValue(ctx, UsernameKey, username)
		ctx = context.WithValue(ctx, RoleKey, role)
		ctx = context.WithValue(ctx, WorkspaceIdKey, effectiveWs)

		next(w, r.WithContext(ctx))
	}
}

// extractToken 从 Authorization Header 或受限的 query 参数提取 JWT
func extractToken(r *http.Request, allowQuery func(*http.Request) bool) string {
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}
	if allowQuery != nil && allowQuery(r) {
		return r.URL.Query().Get("token")
	}
	return ""
}

// resolveWorkspace 校验请求的 workspaceId 是否在用户授权列表内，并返回有效值
// 规则：
//   - admin/superadmin：允许任意 workspaceId 与 "全部"（空字符串）
//   - 普通用户：reqWs 必须在 WorkspaceIds 中；reqWs 为空时回退到首个授权 workspace
func (m *AuthMiddleware) resolveWorkspace(ctx context.Context, userId, role, reqWs string) (string, bool) {
	if role == "admin" || role == "superadmin" {
		return reqWs, true
	}

	entry, ok := m.cache.get(userId)
	if !ok {
		if m.UserModel == nil {
			// 未注入 UserModel 时降级为放行（保持向后兼容），但记录隐患
			return reqWs, true
		}
		u, err := m.UserModel.FindById(ctx, userId)
		if err != nil || u == nil {
			return "", false
		}
		wsSet := make(map[string]struct{}, len(u.WorkspaceIds))
		for _, w := range u.WorkspaceIds {
			wsSet[w] = struct{}{}
		}
		entry = userWorkspaceEntry{
			wsIds:  wsSet,
			role:   u.Role,
			expire: time.Now().Add(60 * time.Second),
		}
		m.cache.set(userId, entry)
	}

	if reqWs == WorkspaceAll {
		// 普通用户不允许跨工作空间聚合：选择首个授权 workspace
		for w := range entry.wsIds {
			return w, true
		}
		return "", false
	}
	if _, exist := entry.wsIds[reqWs]; exist {
		return reqWs, true
	}
	return "", false
}

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 401,
		"msg":  msg,
	})
}

func forbidden(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 403,
		"msg":  msg,
	})
}

// GetUserId 从Context获取用户ID
func GetUserId(ctx context.Context) string {
	if v, ok := ctx.Value(UserIdKey).(string); ok {
		return v
	}
	return ""
}

// GetUsername 从Context获取用户名
func GetUsername(ctx context.Context) string {
	if v, ok := ctx.Value(UsernameKey).(string); ok {
		return v
	}
	return ""
}

// GetRole 从Context获取角色
func GetRole(ctx context.Context) string {
	if v, ok := ctx.Value(RoleKey).(string); ok {
		return v
	}
	return ""
}

// GetWorkspaceId 从Context获取工作空间ID
// 返回空字符串表示"跨工作空间"（仅管理员可达）
func GetWorkspaceId(ctx context.Context) string {
	if v, ok := ctx.Value(WorkspaceIdKey).(string); ok {
		return v
	}
	return ""
}

// RequireAdmin 管理员权限中间件，需要先经过认证中间件
func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role := GetRole(r.Context())
		if role != "admin" && role != "superadmin" {
			forbidden(w, "需要管理员权限")
			return
		}
		next(w, r)
	}
}
