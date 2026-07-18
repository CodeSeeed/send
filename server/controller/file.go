package controller

import (
	"io"
	"os"
	"path/filepath"
	"strconv"

	"send/server/config"
	"send/server/service"
	"send/server/utils"

	"github.com/gin-gonic/gin"
)

type FileController struct {
	svc          *service.FileService
	cfg          *config.Config
	adminSvc     *service.AdminService
	usageCodeSvc *service.UsageCodeService
}

func NewFileController(svc *service.FileService, cfg *config.Config, adminSvc *service.AdminService, usageCodeSvc *service.UsageCodeService) *FileController {
	return &FileController{svc: svc, cfg: cfg, adminSvc: adminSvc, usageCodeSvc: usageCodeSvc}
}

func (ctr *FileController) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		utils.Error(c, 400, "请选择文件")
		return
	}

	if file.Size > ctr.cfg.Upload.MaxSize {
		utils.Error(c, 400, "文件过大，最大支持100MB")
		return
	}

	// Check admin token or usage code
	adminToken := c.GetHeader("X-Admin-Token")
	_, err = ctr.adminSvc.Auth(adminToken)
	if err != nil {
		// Not admin, require usage code
		usageCode := c.PostForm("usage_code")
		if usageCode == "" {
			utils.Error(c, 403, "请提供使用码")
			return
		}
		if err := ctr.usageCodeSvc.Verify(usageCode, file.Size); err != nil {
			utils.Error(c, 403, err.Error())
			return
		}
		// Consume one use
		ctr.usageCodeSvc.Use(usageCode)
	}

	password := c.PostForm("password")
	expireStr := c.PostForm("expire_hours")
	expireHours := 0
	if expireStr != "" {
		expireHours, _ = strconv.Atoi(expireStr)
	}

	params := &service.CreateFileParams{
		FileName:    file.Filename,
		FileSize:    file.Size,
		Password:    password,
		ExpireHours: expireHours,
	}

	result, err := ctr.svc.Create(params)
	if err != nil {
		utils.Error(c, 500, "上传失败")
		return
	}

	dst := filepath.Join(ctr.cfg.Upload.Dir, result.Code)
	src, err := file.Open()
	if err != nil {
		utils.Error(c, 500, "上传失败")
		return
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		utils.Error(c, 500, "上传失败")
		return
	}
	defer out.Close()

	io.Copy(out, src)

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

	c.Header("Content-Disposition", "attachment; filename=\""+filepath.Base(fileName)+"\"")
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
