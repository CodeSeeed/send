package service

import (
	"errors"
	"os"
	"strings"

	"send/server/model"
	"send/server/utils"

	"gorm.io/gorm"
)

type AdminService struct {
	db *gorm.DB
}

func NewAdminService(db *gorm.DB) *AdminService {
	return &AdminService{db: db}
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
	s.db.Model(&admin).Update("token", token)
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
	hash, err := utils.HashPassword(newPwd)
	if err != nil {
		return errors.New("密码加密失败")
	}
	// Regenerate token to invalidate any previously compromised sessions
	newToken := utils.GenerateToken()
	s.db.Model(&admin).Updates(map[string]interface{}{
		"password": hash,
		"token":    newToken,
	})
	return nil
}

// ListAllFiles returns all files regardless of manage token
func (s *AdminService) ListAllFiles() ([]FileInfo, error) {
	var files []model.File
	if err := s.db.Order("created_at DESC").Find(&files).Error; err != nil {
		return nil, err
	}
	result := make([]FileInfo, len(files))
	for i, f := range files {
		result[i] = FileInfo{
			ID:            f.ID,
			Code:          f.Code,
			FileName:      f.FileName,
			FileSize:      f.FileSize,
			DownloadCount: f.DownloadCount,
			HasPassword:   f.PasswordHash != "",
			ExpireAt:      f.ExpireAt,
			CreatedAt:     f.CreatedAt,
		}
	}
	return result, nil
}

// DeleteFile deletes any file by ID (admin only)
func (s *AdminService) DeleteFile(id uint) error {
	var f model.File
	if err := s.db.First(&f, id).Error; err != nil {
		return errors.New("文件不存在")
	}
	os.Remove(f.FilePath)
	s.db.Delete(&f)
	return nil
}
