package service

import (
	"errors"
	"os"
	"strings"
	"time"

	"send/server/model"
	"send/server/utils"

	"gorm.io/gorm"
)

// TokenTTL is the lifetime of an admin session token.
const TokenTTL = 24 * time.Hour

type AdminService struct {
	db *gorm.DB
}

func NewAdminService(db *gorm.DB) *AdminService {
	return &AdminService{db: db}
}

func (s *AdminService) HasAdmin() (bool, error) {
	var count int64
	if err := s.db.Model(&model.Admin{}).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *AdminService) Register(username, password string) (string, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if username == "" || password == "" {
		return "", errors.New("用户名和密码不能为空")
	}
	if err := enforceAdminPasswordPolicy(password); err != nil {
		return "", err
	}

	var token string
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.Admin{}).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New("管理员已注册")
		}

		hash, err := utils.HashPassword(password)
		if err != nil {
			return errors.New("密码加密失败")
		}
		token = utils.GenerateToken()
		now := time.Now()
		expiresAt := now.Add(TokenTTL)
		return tx.Create(&model.Admin{
			Username:       username,
			Password:       hash,
			Token:          token,
			TokenExpiresAt: &expiresAt,
		}).Error
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *AdminService) Login(username, password string) (string, error) {
	var admin model.Admin
	if err := s.db.Where("username = ?", strings.ToLower(username)).First(&admin).Error; err != nil {
		return "", errors.New("用户名或密码错误")
	}
	if !utils.CheckPassword(password, admin.Password) {
		return "", errors.New("用户名或密码错误")
	}
	// Generate new token
	token := utils.GenerateToken()
	now := time.Now()
	expiresAt := now.Add(TokenTTL)
	result := s.db.Model(&admin).Updates(map[string]interface{}{
		"token":            token,
		"token_expires_at": expiresAt,
	})
	if result.Error != nil {
		return "", errors.New("登录失败")
	}
	return token, nil
}

func (s *AdminService) Auth(token string) (uint, error) {
	if token == "" {
		return 0, errors.New("未登录")
	}
	var admin model.Admin
	if err := s.db.Where("token = ?", token).First(&admin).Error; err != nil {
		return 0, errors.New("登录已过期，请重新登录")
	}
	// Check token expiration
	if admin.TokenExpiresAt != nil && time.Now().After(*admin.TokenExpiresAt) {
		return 0, errors.New("登录已过期，请重新登录")
	}
	return admin.ID, nil
}

func (s *AdminService) ChangePassword(adminID uint, oldPwd, newPwd string) error {
	var admin model.Admin
	if err := s.db.First(&admin, adminID).Error; err != nil {
		return errors.New("管理员不存在")
	}
	if !utils.CheckPassword(oldPwd, admin.Password) {
		return errors.New("原密码错误")
	}
	if err := enforceAdminPasswordPolicy(newPwd); err != nil {
		return err
	}
	hash, err := utils.HashPassword(newPwd)
	if err != nil {
		return errors.New("密码加密失败")
	}
	// Regenerate token to invalidate any previously compromised sessions
	newToken := utils.GenerateToken()
	now := time.Now()
	expiresAt := now.Add(TokenTTL)
	result := s.db.Model(&admin).Updates(map[string]interface{}{
		"password":         hash,
		"token":            newToken,
		"token_expires_at": expiresAt,
	})
	if result.Error != nil {
		return errors.New("密码修改失败")
	}
	return nil
}

// InvalidateToken revokes the stored session token, so an already-revoked
// cookie/header token can no longer be used until the admin logs in again.
func (s *AdminService) InvalidateToken(adminID uint) error {
	return s.db.Model(&model.Admin{}).
		Where("id = ?", adminID).
		Updates(map[string]interface{}{"token": "", "token_expires_at": nil}).Error
}

// FileListResult is the paginated response for the admin file list.
type FileListResult struct {
	Files    []FileInfo `json:"files"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

func (s *AdminService) ListAllFiles(keyword string, page, pageSize int) (*FileListResult, error) {
	query := s.db.Model(&model.File{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("file_name LIKE ? OR code LIKE ? OR receive_code LIKE ?", like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var files []model.File
	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&files).Error; err != nil {
		return nil, err
	}
	result := make([]FileInfo, len(files))
	for i := range files {
		result[i] = *fileInfoFromModel(&files[i])
	}
	return &FileListResult{
		Files:    result,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// DeleteFile deletes any file by ID (admin only)
func (s *AdminService) DeleteFile(id uint) error {
	var f model.File
	if err := s.db.First(&f, id).Error; err != nil {
		return errors.New("文件不存在")
	}
	err := os.Remove(f.FilePath)
	if err != nil && !os.IsNotExist(err) {
		return errors.New("文件删除失败")
	}
	if err := s.db.Delete(&f).Error; err != nil {
		return errors.New("文件记录删除失败")
	}
	return nil
}

// enforceAdminPasswordPolicy enforces the password requirements for admin
// accounts: minimum length, character class coverage, and a length cap.
// The 128-character cap is below MaxPasswordBytes (512), so bcrypt pre-hashing
// (utils.HashPassword) is always safe here.
func enforceAdminPasswordPolicy(password string) error {
	if len(password) < 8 {
		return errors.New("密码长度不能少于8位")
	}
	if len(password) > 128 {
		return errors.New("密码长度不能超过128位")
	}
	hasUpper := false
	hasDigit := false
	hasSpecial := false
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= '0' && r <= '9':
			hasDigit = true
		case r >= 32 && r <= 126:
			if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') {
				hasSpecial = true
			}
		}
	}
	if !hasUpper {
		return errors.New("密码必须包含大写字母")
	}
	if !hasDigit {
		return errors.New("密码必须包含数字")
	}
	if !hasSpecial {
		return errors.New("密码必须包含特殊字符")
	}
	return nil
}
