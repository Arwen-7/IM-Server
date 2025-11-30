package service

import (
	"context"
	"time"

	"github.com/arwen/im-server/internal/model"
	"github.com/arwen/im-server/pkg/logger"
	"github.com/arwen/im-server/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GroupService 群组服务
type GroupService struct {
	db *gorm.DB
}

// NewGroupService 创建群组服务
func NewGroupService(db *gorm.DB) *GroupService {
	return &GroupService{
		db: db,
	}
}

// CreateGroup 创建群组
func (s *GroupService) CreateGroup(ctx context.Context, ownerUserID, groupName, faceURL, introduction string, memberUserIDs []string) (*model.Group, error) {
	// 生成群组 ID
	groupID := utils.GenerateID()

	// 创建群组对象
	group := &model.Group{
		ID:          groupID,
		Name:        groupName,
		Avatar:      faceURL,
		Description: introduction,
		OwnerID:     ownerUserID,
		MaxMembers:  500,
		Status:      1, // 正常状态
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 开启事务
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 插入群组
		if err := tx.Create(group).Error; err != nil {
			return err
		}

		// 添加群主
		owner := &model.GroupMember{
			ID:        utils.GenerateID(),
			GroupID:   groupID,
			UserID:    ownerUserID,
			Role:      1, // 群主
			Status:    1,
			JoinedAt:  time.Now(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := tx.Create(owner).Error; err != nil {
			return err
		}

		// 添加群成员
		for _, userID := range memberUserIDs {
			if userID == ownerUserID {
				continue // 跳过群主
			}

			member := &model.GroupMember{
				ID:        utils.GenerateID(),
				GroupID:   groupID,
				UserID:    userID,
				Role:      3, // 普通成员
				Status:    1,
				JoinedAt:  time.Now(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := tx.Create(member).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		logger.Error("Failed to create group", zap.Error(err))
		return nil, err
	}

	logger.Info("Group created", zap.String("group_id", groupID), zap.String("owner", ownerUserID))

	return group, nil
}

// GetGroup 获取群组信息
func (s *GroupService) GetGroup(ctx context.Context, groupID string) (*model.Group, error) {
	var group model.Group
	if err := s.db.Where("id = ?", groupID).First(&group).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	return &group, nil
}

// UpdateGroup 更新群组信息
func (s *GroupService) UpdateGroup(ctx context.Context, groupID string, updates map[string]interface{}) (*model.Group, error) {
	group, err := s.GetGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if err := s.db.Model(&group).Updates(updates).Error; err != nil {
		return nil, err
	}

	return group, nil
}

// JoinGroup 加入群组
func (s *GroupService) JoinGroup(ctx context.Context, groupID, userID string) error {
	// 检查群组是否存在
	if _, err := s.GetGroup(ctx, groupID); err != nil {
		return err
	}

	// 检查是否已经是群成员
	isMember, err := s.IsGroupMember(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if isMember {
		return ErrAlreadyGroupMember
	}

	// 添加群成员
	member := &model.GroupMember{
		ID:        utils.GenerateID(),
		GroupID:   groupID,
		UserID:    userID,
		Role:      3, // 普通成员
		Status:    1,
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.Create(member).Error; err != nil {
		return err
	}

	logger.Info("User joined group", zap.String("group_id", groupID), zap.String("user_id", userID))
	return nil
}

// LeaveGroup 退出群组
func (s *GroupService) LeaveGroup(ctx context.Context, groupID, userID string) error {
	// 检查是否是群主
	group, err := s.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}

	if group.OwnerID == userID {
		return ErrOwnerCannotLeave
	}

	// 删除群成员
	if err := s.db.Where("group_id = ? AND user_id = ?", groupID, userID).Delete(&model.GroupMember{}).Error; err != nil {
		return err
	}

	logger.Info("User left group", zap.String("group_id", groupID), zap.String("user_id", userID))
	return nil
}

// InviteMembers 邀请成员
func (s *GroupService) InviteMembers(ctx context.Context, groupID, inviterID string, userIDs []string) error {
	// 检查群组是否存在
	if _, err := s.GetGroup(ctx, groupID); err != nil {
		return err
	}

	// 检查邀请人是否是群成员
	isMember, err := s.IsGroupMember(ctx, groupID, inviterID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrNotGroupMember
	}

	// 添加成员
	for _, userID := range userIDs {
		isMember, _ := s.IsGroupMember(ctx, groupID, userID)
		if isMember {
			continue
		}

		member := &model.GroupMember{
			ID:        utils.GenerateID(),
			GroupID:   groupID,
			UserID:    userID,
			Role:      3,
			Status:    1,
			JoinedAt:  time.Now(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := s.db.Create(member).Error; err != nil {
			logger.Error("Failed to add member", zap.Error(err), zap.String("user_id", userID))
			continue
		}
	}

	logger.Info("Members invited", zap.String("group_id", groupID), zap.Int("count", len(userIDs)))
	return nil
}

// KickMembers 踢出成员
func (s *GroupService) KickMembers(ctx context.Context, groupID, operatorID string, userIDs []string) error {
	// 检查群组是否存在
	group, err := s.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}

	// 检查操作人是否是群主或管理员
	role, err := s.GetMemberRole(ctx, groupID, operatorID)
	if err != nil {
		return err
	}
	if role != 1 && role != 2 { // 1=群主，2=管理员
		return ErrPermissionDenied
	}

	// 踢出成员
	for _, userID := range userIDs {
		if userID == group.OwnerID {
			continue // 不能踢出群主
		}

		if err := s.db.Where("group_id = ? AND user_id = ?", groupID, userID).Delete(&model.GroupMember{}).Error; err != nil {
			logger.Error("Failed to kick member", zap.Error(err), zap.String("user_id", userID))
		}
	}

	logger.Info("Members kicked", zap.String("group_id", groupID), zap.Int("count", len(userIDs)))
	return nil
}

// DismissGroup 解散群组
func (s *GroupService) DismissGroup(ctx context.Context, groupID, operatorID string) error {
	// 检查群组是否存在
	group, err := s.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}

	// 只有群主可以解散群组
	if group.OwnerID != operatorID {
		return ErrPermissionDenied
	}

	// 开启事务
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 删除群成员
		if err := tx.Where("group_id = ?", groupID).Delete(&model.GroupMember{}).Error; err != nil {
			return err
		}

		// 删除群组
		if err := tx.Delete(&group).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	logger.Info("Group dismissed", zap.String("group_id", groupID))
	return nil
}

// GetMyGroups 获取我的群组列表
func (s *GroupService) GetMyGroups(ctx context.Context, userID string) ([]*model.Group, error) {
	var groups []*model.Group

	err := s.db.
		Joins("INNER JOIN group_members ON groups.id = group_members.group_id").
		Where("group_members.user_id = ? AND group_members.status = 1", userID).
		Order("groups.updated_at DESC").
		Find(&groups).Error

	if err != nil {
		return nil, err
	}

	return groups, nil
}

// GetGroupMembers 获取群成员列表
func (s *GroupService) GetGroupMembers(ctx context.Context, groupID string) ([]*model.User, error) {
	var users []*model.User

	err := s.db.
		Joins("INNER JOIN group_members ON users.user_id = group_members.user_id").
		Where("group_members.group_id = ? AND group_members.status = 1", groupID).
		Order("group_members.role ASC, group_members.joined_at ASC").
		Find(&users).Error

	if err != nil {
		return nil, err
	}

	return users, nil
}

// IsGroupMember 检查是否是群成员
func (s *GroupService) IsGroupMember(ctx context.Context, groupID, userID string) (bool, error) {
	var count int64
	if err := s.db.Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ? AND status = 1", groupID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetMemberRole 获取成员角色
func (s *GroupService) GetMemberRole(ctx context.Context, groupID, userID string) (int, error) {
	var member model.GroupMember
	if err := s.db.Where("group_id = ? AND user_id = ? AND status = 1", groupID, userID).
		First(&member).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return -1, ErrNotGroupMember
		}
		return -1, err
	}

	return member.Role, nil
}
