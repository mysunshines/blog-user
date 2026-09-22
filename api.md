# user-service 对外 API 文档

> 自动生成自 `user.proto`（模式：proto）。
> 网关按 `/api/v1/user/<snake_method>` 反射代理到 gRPC 方法 `user.v1.UserService/<Method>`。
> 生成时间：2026-09-21 19:37:19
> Base URL（网关入口）：http://localhost:8081

## 接口列表

| Method | Path | 鉴权 | 说明 |
| --- | --- | --- | --- |
| `POST` | `/api/v1/user/register` | 公开 |  |
| `POST` | `/api/v1/user/login` | 公开 |  |
| `GET` | `/api/v1/user/generate_captcha` | 登录 | GenerateCaptcha 生成图形验证码（防机器人/脚本），前端在注册、登录页展示并要求用户输入。 |
| `POST` | `/api/v1/user/logout` | 公开 |  |
| `GET` | `/api/v1/user/get_user` | 公开 |  |
| `GET` | `/api/v1/user/validate_token` | 公开 |  |
| `PUT` | `/api/v1/user/update_user` | 公开 |  |
| `DELETE` | `/api/v1/user/delete_user` | 公开 |  |
| `GET` | `/api/v1/user/get_users` | 公开 |  |
| `PUT` | `/api/v1/user/change_password` | 公开 |  |
| `POST` | `/api/v1/user/add_to_blacklist` | 公开 |  |
| `DELETE` | `/api/v1/user/remove_from_blacklist` | 公开 |  |
| `GET` | `/api/v1/user/is_in_blacklist` | 公开 |  |
| `POST` | `/api/v1/user/admin_get_users` | 管理员 |  |
| `PUT` | `/api/v1/user/admin_update_user` | 公开 |  |
| `POST` | `/api/v1/user/admin_delete_user` | 公开 |  |
| `GET` | `/api/v1/user/list_operation_logs` | 公开 |  |
| `POST` | `/api/v1/user/record_log` | 公开 |  |
| `POST` | `/api/v1/user/follow` | 登录 | 关注 / 取关：follower_id 关注 following_id（不能关注自己，幂等）。需登录。 |
| `POST` | `/api/v1/user/unfollow` | 公开 |  |
| `GET` | `/api/v1/user/get_follow_stats` | 公开 | 某用户的粉丝数 / 关注数（公开只读）。 |
| `GET` | `/api/v1/user/get_follow_status` | 登录 | 当前用户是否关注某用户（follower_id 与 following_id）。需登录。 |

## Register

- **URL**: `http://localhost:8081/api/v1/user/register`
- **Method**: `POST`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `username` | `string` |  | `""` |
| `email` | `string` |  | `""` |
| `password` | `string` |  | `""` |
| `nickname` | `string` |  | `""` |
| `captcha_id` | `string` |  | `""` |
| `captcha_code` | `string` |  | `""` |

**Body 示例**：
```json
{"username": "", "email": "", "password": "", "nickname": "", "captcha_id": "", "captcha_code": ""}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `user` | `User` |  | 见 [User](#user) |
| `token` | `string` |  | `""` |
| `csrf_token` | `string` |  | `""` |

**Response 示例**：
```json
{"code": 0, "message": "success", "user": {}, "token": "", "csrf_token": ""}
```

### curl 示例
```bash
curl -X POST 'http://localhost:8081/api/v1/user/register' \
  -H 'Content-Type: application/json' \
  -d '{"username": "", "email": "", "password": "", "nickname": "", "captcha_id": "", "captcha_code": ""}'
```

## Login

- **URL**: `http://localhost:8081/api/v1/user/login`
- **Method**: `POST`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `username` | `string` |  | `""` |
| `password` | `string` |  | `""` |
| `captcha_id` | `string` |  | `""` |
| `captcha_code` | `string` |  | `""` |

**Body 示例**：
```json
{"username": "", "password": "", "captcha_id": "", "captcha_code": ""}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `user` | `User` |  | 见 [User](#user) |
| `token` | `string` |  | `""` |
| `csrf_token` | `string` |  | `""` |

**Response 示例**：
```json
{"code": 0, "message": "success", "user": {}, "token": "", "csrf_token": ""}
```

### curl 示例
```bash
curl -X POST 'http://localhost:8081/api/v1/user/login' \
  -H 'Content-Type: application/json' \
  -d '{"username": "", "password": "", "captcha_id": "", "captcha_code": ""}'
```

## GenerateCaptcha

- **URL**: `http://localhost:8081/api/v1/user/generate_captcha`
- **Method**: `GET`
- **鉴权**: 登录（需 JWT）

