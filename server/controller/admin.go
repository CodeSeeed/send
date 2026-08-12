package controller

import (
	"strconv"

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
	token, err := ctr.adminSvc.Register(req.Username, req.Password)
	if err != nil {
		utils.Error(c, 409, err.Error())
		return
	}
	utils.Success(c, gin.H{"token": token})
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
	if err := ctr.adminSvc.ChangePassword(adminID, req.OldPassword, req.NewPassword); err != nil {
		utils.Error(c, 400, err.Error())
		return
	}
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
