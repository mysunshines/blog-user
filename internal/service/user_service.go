package service

import (
	"context"
	stderrors "errors"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/mysunshines/blog-user/internal/errors"
	"github.com/mysunshines/blog-user/internal/model"
	"github.com/mysunshines/blog-user/internal/repository"
	goconfig "github.com/mysunshines/gocommon/config"
	"github.com/mysunshines/gocommon/constants"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mysunshines/gocommon/cache"
	"github.com/mysunshines/gocommon/log"
	"github.com/mysunshines/gocommon/notify"
	"github.com/mysunshines/gocommon/util"
	"golang.org/x/crypto/bcrypt"
)

// UserService 用户服务接口
type UserService interface {
	// 认证相关
	Register(ctx context.Context, req *model.RegisterRequest) (*model.AuthResponse, error)
	SendVerificationCode(ctx context.Context, req *model.SendVerifyCodeRequest) error
	Login(ctx context.Context, req *model.LoginRequest) (*model.AuthResponse, error)
	Logout(ctx context.Context, req *model.LogoutRequest) error
	ValidateToken(ctx context.Context, token string) (*ValidateTokenResult, error)

	// 用户操作
	GetUser(ctx context.Context, userID uint, username string) (*model.UserResponse, error)
	UpdateUser(ctx context.Context, req *model.UpdateUserRequest) (*model.UserResponse, error)
	DeleteUser(ctx context.Context, userID uint) error
	GetUsers(ctx context.Context, req *model.GetUsersRequest) ([]*model.UserResponse, int64, error)
	ChangePassword(ctx context.Context, req *model.ChangePasswordRequest) error

	// 管理员操作
	AdminGetUsers(ctx context.Context, req *model.GetUsersRequest) ([]*model.UserResponse, int64, error)
	AdminUpdateUser(ctx context.Context, req *model.AdminUpdateUserRequest) (*model.UserResponse, error)
	AdminDeleteUser(ctx context.Context, id uint) error

	// 黑名单操作
	AddToBlacklist(ctx context.Context, req *model.BlacklistRequest) error
	RemoveFromBlacklist(ctx context.Context, req *model.BlacklistRequest) error
	IsInBlacklist(ctx context.Context, userID, targetUserID uint) (bool, error)
}

// ValidateTokenResult Token验证结果
type ValidateTokenResult struct {
	UserID   uint
	Username string
	Valid    bool
}

// userService 用户服务实现
type userService struct {
	repo    repository.UserRepository
	cfg     *goconfig.Config
	bf      *cache.BloomFilter // gocommon Redis 布隆过滤器，缓存全量历史用户名
	bfReady atomic.Bool        // 布隆过滤器预热完成标记，未就绪时注册走 DB 全量查重
}

// NewUserService 创建用户服务
func NewUserService(repo repository.UserRepository, cfg *goconfig.Config) UserService {
	// Redis 布隆过滤器：注册时快速预判 username 是否已存在，命中/未命中逻辑见 Register。
	// 容量按目标量级用公式估算（m = -n·ln(p)/(ln2)², k = 0.693·m/n）：
	//   10 万用户 @1% 误判：m≈96万bit(~120KB)、k=7
	//   1000 万用户 @1% 误判：m≈9585万bit(~11.4MB)、k=7
	//   1 亿用户 @1% 误判：m≈9.6亿bit(~114MB)、k=7
	// 本项目按千万级预估取 m=1亿bit(~12MB)、k=7；量级变化时按公式调整即可。
	// key 自动带 KeyPrefix。
	bf := cache.NewBloomFilter("user:username:bloom", 100_000_000, 7)
	s := &userService{
		repo: repo,
		cfg:  cfg,
		bf:   bf,
	}

	// 异步预热布隆过滤器（千万级用户量若同步预热，全量加载内存 + 逐条 Add RTT
	// 会拖垮启动）。预热完成前 bfReady=false，Register 走 DB 全量查重，语义与
	// 未接入布隆时完全一致，预热完毕自动切换到布隆快路径。
	go s.warmUpBloomFilter()
	return s
}

