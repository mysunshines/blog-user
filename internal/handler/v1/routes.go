package v1

import gcommon "github.com/mysunshines/gocommon/constants"

// 本文件集中定义 user-service 的 HTTP 路由常量，
// 与 proto/user.proto 顶部的 HTTP API Contract 注释一一对应，
// 作为路由路径的单一事实来源（main.go 与各 handler 均引用此处，勿在别处硬编码）。
//
// 版本化前缀统一复用 gocommon/constants.APIPathPrefix（"/api/v1"），不在此重复定义。

const (
	// UserGroup 用户资源组前缀。
	// 完整路径：/api/v1/user/*
	UserGroup = "/user"

	// UserRegisterPath 注册子路径（公开，含 verify_code 字段，proto 无对应方法）。
	// 完整路径：POST /api/v1/user/register
	UserRegisterPath = "/register"

	// UserSendCodePath 发送验证码子路径（公开，proto 无对应方法）。
	// 完整路径：POST /api/v1/user/send-code
	UserSendCodePath = "/send-code"

	// UserAvatarPath 头像上传子路径（multipart field=file，需登录）。
	// 完整路径：POST /api/v1/user/avatar
	UserAvatarPath = "/avatar"

	// UserAdminGroup 后台管理资源组前缀（经 Gateway /admin-api 透传）。
	// 完整路径：/api/v1/admin/user/*
	UserAdminGroup = "/admin/user"

	// UserUploadsBaseURL 头像图片静态托管对外 URL 前缀。
	// 完整路径：GET /api/v1/user/uploads/*
	UserUploadsBaseURL = gcommon.APIPathPrefix + "/user/uploads"
)
