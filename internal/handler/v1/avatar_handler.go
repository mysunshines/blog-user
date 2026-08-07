package v1

import (
	"github.com/mysunshines/blog-user/pkg/response"

	"github.com/mysunshines/gocommon/upload"

	"github.com/gin-gonic/gin"
)

// UploadsDir 上传文件落盘目录（与 article-service 共用 Docker 预创建的 /app/uploads）。
const UploadsDir = "/app/uploads"

// UserUploadsBaseURL 路由常量见 routes.go（与 proto/user.proto 契约对应）。

// UploadAvatar 接收当前登录用户的头像上传（multipart/form-data，字段名 file），
// 返回可访问的 URL。鉴权由路由上的 JWTValidMiddleware 保证，
// 实际写入用户 avatar 字段由前端拿到 URL 后调用 UpdateUser 完成（单一职责）。
//
// 核心落盘逻辑复用 gocommon/upload。
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "未选择文件或上传失败")
		return
	}

	res, err := upload.Save(UploadsDir, UserUploadsBaseURL, file, upload.Options{
		MaxBytes:    2 << 20, // 头像上限 2MB
		AllowedExts: upload.DefaultImageExts,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, map[string]string{"url": res.URL})
}