// warmUpBloomFilter 按 id 游标分批从 DB 灌入全量历史用户名：
// 每批一次 pipeline 往返（AddBulk），避免全量加载内存与逐条网络往返。
func (s *userService) warmUpBloomFilter() {
	ctx := context.Background()
	const pageSize = 5000 // 每批 5000 用户名 × 7 哈希 = 3.5 万条 SetBit，单次 pipeline 往返
	var (
		afterID uint
		added   int
	)
	for {
		names, nextID, err := s.repo.ListUsernamesByPage(ctx, afterID, pageSize)
		if err != nil {
			log.Warnf("warm up bloom filter page failed (after id=%d): %v", afterID, err)
			break
		}
		if len(names) == 0 {
			break
		}
		if err := s.bf.AddBulk(ctx, names); err != nil {
			log.Warnf("warm up bloom filter bulk add failed: %v", err)
			break
		}
		added += len(names)
		afterID = nextID
		if len(names) < pageSize {
			break
		}
	}
	s.bfReady.Store(true)
	log.Infof("username bloom filter warmed up, %d entries", added)
}

// verifyCodeKeyPrefix Redis 验证码 key 前缀
const verifyCodeKeyPrefix = "verify_code:"

// verifyCodeTTL 验证码有效期
const verifyCodeTTL = 5 * time.Minute

// Register 用户注册
func (s *userService) Register(ctx context.Context, req *model.RegisterRequest) (*model.AuthResponse, error) {
	// 检查用户名和邮箱格式
	if !isValidUsername(req.Username) {
		return nil, errors.New(errors.CodeUnauthorized, "用户名格式不正确，支持中英文、数字、下划线，2-32个字符")
	}
	if !isValidEmail(req.Email) {
		return nil, errors.New(errors.CodeUnauthorized, "邮箱格式不正确")
	}
	if !isValidPassword(req.Password) {
		return nil, errors.New(errors.CodeUnauthorized, "密码需包含字母和数字，6-32个字符")
	}

	// 验证验证码
	if req.VerifyCode == "" {
		return nil, errors.New(errors.CodeUnauthorized, "请输入邮箱验证码")
	}
	storedCode, err := cache.Get(ctx, verifyCodeKeyPrefix+req.Email)
	if err != nil || storedCode == "" {
		return nil, errors.New(errors.CodeUnauthorized, "验证码已过期，请重新获取")
	}
	if storedCode != req.VerifyCode {
		return nil, errors.New(errors.CodeUnauthorized, "验证码不正确")
	}
	// 验证通过后删除验证码
	_ = cache.Delete(ctx, verifyCodeKeyPrefix+req.Email)

	if !s.bfReady.Load() {
		// 预热窗口期：退化为 DB 全量查重（username OR email），语义与未接入布隆时一致
		exists, err := s.repo.ExistsByUsernameOrEmail(ctx, req.Username, req.Email)
		if err != nil {
			return nil, errors.New(errors.CodeRegisterFailed, "Failed to check user existence")
		}
		if exists {
			return nil, errors.New(errors.CodeUserAlreadyExists, "User already exists")
		}
	} else {
		// 布隆过滤器快速预判 username（启动时已预热全量历史用户名）：
		// - 未命中 → username 一定不存在，跳过 username 维度查库（快路径，省一次 DB 查询）
		// - 命中 → 可能已存在（布隆只误报"存在"、不漏报"不存在"），查库确认
		// - Redis 不可用 → 降级为 DB 全量查重（username OR email），保证语义不缩水
		if hit, err := s.bf.Exists(ctx, req.Username); err != nil {
			exists, dbErr := s.repo.ExistsByUsernameOrEmail(ctx, req.Username, req.Email)
			if dbErr != nil {
				return nil, errors.New(errors.CodeRegisterFailed, "Failed to check user existence")
			}
			if exists {
				return nil, errors.New(errors.CodeUserAlreadyExists, "User already exists")
			}
		} else if hit {
			if _, e := s.repo.GetByUsername(ctx, req.Username); e == nil {
				return nil, errors.New(errors.CodeUserAlreadyExists, "User already exists")
			}
		}
		// email 唯一性必须始终查库（布隆过滤器仅覆盖 username 维度）
		if _, err := s.repo.GetByEmail(ctx, req.Email); err == nil {
			return nil, errors.New(errors.CodeUserAlreadyExists, "User already exists")
		}
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New(errors.CodeRegisterFailed, "Failed to hash password")
	}

	// 创建用户
	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
		Nickname: req.Nickname,
		Avatar:   generateAvatar(req.Username),
		Role:     model.UserRoleNormal,
		Status:   model.UserStatusNormal,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, errors.New(errors.CodeRegisterFailed, "Failed to create user")
	}

	// 写入布隆过滤器（失败仅影响后续预判，DB 唯一索引仍兜底）
	if err := s.bf.Add(ctx, req.Username); err != nil {
		log.Warnf("Failed to add username to bloom filter: %v", err)
	}

	// 生成Token
	token, csrfToken, err := s.generateToken(user)
	if err != nil {
		return nil, errors.New(errors.CodeRegisterFailed, "Failed to generate token")
	}

	// 保存Token
	if err := s.saveToken(ctx, user.ID, token); err != nil {
		log.Warnf("Failed to save token for user %d: %v", user.ID, err)
	}

	return &model.AuthResponse{
		Token:     token,
		CSRFToken: csrfToken,
		User:      user.ToResponsePtr(),
	}, nil
}

