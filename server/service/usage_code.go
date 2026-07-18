package service

import (
	"errors"
	"time"

	"send/server/model"
	"send/server/utils"

	"gorm.io/gorm"
)

type UsageCodeService struct {
	db *gorm.DB
}

func NewUsageCodeService(db *gorm.DB) *UsageCodeService {
	return &UsageCodeService{db: db}
}

type UsageCodeResult struct {
	ID          uint       `json:"id"`
	Code        string     `json:"code"`
	Remark      string     `json:"remark"`
	MaxFileSize int64      `json:"max_file_size"`
	MaxUses     int        `json:"max_uses"`
	UsedCount   int        `json:"used_count"`
	ExpireAt    *time.Time `json:"expire_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (s *UsageCodeService) Create(remark string, maxFileSize int64, maxUses int, expireHours int) (string, error) {
	code := utils.GenerateCode(16)

	uc := &model.UsageCode{
		Code:        code,
		Remark:      remark,
		MaxFileSize: maxFileSize,
		MaxUses:     maxUses,
	}
	if expireHours > 0 {
		t := time.Now().Add(time.Duration(expireHours) * time.Hour)
		uc.ExpireAt = &t
	}
	if err := s.db.Create(uc).Error; err != nil {
		return "", err
	}
	return code, nil
}

func (s *UsageCodeService) Verify(code string, fileSize int64) error {
	var uc model.UsageCode
	if err := s.db.Where("code = ?", code).First(&uc).Error; err != nil {
		return errors.New("使用码无效")
	}
	if uc.ExpireAt != nil && uc.ExpireAt.Before(time.Now()) {
		return errors.New("使用码已过期")
	}
	if uc.MaxUses > 0 && uc.UsedCount >= uc.MaxUses {
		return errors.New("使用码已达上限")
	}
	if uc.MaxFileSize > 0 && fileSize > uc.MaxFileSize {
		return errors.New("文件大小超出使用码限制")
	}
	return nil
}

func (s *UsageCodeService) Use(code string) error {
	return s.db.Model(&model.UsageCode{}).Where("code = ?", code).
		UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error
}

func (s *UsageCodeService) List() ([]UsageCodeResult, error) {
	var codes []model.UsageCode
	if err := s.db.Order("created_at DESC").Find(&codes).Error; err != nil {
		return nil, err
	}
	result := make([]UsageCodeResult, len(codes))
	for i, c := range codes {
		result[i] = UsageCodeResult{
			ID:          c.ID,
			Code:        c.Code,
			Remark:      c.Remark,
			MaxFileSize: c.MaxFileSize,
			MaxUses:     c.MaxUses,
			UsedCount:   c.UsedCount,
			ExpireAt:    c.ExpireAt,
			CreatedAt:   c.CreatedAt,
		}
	}
	return result, nil
}

func (s *UsageCodeService) Delete(id uint) error {
	result := s.db.Delete(&model.UsageCode{}, id)
	if result.RowsAffected == 0 {
		return errors.New("使用码不存在")
	}
	return nil
}
