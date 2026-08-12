package controller

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"send/server/config"
	"send/server/service"
	"send/server/utils"

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

	password := c.PostForm("password")
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

	params := &service.CreateFileParams{
		FileName:      file.Filename,
		FileSize:      file.Size,
		Password:      password,
		ExpireHours:   expireHours,
		ExpireMinutes: expireMinutes,
	}

	result, err := ctr.svc.Create(params)
	if err != nil {
		utils.Error(c, 500, "上传失败")
		return
	}

	dst := filepath.Join(ctr.cfg.Upload.Dir, result.Code)
	src, err := file.Open()
	if err != nil {
		ctr.svc.DeleteByID(result.Code)
		utils.Error(c, 500, "上传失败")
		return
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		ctr.svc.DeleteByID(result.Code)
		utils.Error(c, 500, "上传失败")
		return
	}

	written, copyErr := io.Copy(out, src)
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil || written != file.Size {
		os.Remove(dst)
		ctr.svc.DeleteByID(result.Code)
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

	// Generate one-time download token
	token := service.GenerateDownloadToken(code)
	utils.Success(c, gin.H{"download_token": token})
}

func (ctr *FileController) Download(c *gin.Context) {
	code := c.Param("code")

	token := c.Query("token")
	if token == "" {
		utils.Error(c, 403, "缺少下载凭证")
		return
	}

	if !service.ValidateDownloadToken(token, code) {
		utils.Error(c, 403, "下载凭证无效或已过期")
		return
	}

	filePath, fileName, err := ctr.svc.GetFilePath(code)
	if err != nil {
		utils.Error(c, 404, err.Error())
		return
	}

	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filepath.Base(fileName)}))
	c.Header("Content-Type", "application/octet-stream")
	c.File(filePath)
}

func (ctr *FileController) List(c *gin.Context) {
	token := c.GetHeader("X-Manage-Token")
	if token == "" {
		utils.Error(c, 401, "未授权")
		return
	}

	files, err := ctr.svc.ListByToken(token)
	if err != nil {
		utils.Error(c, 500, "获取文件列表失败")
		return
	}
	utils.Success(c, gin.H{"files": files})
}

func (ctr *FileController) Delete(c *gin.Context) {
	token := c.GetHeader("X-Manage-Token")
	if token == "" {
		utils.Error(c, 401, "未授权")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	if err := ctr.svc.Delete(uint(id), token); err != nil {
		utils.Error(c, 404, err.Error())
		return
	}
	utils.Success(c, nil)
}