### Headers
```http
Authorization: Bearer <token>
Content-Type: application/json
```

### Request
无请求体 / 无参数。

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `captcha_id` | `string` | 校验时回传 | `""` |
| `image_base64` | `string` | data:image/png;base64,... 可直接用于 <img src> | `""` |

**Response 示例**：
```json
{"code": 0, "message": "success", "captcha_id": "", "image_base64": ""}
```

### curl 示例
```bash
curl -X GET 'http://localhost:8081/api/v1/user/generate_captcha' \
  -H 'Authorization: Bearer <token>'
```

## Logout

- **URL**: `http://localhost:8081/api/v1/user/logout`
- **Method**: `POST`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `user_id` | `uint32` |  | `0` |
| `token` | `string` |  | `""` |

**Body 示例**：
```json
{"user_id": 0, "token": ""}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |

**Response 示例**：
```json
{"code": 0, "message": "success"}
```

### curl 示例
```bash
curl -X POST 'http://localhost:8081/api/v1/user/logout' \
  -H 'Content-Type: application/json' \
  -d '{"user_id": 0, "token": ""}'
```

## GetUser

- **URL**: `http://localhost:8081/api/v1/user/get_user?user_id=0&username=<username>`
- **Method**: `GET`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Query String

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `user_id` | `uint32` |  | `0` |
| `username` | `string` |  | `""` |

**Query 示例**：
```json
user_id=0&username=<username>
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `user` | `User` |  | 见 [User](#user) |

**Response 示例**：
```json
{"code": 0, "message": "success", "user": {}}
```

### curl 示例
```bash
curl -X GET 'http://localhost:8081/api/v1/user/get_user?user_id=0&username=<username>'
```

## ValidateToken

- **URL**: `http://localhost:8081/api/v1/user/validate_token?token=<token>`
- **Method**: `GET`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Query String

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `token` | `string` |  | `""` |

**Query 示例**：
```json
token=<token>
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `user_id` | `uint32` |  | `0` |
| `username` | `string` |  | `""` |
| `valid` | `bool` |  | `false` |

**Response 示例**：
```json
{"code": 0, "message": "success", "user_id": 0, "username": "", "valid": false}
```

### curl 示例
```bash
curl -X GET 'http://localhost:8081/api/v1/user/validate_token?token=<token>'
```

## UpdateUser

- **URL**: `http://localhost:8081/api/v1/user/update_user`
- **Method**: `PUT`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `user_id` | `uint32` |  | `0` |
| `nickname` | `string` |  | `""` |
| `avatar` | `string` |  | `""` |
| `bio` | `string` |  | `""` |

**Body 示例**：
```json
{"user_id": 0, "nickname": "", "avatar": "", "bio": ""}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `user` | `User` |  | 见 [User](#user) |

**Response 示例**：
```json
{"code": 0, "message": "success", "user": {}}
```

### curl 示例
```bash
curl -X PUT 'http://localhost:8081/api/v1/user/update_user' \
  -H 'Content-Type: application/json' \
  -d '{"user_id": 0, "nickname": "", "avatar": "", "bio": ""}'
```

## DeleteUser

- **URL**: `http://localhost:8081/api/v1/user/delete_user`
- **Method**: `DELETE`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `user_id` | `uint32` |  | `0` |

**Body 示例**：
```json
{"user_id": 0}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |

**Response 示例**：
```json
{"code": 0, "message": "success"}
```

### curl 示例
```bash
curl -X DELETE 'http://localhost:8081/api/v1/user/delete_user' \
  -H 'Content-Type: application/json' \
  -d '{"user_id": 0}'
```

## GetUsers

- **URL**: `http://localhost:8081/api/v1/user/get_users?page=0&page_size=0&role=0`
- **Method**: `GET`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Query String

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `page` | `uint32` |  | `0` |
| `page_size` | `uint32` |  | `0` |
| `role` | `uint32` |  | `0` |

**Query 示例**：
```json
page=0&page_size=0&role=0
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `users` | `User[]` |  | [] |
| `total` | `uint32` |  | `0` |

**Response 示例**：
```json
{"code": 0, "message": "success", "users": [], "total": 0}
```

### curl 示例
```bash
curl -X GET 'http://localhost:8081/api/v1/user/get_users?page=0&page_size=0&role=0'
```

## ChangePassword

- **URL**: `http://localhost:8081/api/v1/user/change_password`
- **Method**: `PUT`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `user_id` | `uint32` |  | `0` |
| `old_password` | `string` |  | `""` |
| `new_password` | `string` |  | `""` |

