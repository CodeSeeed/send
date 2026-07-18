package service

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"send/server/config"
	"send/server/model"
	"send/server/utils"

	"gorm.io/gorm"
)

// Download token store (in-memory)
type downloadToken struct {
	token     string
	code      string
	createdAt time.Time
}

var (
	downloadTokens sync.Map
	tokenTTL       = 5 * time.Minute
)

func GenerateDownloadToken(code string) string {
	token := utils.GenerateToken()
	downloadTokens.Store(token, &downloadToken{
		token:     token,
		code:      code,
		createdAt: time.Now(),
	})
	return token
}

func ValidateDownloadToken(token, code string) bool {
	val, ok := downloadTokens.Load(token)
	if !ok {
		return false
	}
	dt := val.(*downloadToken)
	if dt.code != code {
		return false
	}
	if time.Since(dt.createdAt) > tokenTTL {
		downloadTokens.Delete(token)
		return false
	}
	// One-time use — consume immediately
	downloadTokens.Delete(token)
	return true
}

// CleanupExpiredTokens removes expired download tokens periodically
func CleanupExpiredTokens() {
	now := time.Now()
	downloadTokens.Range(func(key, val interface{}) bool {
		dt := val.(*downloadToken)
		if now.Sub(dt.createdAt) > tokenTTL {
			downloadTokens.Delete(key)
		}
		return true
	})
}

type FileService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewFileService(db *gorm.DB, cfg *config.Config) *FileService {
	return &FileService{db: db, cfg: cfg}
}

type CreateFileParams struct {
	FileName    string
	FileSize    int64
	Password    string
	ExpireHours int
}

type FileResult struct {
	Code        string    `json:"code"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	ManageToken string    `json:"manage_token"`
	HasPassword bool      `json:"has_password"`
	ExpireAt    *time.Time `json:"expire_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type FileInfo struct {
	ID            uint       `json:"id"`
	Code          string     `json:"code"`
	FileName      string     `json:"file_name"`
	FileSize      int64      `json:"file_size"`
	DownloadCount int64      `json:"download_count"`
	HasPassword   bool       `json:"has_password"`
	ExpireAt      *time.Time `json:"expire_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (s *FileService) Create(params *CreateFileParams) (*FileResult, error) {
	code := utils.GenerateCode(s.cfg.Upload.CodeLength)
	manageToken := utils.GenerateToken()

	f := &model.File{
		Code:        code,
		FileName:    params.FileName,
		FilePath:    filepath.Join(s.cfg.Upload.Dir, code),
		FileSize:    params.FileSize,
		ManageToken: manageToken,
	}

	if params.Password != "" {
		hash, err := utils.HashPassword(params.Password)
		if err != nil {
			return nil, err
		}
		f.PasswordHash = hash
	}

	if params.ExpireHours > 0 {
		t := time.Now().Add(time.Duration(params.ExpireHours) * time.Hour)
		f.ExpireAt = &t
	}

	if err := s.db.Create(f).Error; err != nil {
		return nil, err
	}

	return &FileResult{
		Code:        f.Code,
		FileName:    f.FileName,
		FileSize:    f.FileSize,
		ManageToken: f.ManageToken,
		HasPassword: f.PasswordHash != "",
		ExpireAt:    f.ExpireAt,
		CreatedAt:   f.CreatedAt,
	}, nil
}

func (s *FileService) GetByCode(code string) (*FileInfo, error) {
	var f model.File
	if err := s.db.Where("code = ?", code).First(&f).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("文件不存在或已过期")
		}
		return nil, err
	}
	return &FileInfo{
		ID:            f.ID,
		Code:          f.Code,
		FileName:      f.FileName,
		FileSize:      f.FileSize,
		DownloadCount: f.DownloadCount,
		HasPassword:   f.PasswordHash != "",
		ExpireAt:      f.ExpireAt,
		CreatedAt:     f.CreatedAt,
	}, nil
}

func (s *FileService) VerifyPassword(code, password string) error {
	var f model.File
	if err := s.db.Where("code = ?", code).First(&f).Error; err != nil {
		return errors.New("文件不存在或已过期")
	}
	if f.PasswordHash == "" {
		return nil
	}
	if !utils.CheckPassword(password, f.PasswordHash) {
		return errors.New("密码错误")
	}
	return nil
}

func (s *FileService) GetFilePath(code string) (string, string, error) {
	var f model.File
	if err := s.db.Where("code = ?", code).First(&f).Error; err != nil {
		return "", "", errors.New("文件不存在或已过期")
	}
	s.db.Model(&f).UpdateColumn("download_count", gorm.Expr("download_count + 1"))
	return f.FilePath, f.FileName, nil
}

func (s *FileService) ListByToken(token string) ([]FileInfo, error) {
	var files []model.File
	if err := s.db.Where("manage_token = ?", token).Order("created_at DESC").Find(&files).Error; err != nil {
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

func (s *FileService) Delete(id uint, token string) error {
	result := s.db.Where("id = ? AND manage_token = ?", id, token).Delete(&model.File{})
	if result.RowsAffected == 0 {
		return errors.New("文件不存在或无权删除")
	}
	// Remove file from disk
	var f model.File
	s.db.Unscoped().Where("id = ?", id).First(&f)
	if f.FilePath != "" {
		os.Remove(f.FilePath)
	}
	return nil
}

func (s *FileService) CleanupExpired() {
	var files []model.File
	now := time.Now()
	s.db.Where("expire_at IS NOT NULL AND expire_at < ?", now).Find(&files)
	for _, f := range files {
		os.Remove(f.FilePath)
		s.db.Delete(&f)
	}
}
