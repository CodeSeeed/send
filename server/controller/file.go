package controller

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"send/server/config"
	"send/server/service"
	"send/server/utils"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
)

type FileController struct {
	svc         *service.FileService
	cfg         *config.Config
	settingsSvc *service.SettingsService
}

func NewFileController(svc *service.FileService, cfg *config.Config, settingsSvc *service.SettingsService) *FileController {
	return &FileController{svc: svc, cfg: cfg, settingsSvc: settingsSvc}
}

func (ctr *FileController) Upload(c *gin.Context) {
	settings, err := ctr.settingsSvc.Get()
	if err != nil {
		utils.Error(c, 500, "获取上传设置失败")
		return
	}
	if settings.MaxFileSize <= 0 {
		utils.Error(c, 500, "上传大小设置无效")
		return
	}

	const multipartOverhead = 1 << 20
	maxBodySize := settings.MaxFileSize + multipartOverhead
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodySize)

	file, err := c.FormFile("file")
	if err != nil {
		utils.Error(c, 400, "请选择文件或文件超过大小限制")
		return
	}
	if file.Size > settings.MaxFileSize {
		utils.Error(c, 400, "文件超过当前大小限制")
		return
	}

	// Validate extension against allowlist before any DB write
	if !ctr.isAllowedExtension(file.Filename) {
		utils.Error(c, 400, "不支持的文件类型")
		return
	}

	fileName := sanitizeFileName(file.Filename)

	password := c.PostForm("password")
	if password != "" && len(password) < 4 {
		utils.Error(c, 400, "访问密码长度不能少于4位")
		return
	}
	if len(password) > utils.MaxPasswordBytes {
		utils.Error(c, 400, "访问密码过长")
		return
	}

	expireHoursText, hasExpireHours := c.GetPostForm("expire_hours")
	expireMinutesText, hasExpireMinutes := c.GetPostForm("expire_minutes")
	if hasExpireHours && hasExpireMinutes {
		utils.Error(c, 400, "过期时间参数不能同时设置")
		return
	}

	expireHours := 0
	if hasExpireHours {
		expireHours, err = strconv.Atoi(expireHoursText)
		if err != nil || expireHours <= 0 {
			utils.Error(c, 400, "过期小时数必须是大于0的整数")
			return
		}
	}

	expireMinutes := 0
	if hasExpireMinutes {
		expireMinutes, err = strconv.Atoi(expireMinutesText)
		if err != nil || expireMinutes <= 0 {
			utils.Error(c, 400, "过期分钟数必须是大于0的整数")
			return
		}
	}

	// Bound expire params
	if expireHours > ctr.cfg.Upload.MaxExpireHours {
		utils.Error(c, 400, fmt.Sprintf("过期小时数不能超过%d", ctr.cfg.Upload.MaxExpireHours))
		return
	}
	if expireMinutes > ctr.cfg.Upload.MaxExpireHours*60 {
		utils.Error(c, 400, fmt.Sprintf("过期分钟数不能超过%d", ctr.cfg.Upload.MaxExpireHours*60))
		return
	}

	maxDownloads := int64(0)
	if maxDownloadsText, ok := c.GetPostForm("max_downloads"); ok && maxDownloadsText != "" {
		md, err := strconv.ParseInt(maxDownloadsText, 10, 64)
		if err != nil || md <= 0 {
			utils.Error(c, 400, "下载次数必须是大于0的整数")
			return
		}
		if md > 1000000 {
			utils.Error(c, 400, "下载次数过大")
			return
		}
		maxDownloads = md
	}

	generateReceiveCode := false
	if value, ok := c.GetPostForm("generate_receive_code"); ok && value != "" {
		generateReceiveCode, err = strconv.ParseBool(value)
		if err != nil {
			utils.Error(c, 400, "收件码参数错误")
			return
		}
	}

	src, err := file.Open()
	if err != nil {
		utils.Error(c, 500, "上传失败")
		return
	}
	defer src.Close()

	// Deep MIME detection: sniff header, then let mimetype inspect the start of content
	header := make([]byte, 512)
	n, _ := io.ReadFull(src, header)
	if n > 0 {
		mimeType := mimetype.Detect(header[:n]).String()
		if !ctr.isAllowedMimeType(mimeType) {
			utils.Error(c, 400, "文件类型不被允许")
			return
		}
	}
	// Reset to beginning for actual copy
	src.Seek(0, 0)

	// Create DB record FIRST (with auto-generated code, retry on duplicate).
	// This is the atomic reservation — no other request can claim the same code.
	params := &service.CreateFileParams{
		FileName:            fileName,
		FileSize:            file.Size,
		Password:            password,
		ExpireHours:         expireHours,
		ExpireMinutes:       expireMinutes,
		MaxDownloads:        maxDownloads,
		GenerateReceiveCode: generateReceiveCode,
	}
	result, err := ctr.svc.Create(params)
	if err != nil {
		utils.Error(c, 500, "上传失败")
		return
	}

	// Write file to disk using the reserved code.
	dst := filepath.Join(ctr.cfg.Upload.Dir, result.Code)
	out, err := os.Create(dst)
	if err != nil {
		ctr.svc.DeleteByCode(result.Code)
		utils.Error(c, 500, "上传失败")
		return
	}

	written, copyErr := io.Copy(out, src)
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil || written != file.Size {
		os.Remove(dst)
		ctr.svc.DeleteByCode(result.Code)
		utils.Error(c, 500, "上传失败")
		return
	}

	utils.Success(c, result)
}

