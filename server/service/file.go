package service

import (
	"errors"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"send/server/config"
	"send/server/model"
	"send/server/utils"

	"gorm.io/gorm"
)

// ErrDownloadLimit is returned when a file's max_downloads quota is exhausted.
// Controllers map it to a dedicated HTTP status (409 Conflict) so clients can
// branch on status instead of parsing message text. 409 is used rather than
// 403/404 — those already mean password-wrong / not-found here — and rather
// than 429, which the IP rate-limit middleware owns.
var ErrDownloadLimit = errors.New("下载次数已达上限")

// Download token store (in-memory)
type downloadToken struct {
	token     string
	code      string
	clientIP  string
	createdAt time.Time
}

var (
	downloadTokens sync.Map
	tokenMu        sync.Mutex
	tokenTTL       = 2 * time.Minute
)

func GenerateDownloadToken(code string, clientIP string) string {
	token := utils.GenerateToken()
	downloadTokens.Store(token, &downloadToken{
		token:     token,
		code:      code,
		clientIP:  clientIP,
		createdAt: time.Now(),
	})
	return token
}

// sameClientIP reports whether two client-IP strings belong to the same client
// for download-token binding purposes. IPv4 must match exactly; IPv6 matches
// when both addresses share the same /64 prefix, which tolerates the rotating
// suffix used by privacy extensions (RFC 4941) and mobile carriers without
// letting an unrelated client reuse a leaked token.
//
// Mismatches of address family, malformed addresses, or one side being empty
// never match — falling back to exact equality only when the inputs parse but
// do not fit the rules above.
func sameClientIP(a, b string) bool {
	if a == b {
		return true
	}
	ipA := net.ParseIP(strings.TrimSpace(a))
	ipB := net.ParseIP(strings.TrimSpace(b))
	if ipA == nil || ipB == nil {
		return false
	}
	// An IPv4-in-IPv6 address matches a bare IPv4 address when they denote the
	// same host, so the token stays usable if the client's address form changes
	// between requests (e.g. 127.0.0.1 vs ::ffff:127.0.0.1).
	if ipA.To4() != nil || ipB.To4() != nil {
		return ipA.Equal(ipB)
	}
	// Both are genuine IPv6 — compare the /64 network prefix.
	mask := net.CIDRMask(64, 128)
	return ipA.Mask(mask).Equal(ipB.Mask(mask))
}

// validateDownloadToken checks a token against the in-memory store. When
// consume is true the token is deleted on success (downloads); when false the
// token is left intact (previews, so the same token can still be used to
// download afterwards).
func validateDownloadToken(token, code, clientIP string, consume bool) bool {
	tokenMu.Lock()
	defer tokenMu.Unlock()

	val, ok := downloadTokens.Load(token)
	if !ok {
		return false
	}
	dt := val.(*downloadToken)
	if dt.code != code {
		return false
	}
	if !sameClientIP(dt.clientIP, clientIP) {
		return false
	}
	if time.Since(dt.createdAt) > tokenTTL {
		downloadTokens.Delete(token)
		return false
	}
	if consume {
		downloadTokens.Delete(token)
	}
	return true
}

func ValidateDownloadToken(token, code, clientIP string) bool {
	return validateDownloadToken(token, code, clientIP, true)
}

// ValidateDownloadTokenPeek validates a download token without consuming it.
// Use this for previews so the same token remains available for a subsequent
// download attempt.
func ValidateDownloadTokenPeek(token, code, clientIP string) bool {
	return validateDownloadToken(token, code, clientIP, false)
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
	FileName            string
	FileSize            int64
	Password            string
	ExpireHours         int
	ExpireMinutes       int
	MaxDownloads        int64
	GenerateReceiveCode bool
}

