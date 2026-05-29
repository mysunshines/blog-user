package handler

import (
	"strconv"

	"github.com/mysunshines/blog-user/internal/model"
	"github.com/mysunshines/blog-user/internal/service"
	"github.com/mysunshines/blog-user/pkg/response"

	"github.com/mysunshines/gocommon/constants"
	commonmiddleware "github.com/mysunshines/gocommon/middleware"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	svc service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// Register 用户注册
// @Summary 用户注册
// @Tags user
// @Accept json
// @Produce json
// @Param user body model.RegisterRequest true "注册信息"
// @Success 200 {object} response.LoginResponse
// @Router /api/v1/user/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	result, err := h.svc.Register(c.Request.Context(), &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessLogin(c, result.Token, result.User)
}

// Login 用户登录
// @Summary 用户登录
// @Tags user
// @Accept json
// @Produce json
// @Param user body model.LoginRequest true "登录信息"
// @Success 200 {object} response.LoginResponse
// @Router /api/v1/user/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	result, err := h.svc.Login(c.Request.Context(), &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessLogin(c, result.Token, result.User)
}

// Logout 用户登出
// @Summary 用户登出
// @Tags user
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} response.Response
// @Router /api/v1/user/logout [post]
func (h *UserHandler) Logout(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		response.Unauthorized(c, "未提供认证令牌")
		return
	}

	if err := h.svc.Logout(c.Request.Context(), &model.LogoutRequest{Token: token}); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessWithMessage(c, "登出成功", nil)
}

// GetUser 获取用户信息
// @Summary 获取用户信息
// @Tags user
// @Produce json
// @Param id query int false "用户ID"
// @Param username query string false "用户名"
// @Success 200 {object} response.Response
// @Router /api/v1/user [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Query("id")
	username := c.Query("username")

	var id uint
	if idStr != "" {
		parsedID, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			response.BadRequest(c, "无效的用户ID")
			return
		}
		id = uint(parsedID)
	}

	user, err := h.svc.GetUser(c.Request.Context(), id, username)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, user)
}

// UpdateUser 更新用户信息
// @Summary 更新用户信息
// @Tags user
// @Accept json
// @Produce json
// @Param user body model.UpdateUserRequest true "更新信息"
// @Success 200 {object} response.Response
// @Router /api/v1/user [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	var req model.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := getUserID(c)
	req.UserID = userID

	user, err := h.svc.UpdateUser(c.Request.Context(), &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, user)
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Tags user
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Router /api/v1/user/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	if err := h.svc.DeleteUser(c.Request.Context(), uint(id)); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetUsers 获取用户列表
// @Summary 获取用户列表
// @Tags user
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(20)
// @Param role query int false "角色"
// @Param status query int false "状态"
// @Success 200 {object} response.PageResponse
// @Router /api/v1/users [get]
func (h *UserHandler) GetUsers(c *gin.Context) {
	var req model.GetUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	users, total, err := h.svc.GetUsers(c.Request.Context(), &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.PageSuccess(c, users, total, int(req.Page), int(req.PageSize))
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Tags user
// @Accept json
// @Produce json
// @Param password body model.ChangePasswordRequest true "密码信息"
// @Success 200 {object} response.Response
// @Router /api/v1/user/password [post]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req model.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := getUserID(c)
	req.UserID = userID

	if err := h.svc.ChangePassword(c.Request.Context(), &req); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessWithMessage(c, "密码修改成功", nil)
}

// ValidateToken 验证token
// @Summary 验证token
// @Tags user
// @Accept json
// @Produce json
// @Param token body model.ValidateTokenRequest true "token"
// @Success 200 {object} response.TokenResponse
// @Router /api/v1/user/validate [post]
func (h *UserHandler) ValidateToken(c *gin.Context) {
	var req model.ValidateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	result, err := h.svc.ValidateToken(c.Request.Context(), req.Token)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessToken(c, result.Valid, result.UserID, result.Username)
}

// AddToBlacklist 添加到黑名单
// @Summary 添加到黑名单
// @Tags user
// @Accept json
// @Produce json
// @Param blacklist body model.BlacklistRequest true "黑名单信息"
// @Success 200 {object} response.Response
// @Router /api/v1/user/blacklist [post]
func (h *UserHandler) AddToBlacklist(c *gin.Context) {
	var req model.BlacklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.svc.AddToBlacklist(c.Request.Context(), &req); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessWithMessage(c, "添加成功", nil)
}

// RemoveFromBlacklist 从黑名单移除
// @Summary 从黑名单移除
// @Tags user
// @Accept json
// @Produce json
// @Param blacklist body model.BlacklistRequest true "黑名单信息"
// @Success 200 {object} response.Response
// @Router /api/v1/user/blacklist [delete]
func (h *UserHandler) RemoveFromBlacklist(c *gin.Context) {
	var req model.BlacklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.svc.RemoveFromBlacklist(c.Request.Context(), &req); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessWithMessage(c, "移除成功", nil)
}

// IsInBlacklist 检查是否在黑名单
// @Summary 检查是否在黑名单
// @Tags user
// @Produce json
// @Param user_id query int true "用户ID"
// @Success 200 {object} response.Response
// @Router /api/v1/user/blacklist/check [get]
func (h *UserHandler) IsInBlacklist(c *gin.Context) {
	userIDStr := c.Query("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	targetUserIDStr := c.Query("target_user_id")
	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的目标用户ID")
		return
	}

	inBlacklist, err := h.svc.IsInBlacklist(c.Request.Context(), uint(userID), uint(targetUserID))
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, gin.H{"in_blacklist": inBlacklist})
}

// 辅助函数
func extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) > constants.JWTAuthSchemeLen && authHeader[:constants.JWTAuthSchemeLen] == constants.JWTAuthScheme {
		return authHeader[constants.JWTAuthSchemeLen:]
	}
	return ""
}

func getUserID(c *gin.Context) uint {
	if userID, exists := c.Get(commonmiddleware.UserIDContextKey); exists {
		switch v := userID.(type) {
		case uint:
			return v
		case uint32:
			return uint(v)
		case uint64:
			return uint(v)
		case float64:
			return uint(v)
		case int:
			return uint(v)
		case int32:
			return uint(v)
		case int64:
			return uint(v)
		}
	}
	return 0
}
