package service

import (
	"errors"

	"github.com/arwen/im-server/internal/model"
	"github.com/arwen/im-server/internal/repository"
	"github.com/arwen/im-server/pkg/crypto"
	"github.com/arwen/im-server/pkg/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserService 用户服务
type UserService struct {
	jwtSecret string
}

// NewUserService 创建用户服务
func NewUserService(jwtSecret string) *UserService {
	return &UserService{
		jwtSecret: jwtSecret,
	}
}

// Register 注册用户
func (s *UserService) Register(username, password, nickname string) (*model.User, error) {
	// 检查用户名是否存在
	var existUser model.User
	err := repository.DB.Where("username = ?", username).First(&existUser).Error
	if err == nil {
		return nil, errors.New("username already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 创建用户
	user := &model.User{
		ID:       utils.GenerateID(),
		Username: username,
		Nickname: nickname,
		Password: string(hashedPassword),
		Status:   1,
	}

	if err := repository.DB.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// Login 用户登录
func (s *UserService) Login(username, password, platform string) (string, *model.User, error) {
	// 查询用户
	var user model.User
	err := repository.DB.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("user not found")
		}
		return "", nil, err
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, errors.New("invalid password")
	}

	// 生成Token
	token, err := crypto.GenerateToken(user.ID, platform, s.jwtSecret, 720) // 30天
	if err != nil {
		return "", nil, err
	}

	return token, &user, nil
}

// ValidateToken 验证Token
func (s *UserService) ValidateToken(token string) (*crypto.JWTClaims, error) {
	return crypto.ValidateToken(token, s.jwtSecret)
}

// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(userID string) (*model.User, error) {
	var user model.User
	err := repository.DB.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUserStatus 更新用户状态
func (s *UserService) UpdateUserStatus(userID string, status int) error {
	return repository.DB.Model(&model.User{}).Where("id = ?", userID).Update("status", status).Error
}