**Body 示例**：
```json
{"user_id": 0, "old_password": "", "new_password": ""}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |

**Response 示例**：
```json
{"code": 0, "message": "success"}
```

### curl 示例
```bash
curl -X PUT 'http://localhost:8081/api/v1/user/change_password' \
  -H 'Content-Type: application/json' \
  -d '{"user_id": 0, "old_password": "", "new_password": ""}'
```

## AddToBlacklist

- **URL**: `http://localhost:8081/api/v1/user/add_to_blacklist`
- **Method**: `POST`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `user_id` | `uint32` |  | `0` |
| `target_user_id` | `uint32` |  | `0` |
| `reason` | `string` |  | `""` |

**Body 示例**：
```json
{"user_id": 0, "target_user_id": 0, "reason": ""}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |

**Response 示例**：
```json
{"code": 0, "message": "success"}
```

### curl 示例
```bash
curl -X POST 'http://localhost:8081/api/v1/user/add_to_blacklist' \
  -H 'Content-Type: application/json' \
  -d '{"user_id": 0, "target_user_id": 0, "reason": ""}'
```

## RemoveFromBlacklist

- **URL**: `http://localhost:8081/api/v1/user/remove_from_blacklist`
- **Method**: `DELETE`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `user_id` | `uint32` |  | `0` |
| `target_user_id` | `uint32` |  | `0` |
| `reason` | `string` |  | `""` |

**Body 示例**：
```json
{"user_id": 0, "target_user_id": 0, "reason": ""}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |

**Response 示例**：
```json
{"code": 0, "message": "success"}
```

### curl 示例
```bash
curl -X DELETE 'http://localhost:8081/api/v1/user/remove_from_blacklist' \
  -H 'Content-Type: application/json' \
  -d '{"user_id": 0, "target_user_id": 0, "reason": ""}'
```

## IsInBlacklist

- **URL**: `http://localhost:8081/api/v1/user/is_in_blacklist?user_id=0&target_user_id=0`
- **Method**: `GET`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Query String

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `user_id` | `uint32` |  | `0` |
| `target_user_id` | `uint32` |  | `0` |

**Query 示例**：
```json
user_id=0&target_user_id=0
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `in_blacklist` | `bool` |  | `false` |

**Response 示例**：
```json
{"code": 0, "message": "success", "in_blacklist": false}
```

### curl 示例
```bash
curl -X GET 'http://localhost:8081/api/v1/user/is_in_blacklist?user_id=0&target_user_id=0'
```

## AdminGetUsers

- **URL**: `http://localhost:8081/api/v1/user/admin_get_users`
- **Method**: `POST`
- **鉴权**: 管理员（需 JWT + 管理员角色）

### Headers
```http
Authorization: Bearer <token>
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `page` | `uint32` |  | `0` |
| `page_size` | `uint32` |  | `0` |
| `role` | `uint32` |  | `0` |
| `keyword` | `string` | 用户名/邮箱模糊匹配 | `""` |
| `status` | `uint32` | 0=全部，1=正常，2=禁用 | `0` |

**Body 示例**：
```json
{"page": 0, "page_size": 0, "role": 0, "keyword": "", "status": 0}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `users` | `User[]` |  | [] |
| `total` | `uint32` |  | `0` |

**Response 示例**：
```json
{"code": 0, "message": "success", "users": [], "total": 0}
```

### curl 示例
```bash
curl -X POST 'http://localhost:8081/api/v1/user/admin_get_users' \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"page": 0, "page_size": 0, "role": 0, "keyword": "", "status": 0}'
```

## AdminUpdateUser

- **URL**: `http://localhost:8081/api/v1/user/admin_update_user`
- **Method**: `PUT`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `id` | `uint32` |  | `0` |
| `nickname` | `string` | 空字符串=不修改 | `""` |
| `role` | `uint32` | 0=不修改 | `0` |
| `status` | `uint32` | 0=不修改 | `0` |

**Body 示例**：
```json
{"id": 0, "nickname": "", "role": 0, "status": 0}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `user` | `User` |  | 见 [User](#user) |

**Response 示例**：
```json
{"code": 0, "message": "success", "user": {}}
```

### curl 示例
```bash
curl -X PUT 'http://localhost:8081/api/v1/user/admin_update_user' \
  -H 'Content-Type: application/json' \
  -d '{"id": 0, "nickname": "", "role": 0, "status": 0}'
```

## AdminDeleteUser

- **URL**: `http://localhost:8081/api/v1/user/admin_delete_user`
- **Method**: `POST`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `id` | `uint32` |  | `0` |