func (ctr *FileController) Info(c *gin.Context) {
	code := c.Param("code")
	info, err := ctr.svc.GetByCode(code)
	if err != nil {
		utils.Error(c, 404, err.Error())
		return
	}
	utils.Success(c, info)
}

func (ctr *FileController) Receive(c *gin.Context) {
	var req struct {
		ReceiveCode string `json:"receive_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "请输入收件码")
		return
	}

	info, err := ctr.svc.GetByReceiveCode(req.ReceiveCode)
	if err != nil {
		utils.Error(c, 404, err.Error())
		return
	}
	utils.Success(c, info)
}

func (ctr *FileController) VerifyPassword(c *gin.Context) {
	code := c.Param("code")

	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	if err := ctr.svc.VerifyPassword(code, req.Password); err != nil {
		utils.Error(c, 403, err.Error())
		return
	}

	// Generate one-time download token bound to the requesting client IP
	clientIP := c.ClientIP()
	token := service.GenerateDownloadToken(code, clientIP)
	utils.Success(c, gin.H{"download_token": token})
}

// bearerToken extracts the token from an Authorization: Bearer header,
// returning "" when the header is absent or not in Bearer form.
func bearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}

func (ctr *FileController) Download(c *gin.Context) {
	code := c.Param("code")

	// Read token from Authorization: Bearer header (keeps it out of logs & history)
	token := bearerToken(c)
	if token == "" {
		utils.Error(c, 403, "缺少下载凭证")
		return
	}

	clientIP := c.ClientIP()
	if !service.ValidateDownloadToken(token, code, clientIP) {
		utils.Error(c, 403, "下载凭证无效或已过期")
		return
	}

	filePath, fileName, err := ctr.svc.GetFilePath(code)
	if err != nil {
		utils.Error(c, 404, err.Error())
		return
	}

	c.Header("Content-Disposition", formatContentDisposition(filepath.Base(fileName)))
	c.Header("Content-Type", "application/octet-stream")
	c.File(filePath)
}

// previewableMimeTypes are the content types that the browser can render inline.
var previewableMimeTypes = map[string]bool{
	"application/pdf":  true,
	"image/png":        true,
	"image/jpeg":       true,
	"image/gif":        true,
	"image/webp":       true,
	"text/plain":       true,
	"text/csv":         true,
	"text/markdown":    true,
	"application/json": true,
}

func (ctr *FileController) Preview(c *gin.Context) {
	code := c.Param("code")

	// Read token from Authorization: Bearer header
	token := bearerToken(c)
	if token == "" {
		utils.Error(c, 403, "缺少预览凭证")
		return
	}

	clientIP := c.ClientIP()
	// Peek validates without consuming the token — the same token can still be
	// used for a subsequent download (or another preview).
	if !service.ValidateDownloadTokenPeek(token, code, clientIP) {
		utils.Error(c, 403, "预览凭证无效或已过期")
		return
	}

	filePath, fileName, err := ctr.svc.GetFilePathForPreview(code)
	if err != nil {
		utils.Error(c, 404, err.Error())
		return
	}

	// Detect MIME type from the file content (not from extension or user input)
	mimeType, err := mimetype.DetectFile(filePath)
	if err != nil {
		utils.Error(c, 500, "读取文件类型失败")
		return
	}
	mimeStr := mimeType.String()
	if !previewableMimeTypes[mimeStr] {
		utils.Error(c, 400, "该文件类型不支持预览")
		return
	}

	c.Header("Content-Disposition", "inline; filename="+url.PathEscape(fileName))
	c.Header("Content-Type", mimeStr)
	c.File(filePath)
}

func (ctr *FileController) isAllowedExtension(filename string) bool {
	if len(ctr.cfg.Upload.AllowedExtensions) == 0 {
		return true
	}
	ext := strings.ToLower(filepath.Ext(filename))
	for _, allowed := range ctr.cfg.Upload.AllowedExtensions {
		if ext == allowed {
			return true
		}
	}
	return false
}

func (ctr *FileController) isAllowedMimeType(mimeType string) bool {
	if len(ctr.cfg.Upload.AllowedMimeTypes) == 0 {
		return true
	}
	// mimetype library returns types without parameters, but handle gracefully
	baseType := strings.SplitN(mimeType, ";", 2)[0]
	baseType = strings.TrimSpace(baseType)
	for _, allowed := range ctr.cfg.Upload.AllowedMimeTypes {
		if baseType == allowed {
			return true
		}
	}
	return false
}

func sanitizeFileName(name string) string {
	base := filepath.Base(name)
	var b strings.Builder
	b.Grow(len(base))
	for _, r := range base {
		// Strip control characters (ASCII 0-31, 127) and Unicode control formats
		if r < 32 || r == 127 {
			continue
		}
		// Zero-width spaces / marks
		if r == 0x200B || r == 0x200C || r == 0x200D || r == 0xFEFF {
			continue
		}
		// Bidirectional text control characters (spoofing risk)
		if r >= 0x200E && r <= 0x200F { // LRE, RLE
			continue
		}
		if r >= 0x2028 && r <= 0x202E { // line/para separators + LRO, RLO, PDF, LRI, RLI, FSI
			continue
		}
		if r >= 0x2066 && r <= 0x2069 { // LRI, RLI, FSI, PDI
			continue
		}
		if r == 0x061C { // Arabic letter mark
			continue
		}
		// General Unicode control categories
		if unicode.Is(unicode.Cc, r) || unicode.Is(unicode.Cf, r) {
			continue
		}
		b.WriteRune(r)
	}
	sanitized := b.String()
	// Truncate safely at a UTF-8 rune boundary (filesystem limit is 255 bytes)
	runes := []rune(sanitized)
	if len(sanitized) > 255 {
		// Truncate by runes until it fits in 255 bytes
		hi := len(runes)
		for hi > 0 && len(string(runes[:hi])) > 255 {
			hi--
		}
		sanitized = string(runes[:hi])
	}
	// Fallback if empty after sanitization
	if sanitized == "" {
		sanitized = "unnamed"
	}
	return sanitized
}

// formatContentDisposition builds a Content-Disposition header that keeps the
// original (possibly non-ASCII) filename via RFC 5987/2231 filename*= while
// also providing a pure-ASCII filename= fallback for clients that cannot parse
// the extended form.
func formatContentDisposition(filename string) string {
	base := filepath.Base(filename)
	// ASCII-safe fallback: strip non-ASCII runes, keep extension shape
	var b strings.Builder
	for _, r := range base {
		if r >= 32 && r <= 126 {
			b.WriteRune(r)
		}
	}
	fallback := b.String()
	if strings.TrimSpace(fallback) == "" {
		fallback = "download"
	}
	fallback = strings.ReplaceAll(fallback, `"`, `'`)

	header := mime.FormatMediaType("attachment", map[string]string{"filename": fallback})
	// Append original name in RFC 5987 form when it differs from the ASCII fallback
	if base != fallback {
		encoded := url.QueryEscape(base)
		encoded = strings.ReplaceAll(encoded, "+", "%20")
		header += "; filename*=UTF-8''" + encoded
	}
	return header
}
