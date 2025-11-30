package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/arwen/im-server/internal/model"
	"github.com/arwen/im-server/internal/service"
	"github.com/arwen/im-server/pkg/logger"
	"go.uber.org/zap"
)

// HTTPHandler HTTP API 处理器
type HTTPHandler struct {
	userService         *service.UserService
	messageService      *service.MessageService
	conversationService *service.ConversationService
}

// NewHTTPHandler 创建 HTTP 处理器
func NewHTTPHandler(
	userService *service.UserService,
	messageService *service.MessageService,
	conversationService *service.ConversationService,
) *HTTPHandler {
	return &HTTPHandler{
		userService:         userService,
		messageService:      messageService,
		conversationService: conversationService,
	}
}

// RegisterRoutes 注册路由
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	// 健康检查
	mux.HandleFunc("/health", h.HealthCheck)

	// 认证相关
	mux.HandleFunc("/api/auth/register", h.Register)
	mux.HandleFunc("/api/auth/login", h.Login)

	// 用户相关
	mux.HandleFunc("/api/user/info/", h.HandleUserInfo)
}

// Response 统一响应格式
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// UserDTO 用户数据传输对象（匹配客户端 IMUser 格式）
type UserDTO struct {
	UserID     string `json:"userID"`
	Nickname   string `json:"nickname"`
	Avatar     string `json:"avatar"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	Gender     int    `json:"gender"`
	Birth      int64  `json:"birth"`
	Signature  string `json:"signature"`
	Extra      string `json:"extra"`
	CreateTime int64  `json:"createTime"`
	UpdateTime int64  `json:"updateTime"`
}

// toUserDTO 将 model.User 转换为 UserDTO
func toUserDTO(user *model.User) *UserDTO {
	return &UserDTO{
		UserID:     user.ID,
		Nickname:   user.Nickname,
		Avatar:     user.Avatar,
		Phone:      user.Phone,
		Email:      user.Email,
		Gender:     0, // 默认为 0（未知）
		Birth:      0,
		Signature:  "",
		Extra:      "",
		CreateTime: user.CreatedAt.Unix() * 1000, // 转换为毫秒时间戳
		UpdateTime: user.UpdatedAt.Unix() * 1000,
	}
}

// HealthCheck 健康检查
func (h *HTTPHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "OK",
		Data:    map[string]string{"status": "healthy"},
	})
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Platform string `json:"platform"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token  string   `json:"token"`
	UserID string   `json:"userID"`
	User   *UserDTO `json:"user"`
}

// Register 用户注册
func (h *HTTPHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		h.writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	if req.Nickname == "" {
		req.Nickname = req.Username
	}

	logger.Info("Register request", zap.String("username", req.Username))

	// 调用 UserService 注册
	user, err := h.userService.Register(req.Username, req.Password, req.Nickname)
	if err != nil {
		logger.Warn("Register failed", zap.String("username", req.Username), zap.Error(err))
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	logger.Info("Register success", zap.String("user_id", user.ID), zap.String("username", req.Username))

	h.writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "Success",
		Data:    toUserDTO(user),
	})
}

// Login 用户登录
func (h *HTTPHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		h.writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	if req.Platform == "" {
		req.Platform = "web"
	}

	logger.Info("Login request", zap.String("username", req.Username), zap.String("platform", req.Platform))

	// 调用 UserService 登录
	token, user, err := h.userService.Login(req.Username, req.Password, req.Platform)
	if err != nil {
		logger.Warn("Login failed", zap.String("username", req.Username), zap.Error(err))
		h.writeError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	logger.Info("Login success", zap.String("user_id", user.ID), zap.String("username", req.Username))

	h.writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "Success",
		Data: LoginResponse{
			Token:  token,
			UserID: user.ID,
			User:   toUserDTO(user),
		},
	})
}

// HandleUserInfo 处理用户信息请求（统一处理 GET 和 POST）
func (h *HTTPHandler) HandleUserInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// GET /api/user/info/{userID}
		// 从 URL 路径中提取 userID
		path := strings.TrimPrefix(r.URL.Path, "/api/user/info/")
		userID := strings.Split(path, "/")[0]

		if userID == "" || userID == "batch" {
			h.writeError(w, http.StatusBadRequest, "userID is required")
			return
		}

		h.getUserInfo(w, userID)

	} else if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/batch") {
		// POST /api/user/info/batch
		h.GetUsersInfo(w, r)
	} else {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// getUserInfo 获取单个用户信息
func (h *HTTPHandler) getUserInfo(w http.ResponseWriter, userID string) {
	logger.Info("Get user info", zap.String("userID", userID))

	// 从数据库获取用户信息
	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		// 如果用户不存在，创建一个临时用户（开发模式）
		logger.Warn("User not found, creating temp user", zap.String("userID", userID), zap.Error(err))

		// 返回一个临时用户 DTO
		tempUserDTO := &UserDTO{
			UserID:     userID,
			Nickname:   userID,
			Avatar:     "",
			Phone:      "",
			Email:      "",
			Gender:     0,
			Birth:      0,
			Signature:  "",
			Extra:      "",
			CreateTime: 0,
			UpdateTime: 0,
		}

		h.writeJSON(w, http.StatusOK, Response{
			Code:    0,
			Message: "Success",
			Data:    tempUserDTO,
		})
		return
	}

	// 转换为 UserDTO
	userDTO := toUserDTO(user)

	h.writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "Success",
		Data:    userDTO,
	})
}

// GetUsersInfoRequest 批量获取用户信息请求
type GetUsersInfoRequest struct {
	UserIDs []string `json:"userIDs"`
}

// GetUsersInfo 批量获取用户信息
func (h *HTTPHandler) GetUsersInfo(w http.ResponseWriter, r *http.Request) {
	var req GetUsersInfoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.UserIDs) == 0 {
		h.writeError(w, http.StatusBadRequest, "userIDs is required")
		return
	}

	logger.Info("Get users info", zap.Int("count", len(req.UserIDs)))

	// 从数据库批量获取用户信息
	userDTOs := make([]*UserDTO, 0, len(req.UserIDs))
	for _, userID := range req.UserIDs {
		user, err := h.userService.GetUserByID(userID)
		if err != nil {
			// 如果用户不存在，创建临时用户 DTO
			userDTOs = append(userDTOs, &UserDTO{
				UserID:     userID,
				Nickname:   userID,
				Avatar:     "",
				Phone:      "",
				Email:      "",
				Gender:     0,
				Birth:      0,
				Signature:  "",
				Extra:      "",
				CreateTime: 0,
				UpdateTime: 0,
			})
		} else {
			userDTOs = append(userDTOs, toUserDTO(user))
		}
	}

	h.writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "Success",
		Data:    userDTOs,
	})
}

// writeJSON 写 JSON 响应
func (h *HTTPHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// writeError 写错误响应
func (h *HTTPHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	h.writeJSON(w, statusCode, Response{
		Code:    statusCode,
		Message: message,
	})
}
