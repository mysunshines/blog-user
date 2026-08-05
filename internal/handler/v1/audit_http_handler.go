package v1

import (
	"strconv"

	"github.com/mysunshines/blog-user/internal/audit"
	"github.com/mysunshines/blog-user/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuditHandler 操作日志的 HTTP 处理器（仅管理端查询）。
type AuditHandler struct {
	db *gorm.DB
}

func NewAuditHandler(db *gorm.DB) *AuditHandler {
	return &AuditHandler{db: db}
}

// List 管理端分页查询操作日志
// @Summary 查询操作日志
// @Tags audit
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页大小"
// @Param action query string false "操作动作"
// @Param target_type query string false "目标类型"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/user/operation-logs [get]
func (h *AuditHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	action := c.Query("action")
	targetType := c.Query("target_type")
	operatorID, _ := strconv.ParseUint(c.Query("operator_id"), 10, 32)

	logs, total, err := audit.List(c.Request.Context(), h.db, page, pageSize, action, targetType, uint(operatorID))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, gin.H{
		"logs":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
