package controller

import (
	"net/http"
	"strconv"
	"time"

	"send/server/service"
	"send/server/utils"

	"github.com/gin-gonic/gin"
)

type AdminController struct {
	adminSvc    *service.AdminService
	settingsSvc *service.SettingsService
}

func NewAdminController(adminSvc *service.AdminService, settingsSvc *service.SettingsService) *AdminController {
	return &AdminController{adminSvc: adminSvc, settingsSvc: settingsSvc}
}

func (ctr *AdminController) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	token, err := ctr.adminSvc.Login(req.Username, req.Password)
	if err != nil {
		utils.Error(c, 401, err.Error())
		return
	}
	setAdminCookie(c, token)
	utils.Success(c, gin.H{"token": token})
}

func (ctr *AdminController) AdminStatus(c *gin.Context) {
	hasAdmin, err := ctr.adminSvc.HasAdmin()
	if err != nil {
		utils.Error(c, 500, "获取管理员状态失败")
		return
	}
	utils.Success(c, gin.H{"registered": hasAdmin})
}

func (ctr *AdminController) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if len(req.Password) > 128 {
		utils.Error(c, 400, "密码长度不能超过128位")
		return
	}
	token, err := ctr.adminSvc.Register(req.Username, req.Password)
	if err != nil {
		utils.Error(c, 409, err.Error())
		return
	}
	setAdminCookie(c, token)
	utils.Success(c, gin.H{"token": token})
}

func (ctr *AdminController) Check(c *gin.Context) {
	// Auth middleware already verified the token; return success
	adminID := c.GetUint("admin_id")
	utils.Success(c, gin.H{"admin_id": adminID})
}

func (ctr *AdminController) Logout(c *gin.Context) {
	clearAdminCookie(c)
	utils.Success(c, nil)
}

func (ctr *AdminController) ChangePassword(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if req.NewPassword == "" {
		utils.Error(c, 400, "新密码不能为空")
		return
	}
	if len(req.NewPassword) > 128 {
		utils.Error(c, 400, "密码长度不能超过128位")
		return
	}
	if err := ctr.adminSvc.ChangePassword(adminID, req.OldPassword, req.NewPassword); err != nil {
		utils.Error(c, 400, err.Error())
		return
	}
	// Clear cookie so user re-logs in with new password
	clearAdminCookie(c)
	utils.Success(c, nil)
}

func (ctr *AdminController) ListFiles(c *gin.Context) {
	files, err := ctr.adminSvc.ListAllFiles()
	if err != nil {
		utils.Error(c, 500, "获取文件列表失败")
		return
	}
	utils.Success(c, gin.H{"files": files})
}

func (ctr *AdminController) DeleteFile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if err := ctr.adminSvc.DeleteFile(uint(id)); err != nil {
		utils.Error(c, 404, err.Error())
		return
	}
	utils.Success(c, nil)
}

func (ctr *AdminController) GetSettings(c *gin.Context) {
	settings, err := ctr.settingsSvc.Get()
	if err != nil {
		utils.Error(c, 500, "获取设置失败")
		return
	}
	utils.Success(c, settings)
}

func (ctr *AdminController) UpdateSettings(c *gin.Context) {
	var req struct {
		MaxFileSize int64  `json:"max_file_size"`
		BaseURL     string `json:"base_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if err := ctr.settingsSvc.Update(req.MaxFileSize, req.BaseURL); err != nil {
		utils.Error(c, 400, err.Error())
		return
	}
	settings, err := ctr.settingsSvc.Get()
	if err != nil {
		utils.Error(c, 500, "获取设置失败")
		return
	}
	utils.Success(c, settings)
}

func setAdminCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "admin_token",
		Value:    token,
		Path:     "/",
		MaxAge:   int(24 * time.Hour / time.Second),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func clearAdminCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "admin_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}