// SendVerificationCode 发送邮箱验证码
func (s *userService) SendVerificationCode(ctx context.Context, req *model.SendVerifyCodeRequest) error {
	if !isValidEmail(req.Email) {
		return errors.New(errors.CodeUnauthorized, "邮箱格式不正确")
	}

	// 检查邮箱是否已被注册
	_, err := s.repo.GetByEmail(ctx, req.Email)
	if err == nil {
		return errors.New(errors.CodeUserAlreadyExists, "该邮箱已被注册")
	}
	if !stderrors.Is(err, repository.ErrNotFound) {
		return errors.New(errors.CodeInternal, "服务器内部错误")
	}

	// 限制发送频率：60秒内只能发送一次
	rateLimitKey := "verify_code_rate:" + req.Email
	ok, setErr := cache.SetNX(ctx, rateLimitKey, "1", 60*time.Second)
	if setErr == nil && !ok {
		return errors.New(errors.CodeTooManyRequests, "验证码发送过于频繁，请60秒后再试")
	}

	// 生成6位随机验证码
	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	// 存入Redis，5分钟过期
	if err := cache.Set(ctx, verifyCodeKeyPrefix+req.Email, code, verifyCodeTTL); err != nil {
		log.Errorf("Failed to store verify code for %s: %v", req.Email, err)
		return errors.New(errors.CodeInternal, "验证码存储失败")
	}

	// 发送邮件
	cfg := notify.Config{
		Host:     s.cfg.Mail.SMTPHost,
		Port:     s.cfg.Mail.SMTPPort,
		Username: s.cfg.Mail.SMTPUsername,
		Password: s.cfg.Mail.SMTPPassword,
		From:     s.cfg.Mail.FromAddress,
		FromName: "Blog",
	}
	msg := notify.Message{
		From:     s.cfg.Mail.FromAddress,
		FromName: "Blog",
		To:       []string{req.Email},
		Subject:  "Blog - 邮箱验证码",
		HTMLBody: fmt.Sprintf(`
		<div style="max-width:600px;margin:0 auto;padding:20px;font-family:Arial,sans-serif;">
			<h2 style="color:#333;">邮箱验证码</h2>
			<p>您的验证码是：</p>
			<div style="font-size:28px;font-weight:bold;color:#1890ff;padding:12px 24px;background:#f0f5ff;display:inline-block;border-radius:4px;letter-spacing:4px;">%s</div>
			<p style="color:#999;margin-top:20px;">验证码5分钟内有效，请勿泄露给他人。</p>
			<p style="color:#999;">如果不是您本人操作，请忽略此邮件。</p>
		</div>`, code),
	}

	if err := notify.Send(ctx, cfg, msg); err != nil {
		log.Errorf("Failed to send verify code email to %s: %v", req.Email, err)
		// 邮件发送失败时删除已存储的验证码
		_ = cache.Delete(ctx, verifyCodeKeyPrefix+req.Email)
		return errors.New(errors.CodeInternal, "邮件发送失败，请稍后重试")
	}

	log.Infof("Verification code sent to %s", req.Email)
	return nil
}