**Body 示例**：
```json
{"id": 0}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |

**Response 示例**：
```json
{"code": 0, "message": "success"}
```

### curl 示例
```bash
curl -X POST 'http://localhost:8081/api/v1/user/admin_delete_user' \
  -H 'Content-Type: application/json' \
  -d '{"id": 0}'
```

## ListOperationLogs

- **URL**: `http://localhost:8081/api/v1/user/list_operation_logs?page=0&page_size=0&action=<action>&target_type=<target_type>&operator_id=0`
- **Method**: `GET`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Query String

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `page` | `uint32` |  | `0` |
| `page_size` | `uint32` |  | `0` |
| `action` | `string` |  | `""` |
| `target_type` | `string` |  | `""` |
| `operator_id` | `uint32` |  | `0` |

**Query 示例**：
```json
page=0&page_size=0&action=<action>&target_type=<target_type>&operator_id=0
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `logs` | `OperationLog[]` |  | [] |
| `total` | `uint32` |  | `0` |

**Response 示例**：
```json
{"code": 0, "message": "success", "logs": [], "total": 0}
```

### curl 示例
```bash
curl -X GET 'http://localhost:8081/api/v1/user/list_operation_logs?page=0&page_size=0&action=<action>&target_type=<target_type>&operator_id=0'
```

## RecordLog

- **URL**: `http://localhost:8081/api/v1/user/record_log`
- **Method**: `POST`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `operator_id` | `uint32` |  | `0` |
| `operator` | `string` |  | `""` |
| `action` | `AuditAction` |  | `AUDIT_ACTION_UPDATE_USER` |
| `target_type` | `string` |  | `""` |
| `target_id` | `uint32` |  | `0` |
| `target_title` | `string` |  | `""` |
| `detail` | `string` |  | `""` |
| `ip` | `string` |  | `""` |

**Body 示例**：
```json
{"operator_id": 0, "operator": "", "action": {}, "target_type": "", "target_id": 0, "target_title": "", "detail": "", "ip": ""}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `id` | `uint32` |  | `0` |

**Response 示例**：
```json
{"code": 0, "message": "success", "id": 0}
```

### curl 示例
```bash
curl -X POST 'http://localhost:8081/api/v1/user/record_log' \
  -H 'Content-Type: application/json' \
  -d '{"operator_id": 0, "operator": "", "action": {}, "target_type": "", "target_id": 0, "target_title": "", "detail": "", "ip": ""}'
```

## Follow

- **URL**: `http://localhost:8081/api/v1/user/follow`
- **Method**: `POST`
- **鉴权**: 登录（需 JWT）

### Headers
```http
Authorization: Bearer <token>
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `follower_id` | `uint32` | 关注者（当前登录用户） | `0` |
| `following_id` | `uint32` | 被关注者 | `0` |

**Body 示例**：
```json
{"follower_id": 0, "following_id": 0}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `follower_count` | `uint32` | 关注者当前关注数 | `0` |
| `following_count` | `uint32` | 被关注者当前粉丝数 | `0` |

**Response 示例**：
```json
{"code": 0, "message": "success", "follower_count": 0, "following_count": 0}
```

### curl 示例
```bash
curl -X POST 'http://localhost:8081/api/v1/user/follow' \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"follower_id": 0, "following_id": 0}'
```

## Unfollow

- **URL**: `http://localhost:8081/api/v1/user/unfollow`
- **Method**: `POST`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `follower_id` | `uint32` |  | `0` |
| `following_id` | `uint32` |  | `0` |

**Body 示例**：
```json
{"follower_id": 0, "following_id": 0}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `follower_count` | `uint32` |  | `0` |
| `following_count` | `uint32` |  | `0` |

**Response 示例**：
```json
{"code": 0, "message": "success", "follower_count": 0, "following_count": 0}
```

### curl 示例
```bash
curl -X POST 'http://localhost:8081/api/v1/user/unfollow' \
  -H 'Content-Type: application/json' \
  -d '{"follower_id": 0, "following_id": 0}'
```

## GetFollowStats

- **URL**: `http://localhost:8081/api/v1/user/get_follow_stats?user_id=0`
- **Method**: `GET`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Query String

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `user_id` | `uint32` | 查询对象 | `0` |

