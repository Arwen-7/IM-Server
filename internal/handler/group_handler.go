package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/arwen/im-server/internal/model"
	"github.com/arwen/im-server/internal/service"
	"github.com/arwen/im-server/pkg/logger"
	"go.uber.org/zap"
)

// GroupHandler 群组 HTTP 处理器
type GroupHandler struct {
	groupService *service.GroupService
	userService  *service.UserService
}

// NewGroupHandler 创建群组处理器
func NewGroupHandler(groupService *service.GroupService, userService *service.UserService) *GroupHandler {
	return &GroupHandler{
		groupService: groupService,
		userService:  userService,
	}
}

// GroupDTO 群组数据传输对象
type GroupDTO struct {
	GroupID      string `json:"groupID"`
	GroupName    string `json:"groupName"`
	FaceURL      string `json:"faceURL"`
	OwnerUserID  string `json:"ownerUserID"`
	MemberCount  int    `json:"memberCount"`
	Introduction string `json:"introduction"`
	Notification string `json:"notification"`
	Extra        string `json:"extra"`
	Status       int    `json:"status"`
	CreateTime   int64  `json:"createTime"`
	UpdateTime   int64  `json:"updateTime"`
}

// toGroupDTO 将 model.Group 转换为 GroupDTO
func toGroupDTO(group *model.Group) *GroupDTO {
	return &GroupDTO{
		GroupID:      group.ID,
		GroupName:    group.Name,
		FaceURL:      group.Avatar,
		OwnerUserID:  group.OwnerID,
		MemberCount:  0, // TODO: 从群成员表统计
		Introduction: group.Description,
		Notification: "",
		Extra:        "",
		Status:       group.Status,
		CreateTime:   group.CreatedAt.UnixMilli(),
		UpdateTime:   group.UpdatedAt.UnixMilli(),
	}
}

// CreateGroupRequest 创建群组请求
type CreateGroupRequest struct {
	GroupName     string   `json:"groupName"`
	FaceURL       string   `json:"faceURL"`
	Introduction  string   `json:"introduction"`
	MemberUserIDs []string `json:"memberUserIDs"`
}

// UpdateGroupRequest 更新群组请求
type UpdateGroupRequest struct {
	GroupID      string `json:"groupID"`
	GroupName    string `json:"groupName"`
	FaceURL      string `json:"faceURL"`
	Introduction string `json:"introduction"`
	Notification string `json:"notification"`
}

// InviteMembersRequest 邀请成员请求
type InviteMembersRequest struct {
	UserIDs []string `json:"userIDs"`
}

// RegisterRoutes 注册路由
func (h *GroupHandler) RegisterRoutes(mux *http.ServeMux) {
	// 群组路由
	mux.HandleFunc("/api/group/create", h.AuthMiddleware(h.CreateGroup))
	mux.HandleFunc("/api/group/", h.AuthMiddleware(h.HandleGroup))
}