// Login 用户登录
func (s *userService) Login(ctx context.Context, req *model.LoginRequest) (*model.AuthResponse, error) {
	// 获取用户
	user, err := s.repo.GetByUsernameOrEmail(ctx, req.Username, req.Username)
	if err != nil {
		if stderrors.Is(err, repository.ErrNotFound) {
			return nil, errors.New(errors.CodeUserAlreadyExists, "Invalid username or password")
		}
		return nil, errors.New(errors.CodeLoginFailed, "Failed to get user")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New(errors.CodePasswordIncorrect, "Invalid username or password")
	}

	// 检查用户状态
	if user.Status != model.UserStatusNormal {
		return nil, errors.New(errors.CodeForbidden, "User account is disabled")
	}

	// 生成Token
	token, csrfToken, err := s.generateToken(user)
	if err != nil {
		return nil, errors.New(errors.CodeLoginFailed, "Failed to generate token")
	}

	// 保存Token
	if err := s.saveToken(ctx, user.ID, token); err != nil {
		log.Warnf("Failed to save token for user %d: %v", user.ID, err)
	}

	return &model.AuthResponse{
		Token:     token,
		CSRFToken: csrfToken,
		User:      user.ToResponsePtr(),
	}, nil
}

// Logout 用户登出
func (s *userService) Logout(ctx context.Context, req *model.LogoutRequest) error {
	// 删除Token
	if err := s.repo.DeleteToken(ctx, req.Token); err != nil {
		return errors.New(errors.CodeInternal, "Failed to logout")
	}
	return nil
}

// ValidateToken 验证Token
func (s *userService) ValidateToken(ctx context.Context, token string) (*ValidateTokenResult, error) {
	// 解析JWT
	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, stderrors.New("unexpected signing method")
		}
		return []byte(s.cfg.JWT.Secret), nil
	})

	if err != nil || !parsedToken.Valid {
		return &ValidateTokenResult{Valid: false}, nil
	}

	// 获取Token记录
	tokenRecord, err := s.repo.GetToken(ctx, token)
	if err != nil {
		if stderrors.Is(err, repository.ErrNotFound) {
			return &ValidateTokenResult{Valid: false}, nil
		}
		return &ValidateTokenResult{Valid: false}, nil
	}

	// 检查Token是否过期
	if time.Now().After(tokenRecord.ExpiresAt) {
		return &ValidateTokenResult{Valid: false}, nil
	}

	// 获取用户信息
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return &ValidateTokenResult{Valid: false}, nil
	}

	userID := uint(claims[constants.JWTClaimUserID].(float64))
	username := claims[constants.JWTClaimUsername].(string)

	return &ValidateTokenResult{
		UserID:   userID,
		Username: username,
		Valid:    true,
	}, nil
}

// GetUser 获取用户信息
func (s *userService) GetUser(ctx context.Context, userID uint, username string) (*model.UserResponse, error) {
	var user *model.User
	var err error

	if userID > 0 {
		user, err = s.repo.GetByID(ctx, userID)
	} else if username != "" {
		user, err = s.repo.GetByUsername(ctx, username)
	} else {
		return nil, errors.New(errors.CodeUnauthorized, "user_id or username required")
	}

	if err != nil {
		if stderrors.Is(err, repository.ErrNotFound) {
			return nil, errors.New(errors.CodeUserNotFound, "User not found")
		}
		return nil, errors.New(errors.CodeInternal, "Failed to get user")
	}

	return user.ToResponsePtr(), nil
}

