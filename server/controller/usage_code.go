package controller

import (
	"strconv"

	"send/server/service"
	"send/server/utils"

	"github.com/gin-gonic/gin"
)

type UsageCodeController struct {
	svc *service.UsageCodeService
}

func NewUsageCodeController(svc *service.UsageCodeService) *UsageCodeController {
	return &UsageCodeController{svc: svc}
}

type createReq struct {
	Remark      string `json:"remark"`
	MaxFileSize string `json:"max_file_size"`
	MaxUses     int    `json:"max_uses"`
	ExpireHours int    `json:"expire_hours"`
}

func (ctr *UsageCodeController) Create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if req.Remark == "" {
		utils.Error(c, 400, "请输入备注")
		return
	}
	if req.MaxUses <= 0 && req.ExpireHours <= 0 {
		utils.Error(c, 400, "必须设置使用次数或过期时间")
		return
	}
	maxFileSize, err := utils.ParseSizeString(req.MaxFileSize)
	if err != nil {
		utils.Error(c, 400, err.Error())
		return
	}
	code, err := ctr.svc.Create(req.Remark, maxFileSize, req.MaxUses, req.ExpireHours)
	if err != nil {
		utils.Error(c, 500, "创建失败")
		return
	}
	utils.Success(c, gin.H{"code": code})
}

func (ctr *UsageCodeController) List(c *gin.Context) {
	list, err := ctr.svc.List()
	if err != nil {
		utils.Error(c, 500, "获取使用码列表失败")
		return
	}
	utils.Success(c, gin.H{"codes": list})
}

func (ctr *UsageCodeController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if err := ctr.svc.Delete(uint(id)); err != nil {
		utils.Error(c, 404, err.Error())
		return
	}
	utils.Success(c, nil)
}