// HandleGroup 处理群组相关请求（路由分发）
func (h *GroupHandler) HandleGroup(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/group/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		h.writeError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// 处理不同的子路由
	switch {
	case parts[0] == "update" && r.Method == "POST":
		h.UpdateGroup(w, r)
	case parts[0] == "my" && len(parts) > 1 && parts[1] == "list" && r.Method == "GET":
		h.GetMyGroups(w, r)
	case len(parts) == 1 && r.Method == "GET":
		// GET /api/group/{groupID}
		h.GetGroup(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "join" && r.Method == "POST":
		h.JoinGroup(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "leave" && r.Method == "POST":
		h.LeaveGroup(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "invite" && r.Method == "POST":
		h.InviteMembers(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "kick" && r.Method == "POST":
		h.KickMembers(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "dismiss" && r.Method == "POST":
		h.DismissGroup(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "members" && r.Method == "GET":
		h.GetGroupMembers(w, r, parts[0])
	default:
		h.writeError(w, http.StatusNotFound, "Route not found")
	}
}

// CreateGroup 创建群组
func (h *GroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.GroupName == "" {
		h.writeError(w, http.StatusBadRequest, "groupName is required")
		return
	}

	// 从 context 获取当前用户 ID
	userID := r.Context().Value("user_id")
	if userID == nil {
		h.writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	ownerUserID := userID.(string)

	// 创建群组
	group, err := h.groupService.CreateGroup(
		r.Context(),
		ownerUserID,
		req.GroupName,
		req.FaceURL,
		req.Introduction,
		req.MemberUserIDs,
	)

	if err != nil {
		logger.Error("Failed to create group", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to create group: "+err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    toGroupDTO(group),
	})
}

// GetGroup 获取群组信息
func (h *GroupHandler) GetGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	group, err := h.groupService.GetGroup(r.Context(), groupID)
	if err != nil {
		if err == service.ErrGroupNotFound {
			h.writeError(w, http.StatusNotFound, "Group not found")
			return
		}

		logger.Error("Failed to get group", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to get group: "+err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    toGroupDTO(group),
	})
}

// UpdateGroup 更新群组信息
func (h *GroupHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	var req UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.GroupID == "" {
		h.writeError(w, http.StatusBadRequest, "groupID is required")
		return
	}

	// 构建更新参数
	updates := make(map[string]interface{})
	if req.GroupName != "" {
		updates["groupName"] = req.GroupName
	}
	if req.FaceURL != "" {
		updates["faceURL"] = req.FaceURL
	}
	if req.Introduction != "" {
		updates["introduction"] = req.Introduction
	}
	if req.Notification != "" {
		updates["notification"] = req.Notification
	}

	// 更新群组
	group, err := h.groupService.UpdateGroup(r.Context(), req.GroupID, updates)
	if err != nil {
		if err == service.ErrGroupNotFound {
			h.writeError(w, http.StatusNotFound, "Group not found")
			return
		}

		logger.Error("Failed to update group", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to update group: "+err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    toGroupDTO(group),
	})
}

// JoinGroup 加入群组
func (h *GroupHandler) JoinGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	// 从 context 获取当前用户 ID
	userID := r.Context().Value("user_id")
	if userID == nil {
		h.writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err := h.groupService.JoinGroup(r.Context(), groupID, userID.(string))
	if err != nil {
		if err == service.ErrGroupNotFound {
			h.writeError(w, http.StatusNotFound, "Group not found")
			return
		}

		if err == service.ErrAlreadyGroupMember {
			h.writeError(w, http.StatusBadRequest, "Already a group member")
			return
		}

		logger.Error("Failed to join group", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to join group: "+err.Error())
		return
	}

	// 重新获取群组信息
	group, _ := h.groupService.GetGroup(r.Context(), groupID)

	h.writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    toGroupDTO(group),
	})
}

// LeaveGroup 退出群组
func (h *GroupHandler) LeaveGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	// 从 context 获取当前用户 ID
	userID := r.Context().Value("user_id")
	if userID == nil {
		h.writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err := h.groupService.LeaveGroup(r.Context(), groupID, userID.(string))
	if err != nil {
		if err == service.ErrGroupNotFound {
			h.writeError(w, http.StatusNotFound, "Group not found")
			return
		}

		if err == service.ErrOwnerCannotLeave {
			h.writeError(w, http.StatusBadRequest, "Owner cannot leave group")
			return
		}

		logger.Error("Failed to leave group", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to leave group: "+err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
	})
}

// InviteMembers 邀请成员
func (h *GroupHandler) InviteMembers(w http.ResponseWriter, r *http.Request, groupID string) {
	var req InviteMembersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.UserIDs) == 0 {
		h.writeError(w, http.StatusBadRequest, "userIDs is required")
		return
	}

	// 从 context 获取当前用户 ID
	userID := r.Context().Value("user_id")
	if userID == nil {
		h.writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err := h.groupService.InviteMembers(r.Context(), groupID, userID.(string), req.UserIDs)
	if err != nil {
		logger.Error("Failed to invite members", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to invite members: "+err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
	})
}

// KickMembers 踢出成员
func (h *GroupHandler) KickMembers(w http.ResponseWriter, r *http.Request, groupID string) {
	var req InviteMembersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.UserIDs) == 0 {
		h.writeError(w, http.StatusBadRequest, "userIDs is required")
		return
	}

	// 从 context 获取当前用户 ID
	userID := r.Context().Value("user_id")
	if userID == nil {
		h.writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err := h.groupService.KickMembers(r.Context(), groupID, userID.(string), req.UserIDs)
	if err != nil {
		if err == service.ErrPermissionDenied {
			h.writeError(w, http.StatusForbidden, "Permission denied")
			return
		}

		logger.Error("Failed to kick members", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to kick members: "+err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
	})
}

// DismissGroup 解散群组
func (h *GroupHandler) DismissGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	// 从 context 获取当前用户 ID
	userID := r.Context().Value("user_id")
	if userID == nil {
		h.writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err := h.groupService.DismissGroup(r.Context(), groupID, userID.(string))
	if err != nil {
		if err == service.ErrPermissionDenied {
			h.writeError(w, http.StatusForbidden, "Only owner can dismiss group")
			return
		}

		logger.Error("Failed to dismiss group", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to dismiss group: "+err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
	})
}

// GetMyGroups 获取我的群组列表
func (h *GroupHandler) GetMyGroups(w http.ResponseWriter, r *http.Request) {
	// 从 context 获取当前用户 ID
	userID := r.Context().Value("user_id")
	if userID == nil {
		h.writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	groups, err := h.groupService.GetMyGroups(r.Context(), userID.(string))
	if err != nil {
		logger.Error("Failed to get my groups", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to get my groups: "+err.Error())
		return
	}

	// 转换为 DTO
	groupDTOs := make([]*GroupDTO, 0, len(groups))
	for _, group := range groups {
		groupDTOs = append(groupDTOs, toGroupDTO(group))
	}

	h.writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    groupDTOs,
	})
}

// GetGroupMembers 获取群成员列表
func (h *GroupHandler) GetGroupMembers(w http.ResponseWriter, r *http.Request, groupID string) {
	members, err := h.groupService.GetGroupMembers(r.Context(), groupID)
	if err != nil {
		logger.Error("Failed to get group members", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to get group members: "+err.Error())
		return
	}

	// 转换为 UserDTO
	memberDTOs := make([]*UserDTO, 0, len(members))
	for _, member := range members {
		memberDTOs = append(memberDTOs, toUserDTO(member))
	}

	h.writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    memberDTOs,
	})
}

// AuthMiddleware 认证中间件（简化版）
func (h *GroupHandler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("AuthMiddleware called", zap.String("path", r.URL.Path), zap.String("method", r.Method))

		// 从 Authorization header 获取 token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			logger.Warn("Missing authorization header")
			h.writeError(w, http.StatusUnauthorized, "Missing authorization header")
			return
		}

		logger.Info("Auth header received", zap.String("header", authHeader))

		// 解析 Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Warn("Invalid authorization header format", zap.String("header", authHeader))
			h.writeError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		token := parts[1]
		logger.Info("Token extracted", zap.String("token", token[:10]+"..."))

		// 验证 JWT token
		claims, err := h.userService.ValidateToken(token)
		if err != nil {
			logger.Warn("Token validation failed", zap.Error(err))
			h.writeError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		logger.Info("Token validated successfully", zap.String("user_id", claims.UserID))

		// 将用户 ID 存入 context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		r = r.WithContext(ctx)

		next(w, r)
	}
}

// Helper functions

func (h *GroupHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *GroupHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	h.writeJSON(w, statusCode, Response{
		Code:    statusCode,
		Message: message,
	})
}