// UpdateUser 更新用户信息
func (s *userService) UpdateUser(ctx context.Context, req *model.UpdateUserRequest) (*model.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, req.UserID)
	if err != nil {
		if stderrors.Is(err, repository.ErrNotFound) {
			return nil, errors.New(errors.CodeUserNotFound, "User not found")
		}
		return nil, errors.New(errors.CodeInternal, "Failed to get user")
	}

	// 更新字段
	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.Bio != "" {
		user.Bio = req.Bio
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, errors.New(errors.CodeUserUpdateFailed, "Failed to update user")
	}

	return user.ToResponsePtr(), nil
}

// DeleteUser 删除用户
func (s *userService) DeleteUser(ctx context.Context, userID uint) error {
	// 删除用户的Token
	if err := s.repo.DeleteUserTokens(ctx, userID); err != nil {
		log.Warnf("Failed to delete tokens for user %d: %v", userID, err)
	}

	// 删除用户
	if err := s.repo.Delete(ctx, userID); err != nil {
		if stderrors.Is(err, repository.ErrNotFound) {
			return errors.New(errors.CodeUserNotFound, "User not found")
		}
		return errors.New(errors.CodeUserDeleteFailed, "Failed to delete user")
	}

	return nil
}

// GetUsers 获取用户列表
func (s *userService) GetUsers(ctx context.Context, req *model.GetUsersRequest) ([]*model.UserResponse, int64, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	users, total, err := s.repo.List(ctx, page, pageSize, req.Role, req.Status)
	if err != nil {
		return nil, 0, errors.New(errors.CodeInternal, "Failed to list users")
	}

	responses := make([]*model.UserResponse, len(users))
	for i, u := range users {
		responses[i] = u.ToResponsePtr()
	}

	return responses, total, nil
}

// ChangePassword 修改密码
func (s *userService) ChangePassword(ctx context.Context, req *model.ChangePasswordRequest) error {
	user, err := s.repo.GetByID(ctx, req.UserID)
	if err != nil {
		if stderrors.Is(err, repository.ErrNotFound) {
			return errors.New(errors.CodeUserNotFound, "User not found")
		}
		return errors.New(errors.CodeInternal, "Failed to get user")
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return errors.New(errors.CodePasswordIncorrect, "Old password incorrect")
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New(errors.CodeInternal, "Failed to hash password")
	}

	// 更新密码
	user.Password = string(hashedPassword)
	if err := s.repo.Update(ctx, user); err != nil {
		return errors.New(errors.CodeUserUpdateFailed, "Failed to update password")
	}

	return nil
}

// AddToBlacklist 添加到黑名单
func (s *userService) AddToBlacklist(ctx context.Context, req *model.BlacklistRequest) error {
	blacklist := &model.UserBlacklist{
		UserID:        req.UserID,
		BlockedUserID: req.TargetUserID,
		Reason:        req.Reason,
	}

	if err := s.repo.AddToBlacklist(ctx, blacklist); err != nil {
		return errors.New(errors.CodeInternal, "Failed to add to blacklist")
	}

	return nil
}

// RemoveFromBlacklist 从黑名单移除
func (s *userService) RemoveFromBlacklist(ctx context.Context, req *model.BlacklistRequest) error {
	if err := s.repo.RemoveFromBlacklist(ctx, req.UserID, req.TargetUserID); err != nil {
		return errors.New(errors.CodeInternal, "Failed to remove from blacklist")
	}
	return nil
}

// IsInBlacklist 检查是否在黑名单中
func (s *userService) IsInBlacklist(ctx context.Context, userID, targetUserID uint) (bool, error) {
	inBlacklist, err := s.repo.IsInBlacklist(ctx, userID, targetUserID)
	if err != nil {
		return false, errors.New(errors.CodeInternal, "Failed to check blacklist")
	}
	return inBlacklist, nil
}

