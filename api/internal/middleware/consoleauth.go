package middleware

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// ConsoleAuthMiddleware 控制台权限中间件（仅管理员可访问）
type ConsoleAuthMiddleware struct {
	RedisClient *redis.Client
}

// NewConsoleAuthMiddleware 创建控制台权限中间件
func NewConsoleAuthMiddleware(redisClient *redis.Client) *ConsoleAuthMiddleware {
	return &ConsoleAuthMiddleware{
		RedisClient: redisClient,
	}
}

// Handle 控制台权限检查处理
func (m *ConsoleAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从Context获取用户角色（需要先经过AuthMiddleware）
		role := GetRole(r.Context())
		if role != "admin" && role != "superadmin" {
			logx.Errorf("[ConsoleAuth] Access denied for non-admin user, role: %s, path: %s", role, r.URL.Path)
			consoleForbidden(w, "需要管理员权限访问控制台")
			return
		}

		// 先取出用户信息，避免在 goroutine 中访问可能被回收的请求对象（修复 #10）
		entry := consoleAccessEntry{
			userId:    GetUserId(r.Context()),
			username:  GetUsername(r.Context()),
			path:      r.URL.Path,
			method:    r.Method,
			clientIP:  getClientIPFromRequest(r),
			userAgent: r.UserAgent(),
		}
		go m.recordConsoleAccess(entry)

		next(w, r)
	}
}

// consoleAccessEntry 控制台访问审计快照
type consoleAccessEntry struct {
	userId    string
	username  string
	path      string
	method    string
	clientIP  string
	userAgent string
}

// recordConsoleAccess 记录控制台访问日志（使用值快照，避免共享 *http.Request）
func (m *ConsoleAuthMiddleware) recordConsoleAccess(e consoleAccessEntry) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logJSON, err := json.Marshal(map[string]interface{}{
		"type":      "console_access",
		"userId":    e.userId,
		"username":  e.username,
		"path":      e.path,
		"method":    e.method,
		"clientIP":  e.clientIP,
		"userAgent": e.userAgent,
		"timestamp": time.Now().UnixMilli(),
	})
	if err != nil {
		return
	}

	m.RedisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: "tower:audit:console",
		MaxLen: 10000,
		Approx: true,
		Values: map[string]interface{}{"data": string(logJSON)},
	})
}

// consoleForbidden 返回403禁止访问响应
func consoleForbidden(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 403,
		"msg":  msg,
	})
}

// getClientIPFromRequest 从请求获取客户端IP
// 安全说明：默认不信任 X-Forwarded-For / X-Real-IP（可被任意客户端伪造）。
// 仅当部署在受信反向代理后（通过环境变量 TRUST_PROXY_HEADERS=1 启用）时才采用代理头。
// 修复 #9
func getClientIPFromRequest(r *http.Request) string {
	if os.Getenv("TRUST_PROXY_HEADERS") == "1" {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			for i := 0; i < len(xff); i++ {
				if xff[i] == ',' {
					return strings.TrimSpace(xff[:i])
				}
			}
			return strings.TrimSpace(xff)
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return xri
		}
	}

	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}
	return ip
}
