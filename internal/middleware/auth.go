package middleware

import (
	"net/http"
	"strings"

	"github.com/arwen/im-server/pkg/crypto"
)

// AuthMiddleware 认证中间件
func AuthMiddleware(jwtSecret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取Token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 解析Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// 验证Token
		claims, err := crypto.ValidateToken(token, jwtSecret)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// 将用户信息存入context（简化版本，实际应该使用context）
		r.Header.Set("X-User-ID", claims.UserID)
		r.Header.Set("X-Platform", claims.Platform)

		next(w, r)
	}
}