// AdminUpdateUser 管理员更新用户信息（可修改角色、状态等）
func (s *userService) AdminUpdateUser(ctx context.Context, req *model.AdminUpdateUserRequest) (*model.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, req.UserID)
	if err != nil {
		if stderrors.Is(err, repository.ErrNotFound) {
			return nil, errors.New(errors.CodeUserNotFound, "User not found")
		}
		return nil, errors.New(errors.CodeInternal, "Failed to get user")
	}

	// 构造需更新的列（用 map 避免结构体 Updates 跳过零值，如禁用 status=0）
	cols := map[string]interface{}{}
	if req.Nickname != "" {
		cols["nickname"] = req.Nickname
	}
	if req.Role != nil {
		cols["role"] = *req.Role
	}
	if req.Status != nil {
		cols["status"] = *req.Status
	}

	if err := s.repo.UpdateFields(ctx, req.UserID, cols); err != nil {
		return nil, errors.New(errors.CodeUserUpdateFailed, "Failed to update user")
	}

	return user.ToResponsePtr(), nil
}

// AdminGetUsers 管理员查询用户列表（复用 GetUsers，支持按角色/状态/关键字过滤）
func (s *userService) AdminGetUsers(ctx context.Context, req *model.GetUsersRequest) ([]*model.UserResponse, int64, error) {
	return s.GetUsers(ctx, req)
}

// AdminDeleteUser 管理员删除用户（复用 DeleteUser）
func (s *userService) AdminDeleteUser(ctx context.Context, id uint) error {
	return s.DeleteUser(ctx, id)
}

// generateToken 生成JWT Token，并返回随 JWT 下发的 CSRF token。
// csrf 声明写入 JWT，前端在已登录的状态变更请求中通过 X-CSRF-Token 头回传，由 CSRFMiddleware 校验。
func (s *userService) generateToken(user *model.User) (string, string, error) {
	csrfToken := util.GenerateRandomString(32)
	claims := jwt.MapClaims{
		constants.JWTClaimUserID:   user.ID,
		constants.JWTClaimUsername: user.Username,
		constants.JWTClaimRole:     user.Role,
		constants.JWTClaimCSRF:     csrfToken,
		"exp":                      time.Now().Add(time.Duration(s.cfg.JWT.ExpireTime) * time.Second).Unix(),
		"iat":                      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return "", "", err
	}
	return signed, csrfToken, nil
}

// saveToken 保存Token
func (s *userService) saveToken(ctx context.Context, userID uint, token string) error {
	tokenRecord := &model.Token{
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(time.Duration(s.cfg.JWT.ExpireTime) * time.Second),
	}
	return s.repo.CreateToken(ctx, tokenRecord)
}

// ============ 辅助函数 ============

// isValidUsername 验证用户名，支持中文、英文、数字、下划线
func isValidUsername(username string) bool {
	// 对中文字符按 rune 计长，对 ASCII 按实际字节，确保两者计长一致
	runeLen := len([]rune(username))
	if runeLen < 2 || runeLen > 32 {
		return false
	}
	for _, r := range username {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			continue
		}
		return false
	}
	return true
}

// isValidEmail 验证邮箱
func isValidEmail(email string) bool {
	if len(email) < 3 || len(email) > 128 {
		return false
	}
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			if atIndex != -1 {
				return false // 多个@
			}
			atIndex = i
		}
	}
	return atIndex > 0 && atIndex < len(email)-1
}

// isValidPassword 验证密码
func isValidPassword(password string) bool {
	if len(password) < 6 || len(password) > 32 {
		return false
	}
	hasLetter := false
	hasNumber := false
	for _, c := range password {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			hasLetter = true
		}
		if c >= '0' && c <= '9' {
			hasNumber = true
		}
	}
	return hasLetter && hasNumber
}

// generateAvatar 生成默认头像
func generateAvatar(username string) string {
	return "https://api.dicebear.com/7.x/avataaars/svg?seed=" + username
}