type FileResult struct {
	Code         string     `json:"code"`
	ReceiveCode  *string    `json:"receive_code,omitempty"`
	FileName     string     `json:"file_name"`
	FileSize     int64      `json:"file_size"`
	HasPassword  bool       `json:"has_password"`
	MaxDownloads int64      `json:"max_downloads"`
	ExpireAt     *time.Time `json:"expire_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// FileInfo is the full file view returned to the admin dashboard, including
// the receive code and download progress. Public endpoints must NOT use this
// type — it leaks the receive code (a selectively-shared extraction secret)
// and download metadata to anyone who knows the share code.
type FileInfo struct {
	ID            uint       `json:"id"`
	Code          string     `json:"code"`
	ReceiveCode   *string    `json:"receive_code,omitempty"`
	FileName      string     `json:"file_name"`
	FileSize      int64      `json:"file_size"`
	DownloadCount int64      `json:"download_count"`
	MaxDownloads  int64      `json:"max_downloads"`
	HasPassword   bool       `json:"has_password"`
	ExpireAt      *time.Time `json:"expire_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// PublicFileInfo is the redacted view returned by the public file-info and
// receive-code endpoints. It omits the id, receive code, download count, and
// max-downloads limit so an unauthenticated caller who only knows the share
// code cannot derive the receive code (capability separation) or infer
// remaining download budget.
type PublicFileInfo struct {
	Code        string     `json:"code"`
	FileName    string     `json:"file_name"`
	FileSize    int64      `json:"file_size"`
	HasPassword bool       `json:"has_password"`
	ExpireAt    *time.Time `json:"expire_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// publicFileInfoFromModel builds the redacted public view. It deliberately
// drops id, receive_code, download_count and max_downloads.
func publicFileInfoFromModel(f *model.File) *PublicFileInfo {
	return &PublicFileInfo{
		Code:        f.Code,
		FileName:    f.FileName,
		FileSize:    f.FileSize,
		HasPassword: f.PasswordHash != "",
		ExpireAt:    f.ExpireAt,
		CreatedAt:   f.CreatedAt,
	}
}

// Create inserts a new file record with an auto-generated share code.
// Retries on duplicate key collisions (up to 5 attempts).
func (s *FileService) Create(params *CreateFileParams) (*FileResult, error) {
	for attempt := 0; attempt < 5; attempt++ {
		code := utils.GenerateCode(s.cfg.Upload.CodeLength)
		candidate := &model.File{
			Code:         code,
			FileName:     params.FileName,
			FilePath:     filepath.Join(s.cfg.Upload.Dir, code),
			FileSize:     params.FileSize,
			MaxDownloads: params.MaxDownloads,
		}
		if params.GenerateReceiveCode {
			length := s.cfg.Upload.ReceiveCodeLength
			if length <= 0 || length > 32 {
				length = 8
			}
			receiveCode := utils.GenerateReceiveCode(length)
			candidate.ReceiveCode = &receiveCode
		}

		if params.Password != "" {
			hash, err := utils.HashPassword(params.Password)
			if err != nil {
				return nil, err
			}
			candidate.PasswordHash = hash
		}

		if params.ExpireMinutes > 0 {
			t := time.Now().Add(time.Duration(params.ExpireMinutes) * time.Minute)
			candidate.ExpireAt = &t
		} else if params.ExpireHours > 0 {
			t := time.Now().Add(time.Duration(params.ExpireHours) * time.Hour)
			candidate.ExpireAt = &t
		}

		err := s.db.Create(candidate).Error
		if err == nil {
			return &FileResult{
				Code:         candidate.Code,
				ReceiveCode:  candidate.ReceiveCode,
				FileName:     candidate.FileName,
				FileSize:     candidate.FileSize,
				HasPassword:  candidate.PasswordHash != "",
				MaxDownloads: candidate.MaxDownloads,
				ExpireAt:     candidate.ExpireAt,
				CreatedAt:    candidate.CreatedAt,
			}, nil
		}
		if !errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, err
		}
		// Duplicate key: retry with a new code
	}
	return nil, errors.New("生成文件代码失败，请重试")
}

func (s *FileService) isExpired(f *model.File) bool {
	return f.ExpireAt != nil && !time.Now().Before(*f.ExpireAt)
}

func (s *FileService) GetByCode(code string) (*PublicFileInfo, error) {
	var f model.File
	if err := s.db.Where("code = ?", code).First(&f).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("文件不存在或已过期")
		}
		return nil, err
	}
	if s.isExpired(&f) {
		return nil, errors.New("文件不存在或已过期")
	}
	return publicFileInfoFromModel(&f), nil
}

func (s *FileService) GetByReceiveCode(receiveCode string) (*PublicFileInfo, error) {
	receiveCode = strings.ToUpper(strings.TrimSpace(receiveCode))
	if receiveCode == "" || len(receiveCode) > 32 {
		return nil, errors.New("收件码无效或文件已过期")
	}

	var f model.File
	if err := s.db.Where("receive_code = ?", receiveCode).First(&f).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("收件码无效或文件已过期")
		}
		return nil, err
	}
	if s.isExpired(&f) {
		return nil, errors.New("收件码无效或文件已过期")
	}
	return publicFileInfoFromModel(&f), nil
}

func fileInfoFromModel(f *model.File) *FileInfo {
	return &FileInfo{
		ID:            f.ID,
		Code:          f.Code,
		ReceiveCode:   f.ReceiveCode,
		FileName:      f.FileName,
		FileSize:      f.FileSize,
		DownloadCount: f.DownloadCount,
		MaxDownloads:  f.MaxDownloads,
		HasPassword:   f.PasswordHash != "",
		ExpireAt:      f.ExpireAt,
		CreatedAt:     f.CreatedAt,
	}
}

func (s *FileService) VerifyPassword(code, password string) error {
	var f model.File
	if err := s.db.Where("code = ?", code).First(&f).Error; err != nil {
		return errors.New("文件不存在或已过期")
	}
	if s.isExpired(&f) {
		return errors.New("文件不存在或已过期")
	}
	if f.MaxDownloads > 0 && f.DownloadCount >= f.MaxDownloads {
		return ErrDownloadLimit
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
	if s.isExpired(&f) {
		return "", "", errors.New("文件不存在或已过期")
	}
	// Atomically increment the download counter ONLY when the limit allows it.
	// The conditional UPDATE makes the limit check race-free even when several
	// clients download simultaneously: exactly one request wins the last slot.
	res := s.db.Model(&model.File{}).
		Where("code = ? AND (max_downloads = 0 OR download_count < max_downloads)", code).
		UpdateColumn("download_count", gorm.Expr("download_count + 1"))
	if res.Error != nil {
		return "", "", errors.New("下载失败")
	}
	if res.RowsAffected != 1 {
		return "", "", ErrDownloadLimit
	}
	return f.FilePath, f.FileName, nil
}

// GetFilePathForPreview returns the file path WITHOUT incrementing the
// download counter. Previewing a file does not consume a download slot.
func (s *FileService) GetFilePathForPreview(code string) (string, string, error) {
	var f model.File
	if err := s.db.Where("code = ?", code).First(&f).Error; err != nil {
		return "", "", errors.New("文件不存在或已过期")
	}
	if s.isExpired(&f) {
		return "", "", errors.New("文件不存在或已过期")
	}
	if f.MaxDownloads > 0 && f.DownloadCount >= f.MaxDownloads {
		return "", "", ErrDownloadLimit
	}
	return f.FilePath, f.FileName, nil
}

// DeleteByCode removes a file record by share code (for rollback on upload failure).
// Does not remove the physical file — the caller is responsible for that.
func (s *FileService) DeleteByCode(code string) error {
	return s.db.Where("code = ?", code).Delete(&model.File{}).Error
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

// CleanupOrphans reconciles the database and upload directory at startup.
// It removes both disk files without a record and records whose file is
// missing or incomplete after an interrupted DB-first upload.
func (s *FileService) CleanupOrphans() {
	var files []model.File
	if err := s.db.Find(&files).Error; err != nil {
		log.Printf("[warn] 扫描文件记录失败: %v", err)
		return
	}
	for i := range files {
		f := &files[i]
		info, err := os.Stat(f.FilePath)
		if err == nil && !info.IsDir() && info.Size() == f.FileSize {
			continue
		}
		if err != nil && !os.IsNotExist(err) {
			log.Printf("[warn] 检查文件 %s 失败: %v", f.FilePath, err)
			continue
		}

		// A partial file may still exist after a crash. Remove it before
		// deleting the broken record; a later disk pass retries failed removes.
		if err == nil {
			if removeErr := os.Remove(f.FilePath); removeErr != nil && !os.IsNotExist(removeErr) {
				log.Printf("[warn] 删除不完整文件 %s 失败: %v", f.FilePath, removeErr)
			}
		}
		if deleteErr := s.db.Delete(f).Error; deleteErr != nil {
			log.Printf("[warn] 删除无效文件记录 %s 失败: %v", f.Code, deleteErr)
		}
	}

	entries, err := os.ReadDir(s.cfg.Upload.Dir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[warn] 扫描上传目录失败: %v", err)
		}
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		var count int64
		if err := s.db.Model(&model.File{}).Where("code = ?", entry.Name()).Count(&count).Error; err != nil {
			log.Printf("[warn] 检查孤儿文件 %s 失败: %v", entry.Name(), err)
			continue
		}
		if count == 0 {
			path := filepath.Join(s.cfg.Upload.Dir, entry.Name())
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				log.Printf("[warn] 删除孤儿文件 %s 失败: %v", path, err)
			}
		}
	}
}