**Query 示例**：
```json
user_id=0
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `follower_count` | `uint32` | 粉丝数（被人关注） | `0` |
| `following_count` | `uint32` | 关注数（关注别人） | `0` |

**Response 示例**：
```json
{"code": 0, "message": "success", "follower_count": 0, "following_count": 0}
```

### curl 示例
```bash
curl -X GET 'http://localhost:8081/api/v1/user/get_follow_stats?user_id=0'
```

## GetFollowStatus

- **URL**: `http://localhost:8081/api/v1/user/get_follow_status?follower_id=0&following_id=0`
- **Method**: `GET`
- **鉴权**: 登录（需 JWT）

### Headers
```http
Authorization: Bearer <token>
Content-Type: application/json
```

### Request
**参数位置**：Query String

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `follower_id` | `uint32` |  | `0` |
| `following_id` | `uint32` |  | `0` |

**Query 示例**：
```json
follower_id=0&following_id=0
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `is_following` | `bool` |  | `false` |

**Response 示例**：
```json
{"code": 0, "message": "success", "is_following": false}
```

### curl 示例
```bash
curl -X GET 'http://localhost:8081/api/v1/user/get_follow_status?follower_id=0&following_id=0' \
  -H 'Authorization: Bearer <token>'
```

---

## 数据结构

> 下列 message / enum 被上述接口的请求或响应引用；结构体字段中的 message 类型可点击跳转到对应定义。

### User

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `id` | `uint32` |  | `0` |
| `username` | `string` |  | `""` |
| `email` | `string` |  | `""` |
| `password` | `string` |  | `""` |
| `nickname` | `string` |  | `""` |
| `avatar` | `string` |  | `""` |
| `bio` | `string` |  | `""` |
| `role` | `uint32` |  | `0` |
| `status` | `uint32` |  | `0` |
| `created_at` | `string` |  | `""` |
| `updated_at` | `string` |  | `""` |

### OperationLog

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `id` | `uint32` |  | `0` |
| `operator_id` | `uint32` |  | `0` |
| `operator` | `string` |  | `""` |
| `action` | `string` |  | `""` |
| `target_type` | `string` |  | `""` |
| `target_id` | `uint32` |  | `0` |
| `target_title` | `string` |  | `""` |
| `detail` | `string` |  | `""` |
| `ip` | `string` |  | `""` |
| `created_at` | `string` |  | `""` |

### AuditAction (enum)

| 值 | 编号 | 说明 |
| --- | --- | --- |
| `AUDIT_ACTION_UNSPECIFIED` | `0` |  |
| `AUDIT_ACTION_UPDATE_USER` | `1` | 更新用户资料 |
| `AUDIT_ACTION_DELETE_USER` | `2` | 删除用户 |
| `AUDIT_ACTION_DISABLE_USER` | `3` | 禁用用户 |
| `AUDIT_ACTION_ENABLE_USER` | `4` | 启用用户 |
| `AUDIT_ACTION_SET_ROLE` | `5` | 修改用户角色 |
| `AUDIT_ACTION_COMMENT_CREATE` | `10` | 创建评论 |
| `AUDIT_ACTION_COMMENT_REPLY` | `11` | 回复评论 |
| `AUDIT_ACTION_COMMENT_UPDATE` | `12` | 更新评论 |
| `AUDIT_ACTION_COMMENT_DELETE` | `13` | 删除评论 |
| `AUDIT_ACTION_COMMENT_LIKE` | `14` | 点赞评论 |
| `AUDIT_ACTION_COMMENT_ENABLE` | `15` | 开启文章评论 |
| `AUDIT_ACTION_COMMENT_DISABLE` | `16` | 关闭文章评论 |
| `AUDIT_ACTION_ARTICLE_CREATE` | `20` | 创建文章 |
| `AUDIT_ACTION_ARTICLE_UPDATE` | `21` | 更新文章 |
| `AUDIT_ACTION_ARTICLE_DELETE` | `22` | 删除文章 |
| `AUDIT_ACTION_ARTICLE_PUBLISH` | `23` | 发布文章 |
| `AUDIT_ACTION_ARTICLE_OFFLINE` | `24` | 下架文章 |
| `AUDIT_ACTION_ARTICLE_APPROVE` | `25` | 审核通过 |
| `AUDIT_ACTION_ARTICLE_REJECT` | `26` | 审核拒绝 |
| `AUDIT_ACTION_ARTICLE_SUBMIT` | `27` | 提交审核 |
| `AUDIT_ACTION_ARTICLE_SET_CATEGORY` | `28` | 设置文章分类 |
| `AUDIT_ACTION_CATEGORY_CREATE` | `30` | 创建分类 |
| `AUDIT_ACTION_CATEGORY_UPDATE` | `31` | 更新分类 |
| `AUDIT_ACTION_CATEGORY_DELETE` | `32` | 删除分类 |

