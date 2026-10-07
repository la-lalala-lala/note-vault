# NoteVault 墨匣

## 1. 项目定位

NoteVault 是一个基于 Go、Gin、GORM、PostgreSQL 和 Redis 的个人知识管理后端，目标是提供用户管理、笔记簿、笔记、收藏夹、草稿、回收站、文件、操作日志、菜单权限和登录鉴权能力。

当前仓库是项目骨架，已经包含用户登录和用户 CRUD 的示例实现；本文件是后续完整实现的需求梳理和技术设计，目标数据库结构见 [`schema.sql`](schema.sql)。

## 2. 需求梳理

### 2.1 用户管理

- 用户注册、登录、退出登录。
- 用户信息新增、修改、查询、禁用和删除。
- 用户密码必须使用密码哈希保存，不能保存明文密码。
- 用户可以绑定多个角色。
- 用户状态包括正常、锁定和注销。

### 2.2 笔记簿管理

- 用户可以创建、修改、查询和删除自己的笔记簿。
- 笔记簿属于用户，用户之间不能互相访问。
- 笔记簿有名称、封面图标名、标签、描述、状态和排序字段。
- 默认不允许删除仍包含笔记的笔记簿，避免误删数据。

### 2.3 笔记管理

- 用户可以在指定笔记簿下新增、修改、查询和删除笔记。
- 笔记包含标题、标签、摘要、封面、正文和状态。
- 所有业务查询必须附加当前用户的 `owner_id` 条件。
- 正文既支持直接保存，也预留 `content_uri` 用于本地文件或对象存储。
- 使用 `version` 实现乐观锁，避免多个页面同时编辑时互相覆盖。

### 2.4 收藏夹管理

- 用户可以新增、修改、查询和删除收藏链接。
- 收藏链接属于用户。
- 同一用户的同一 URL 不允许重复收藏。

### 2.5 草稿管理

- 前端编辑笔记时定时自动保存草稿。
- 草稿可以对应已有笔记，也可以对应尚未正式创建的笔记。
- 自动保存必须幂等，网络重试不能产生重复草稿。
- 正式保存笔记成功后，在同一个事务中删除对应草稿。
- 草稿和正式笔记分离，草稿不进入回收站。

### 2.6 废纸篓管理

- 删除笔记时不直接物理删除，而是将笔记标记为回收站状态。
- 默认保留 30 天。
- 用户可以手动恢复或永久删除。
- 后台任务定期清理超过保留时间的笔记和关联文件。
- 回收站仍然按照 `owner_id` 隔离，用户不能查看其他用户的回收站。

### 2.7 日志管理

- 对新增、修改、删除、恢复、登录、退出登录、密码修改和权限变更进行审计。
- 审计日志只记录谁在什么时间、什么地点执行了什么操作。
- 操作地点由请求 IP 和可选的地点描述组成。
- Zap 用于系统运行日志，PostgreSQL 用于可查询的业务审计日志。

### 2.8 菜单与权限

- 支持目录、页面、按钮和 API 权限。
- 支持菜单树的新增、修改、查询和删除。
- 支持角色新增、修改、查询和删除。
- 支持给角色绑定菜单和按钮权限。
- 支持给用户绑定多个角色。
- 后端接口通过 `http_method + api_path` 进行访问控制；页面和按钮仍可使用菜单编码控制前端展示。

### 2.9 鉴权授权

- 登录成功后生成不透明随机 Token，不使用 JWT。
- Token 使用 `crypto/rand` 生成至少 32 字节随机数据，并使用 Base64URL 编码后返回客户端。
- Token 本身不包含用户 ID、账号、角色、权限和过期时间，客户端无法从 Token 中解析用户信息。
- 服务端对 Token 计算 SHA-256，仅使用 Token 哈希作为 Redis Key，避免在 Redis Key 中直接保存原始 Token。
- Redis 会话只保存用户 ID、账号、可访问接口列表和登录 IP，会话过期由 Redis TTL 控制。
- 默认校验请求 IP 与登录 IP 是否一致，可通过配置关闭或调整。
- 用户退出登录时删除 Redis 会话，使当前 Token 立即失效。
- 自增 ID、时间戳和雪花 ID 可以作为内部会话编号，但不直接作为登录 Token，因为这类 ID 存在规律且容易被推测。

## 3. 当前工程分析

Go 和 gofmt安装位置： /data/program/go1.27.1.linux-amd64/bin

当前已有代码结构如下：

```text
app/
├── config/       配置和数据库连接
├── controller/   HTTP 控制器
├── dto/          请求和响应数据传输对象
├── generated/    GORM 生成查询对象
├── middleware/   服务端会话认证中间件
├── model/        GORM 模型
├── repository/   数据访问层
├── router/       路由注册
├── scheduler/    定时任务
├── service/      业务服务
├── storage/      文件存储
└── utils/        Token、Redis 和通用工具
main.go       程序入口
sql.sql       历史数据库导出脚本
source.md     用户模块示例代码说明
```

前端构建产物放在项目根目录 `dist/`。后端启动后会同时托管该目录：静态资源通过根路径访问，React 前端路由会自动回退到 `dist/index.html`；构建部署时 `build.sh` 会将 `dist/` 复制到 `build/dist/`。

当前仍存在以下需要在后续阶段解决的问题：

1. 当前随机 Token 会话已经以 Redis 为认证来源，但仍需补充用户状态校验和最大会话数控制。
2. 当前认证中间件仍需增加用户被禁用后的会话批量失效处理。
3. `sql.sql` 中业务表用 `source` 保存作者账号，目标结构统一改为 `owner_id` 外键。
4. 当前 `note.status` 不足以完整支持 30 天回收站，需要增加删除时间、清理时间和删除人。
5. 角色、菜单、用户角色和角色菜单表已经定义，但对应管理接口尚未实现。
6. 审计日志上下文和审计查询接口尚未实现。
7. `config.yaml` 中存在数据库和 Redis 敏感配置，正式环境必须通过环境变量注入。

当前已经完成 User Model 与 `schema.sql` 对齐、请求/响应 DTO 分离、bcrypt 密码哈希校验、统一响应与分页、Request ID、路由分组和接口权限匹配。

`sql.sql` 是历史数据导出脚本，不作为新功能的目标结构。新结构使用 `note_vault` schema，迁移时需要将历史表数据映射到新表。

## 4. 总体架构

项目采用模块化单体架构，先保持一个服务进程，按业务模块划分代码边界。

```text
HTTP Request
    ↓
Gin Router
    ↓
RequestID / Recovery / CORS
    ↓
Opaque Token Authentication
    ↓
Redis Session Check
    ↓
Permission Check
    ↓
Controller
    ↓
Application Service
    ↓
Repository
    ↓
PostgreSQL / Redis / File Storage
```

推荐目录结构：

```text
main.go
app/
├── config/
├── controller/
├── dto/
├── generated/
├── middleware/
├── model/
├── repository/
├── router/
├── scheduler/
├── service/
├── storage/
└── utils/
migrations/
```

调用约束：

- Controller 只负责参数绑定、权限上下文读取和响应转换。
- Service 负责业务规则、事务和资源归属校验。
- Repository 负责数据库读写。
- Model 只表达数据结构和持久化关系。
- 业务代码禁止直接使用前端传入的用户 ID 判断数据权限。

## 5. 数据模型设计

目标表结构包含以下核心实体：

```text
users ──< user_roles >── roles ──< role_menus >── menus
  │
  ├──< notebooks ──< notes ──< drafts
  │                    │
  │                    └── 回收站状态由 notes.status 表示
  │
  ├──< bookmarks
  ├──< files
  └──< audit_logs
```

### 5.1 用户和权限

- `users`：用户基本信息、密码哈希和状态。
- `roles`：角色定义。
- `menus`：目录、页面、按钮和 API 权限资源。
- `user_roles`：用户与角色多对多关系。
- `role_menus`：角色与权限资源多对多关系。

### 5.2 笔记业务

- `notebooks`：用户的笔记簿。
- `notes`：笔记元数据和正文。
- `drafts`：自动保存的草稿。
- `bookmarks`：收藏链接。
- `files`：上传文件和附件元数据。

### 5.3 日志和回收站

- `notes.status = 2` 表示笔记进入回收站。
- `notes.deleted_at` 表示删除时间。
- `notes.purge_at` 表示允许自动清理的时间。
- `audit_logs` 保存操作人、操作时间、IP、地点和操作内容。

## 6. 状态约定

### 用户状态

```text
1 NORMAL   正常
2 LOCKED   锁定
3 CLOSED   注销
```

### 笔记簿状态

```text
1 NORMAL   正常
2 DISABLED 禁用
3 DELETED  删除
```

### 笔记状态

```text
1 NORMAL   正常
2 TRASH    回收站
```

### 菜单类型

```text
1 DIRECTORY 目录
2 PAGE      页面
3 BUTTON    按钮
4 API       接口权限
```

## 7. API 设计

统一前缀：`/api/v1`。

统一响应格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "requestId": "请求ID"
}
```

所有 JSON 响应均返回 `code`、`message`、`data` 和 `requestId`；响应头同时返回 `X-Request-ID`。当前公共业务码如下：

| 业务码 | 含义 | 常用 HTTP 状态码 |
| --- | --- | --- |
| `0` | 成功 | `200` |
| `40001` | 参数错误 | `400` |
| `40101` | 未登录或会话无效 | `401` |
| `40301` | 无权限 | `403` |
| `40401` | 资源不存在 | `404` |
| `40901` | 数据冲突 | `409` |
| `40902` | 笔记版本冲突 | `409` |
| `41301` | 请求体过大 | `413` |
| `50001` | 系统错误 | `500` |
| `50301` | 依赖服务不可用 | `503` |

### 7.1 认证接口

```text
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/logout
POST /api/v1/auth/refresh
PUT  /api/v1/auth/password
```

### 7.2 用户和授权接口

```text
GET  /api/v1/users
POST /api/v1/users
GET  /api/v1/users/profile
PUT  /api/v1/users/profile
GET  /api/v1/users/:id
PUT  /api/v1/users/:id
DELETE /api/v1/users/:id
PUT  /api/v1/users/:id/status
PUT  /api/v1/users/:id/roles

GET  /api/v1/roles
POST /api/v1/roles
PUT  /api/v1/roles/:id
DELETE /api/v1/roles/:id
PUT  /api/v1/roles/:id/menus

GET  /api/v1/menus/tree
POST /api/v1/menus
PUT  /api/v1/menus/:id
DELETE /api/v1/menus/:id
```

### 7.3 笔记业务接口

```text
GET  /api/v1/notebooks
POST /api/v1/notebooks
GET  /api/v1/notebooks/:id
PUT  /api/v1/notebooks/:id
PUT  /api/v1/notebooks/:id/sort
PUT  /api/v1/notebooks/:id/status
DELETE /api/v1/notebooks/:id

GET  /api/v1/notes
POST /api/v1/notes
GET  /api/v1/notes/:id
PUT  /api/v1/notes/:id
DELETE /api/v1/notes/:id
PUT  /api/v1/notes/:id/restore
DELETE /api/v1/notes/:id/permanent
PUT  /api/v1/notes/:id/move

GET  /api/v1/drafts
GET  /api/v1/drafts/:id
PUT  /api/v1/drafts/:draftKey
POST /api/v1/drafts/:id/submit
DELETE /api/v1/drafts/:id

GET  /api/v1/trash/notes
PUT  /api/v1/trash/notes/:id/restore
DELETE /api/v1/trash/notes/:id

GET  /api/v1/bookmarks
POST /api/v1/bookmarks
GET  /api/v1/bookmarks/:id
PUT  /api/v1/bookmarks/:id
DELETE /api/v1/bookmarks/:id

POST /api/v1/files
POST /api/v1/files/base64
GET  /api/v1/files
GET  /api/v1/files/:id
DELETE /api/v1/files/:id
```

### 7.4 分页约定

所有列表接口统一支持：

```text
page=1
pageSize=20
keyword=
```

用户列表 `GET /api/v1/users` 在通用分页参数之外支持以下筛选条件：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `keyword` | string | 模糊匹配账号、姓名、邮箱和手机号 |
| `account` | string | 按账号模糊匹配 |
| `name` | string | 按姓名模糊匹配 |
| `email` | string | 按邮箱模糊匹配 |
| `phone` | string | 按手机号模糊匹配 |
| `status` | integer | 用户状态：`1` 正常、`2` 锁定、`3` 注销 |

示例：

```text
GET /api/v1/users?keyword=张&page=1&pageSize=20
GET /api/v1/users?account=admin&status=1&page=1&pageSize=20
```

返回：

```json
{
  "list": [],
  "page": 1,
  "pageSize": 20,
  "total": 0
}
```

## 8. 关键业务流程

### 8.1 登录

```text
校验账号和密码
    ↓
校验用户状态
    ↓
生成 32 字节密码学安全随机 Token
    ↓
计算 tokenHash = SHA-256(Token)
    ↓
Redis 保存 auth:session:{tokenHash}
    ↓
记录登录审计日志
    ↓
只向客户端返回一次原始 Token 和用户基本信息
```

Redis 会话不保存密码，建议结构如下：

```json
{
  "userId": 1,
  "account": "demo",
  "apiPermissions": [
    "GET:/api/v1/notes",
    "POST:/api/v1/notes",
    "PUT:/api/v1/notes/:id"
  ],
  "loginIp": "127.0.0.1"
}
```

### 8.2 自动保存草稿

```text
页面编辑
    ↓
PUT /drafts/:id
    ↓
按 owner_id + draft_key Upsert
    ↓
返回草稿版本和更新时间
```

### 8.3 正式保存笔记

```text
开启事务
    ├── 校验笔记簿归属
    ├── 创建或更新 notes
    ├── 删除对应 drafts
    └── 写入 audit_logs
提交事务
```

### 8.4 删除和恢复笔记

删除时更新：

```text
status = 2
deleted_at = 当前时间
deleted_by = 当前用户
purge_at = 当前时间 + 30 天
```

恢复时将上述状态和时间字段恢复为空。自动清理任务按 `purge_at` 分批处理，避免一次删除大量数据阻塞业务请求。

## 9. 权限校验流程

API 类型菜单直接保存 HTTP 方法和 Gin 路由模板，例如：

```text
GET    + /api/v1/notes
POST   + /api/v1/notes
PUT    + /api/v1/notes/:id
DELETE + /api/v1/notes/:id
```

会话中保存组合后的接口权限键，例如 `GET:/api/v1/notes`。动态路径必须保存 Gin 路由模板 `/api/v1/notes/:id`，不能保存某次请求的实际路径 `/api/v1/notes/100`。

中间件按以下顺序处理：

1. 从 `Authorization: <token>` 获取不透明 Token，请求头内容直接是 Token，不使用 Bearer 前缀。
2. 计算 Token 的 SHA-256 哈希。
3. 根据 `auth:session:{tokenHash}` 查询 Redis 会话。
4. 校验会话是否存在以及是否过期。
5. 校验用户状态。
6. 校验登录 IP 和当前请求 IP。
7. 使用请求方法和 `c.FullPath()` 生成接口权限键。
8. 判断 Redis 会话的 `apiPermissions` 是否包含该接口权限键。
9. 将 `userId`、`account` 和接口权限列表写入 Gin Context。

超级管理员登录时可以在 `apiPermissions` 中写入通配符 `*`。角色或 API 权限发生变化时，删除对应用户的全部登录会话，要求重新登录并生成新的接口权限快照。

## 10. SQL 使用说明

`schema.sql` 是目标数据库结构，执行前建议创建独立数据库，然后执行：

```bash
psql "$DATABASE_URL" -f schema.sql
```

历史 `sql.sql` 中已有数据时，不建议直接覆盖生产库。推荐使用迁移脚本完成：

1. 创建 `note_vault` schema。
2. 导入用户并建立账号到用户 ID 的映射。
3. 将各业务表的 `source` 转换为 `owner_id`。
4. 将旧笔记状态转换为新的 `notes.status`。
5. 将旧的 `path` 映射到 `content_uri`。
6. 校验用户、笔记簿、笔记和文件的归属关系。

## 11. 配置建议

当前服务从 `config.yaml` 加载配置。生产环境应通过安全的配置发布方式注入数据库和 Redis 密码，不应将真实密钥提交到仓库。

```yaml
server:
  host: 0.0.0.0
  port: 8080
  readTimeout: 15s
  writeTimeout: 15s
  shutdownTimeout: 10s
  healthCheckTimeout: 3s
  maxBodyBytes: 16777216

cors:
  allowedOrigins:
    - "*"
  allowedMethods:
    - GET
    - POST
    - PUT
    - PATCH
    - DELETE
    - OPTIONS
  allowedHeaders:
    - Authorization
    - Content-Type
    - X-Request-ID
  exposedHeaders:
    - X-Request-ID
  allowCredentials: false
  maxAge: 12h

database:
  dsn: <POSTGRESQL_DSN>
  pool:
    maxIdleConns: 10
    maxOpenConns: 100
    connMaxLifetime: 1h
    connMaxIdleTime: 30m

redis:
  addr: <REDIS_HOST:PORT>
  password: <REDIS_PASSWORD>
  db: 0
  poolSize: 10
  minIdleConns: 5
  maxRetries: 3
  operationTimeout: 3s

session:
  expireHours: 24
  tokenBytes: 32
  bindIP: true

auth:
  trustedProxies: []
  maxSessions: 5

storage:
  type: local
  basePath: ./data
  maxFileSize: 10485760
  allowedMimeTypes:
    - image/jpeg
    - image/png
    - image/gif
    - image/webp
    - application/pdf
    - text/plain
    - text/markdown
  allowedExtensions:
    - .jpg
    - .jpeg
    - .png
    - .gif
    - .webp
    - .pdf
    - .txt
    - .md

trash:
  retentionDays: 30
  cleanupInterval: 1h

log:
  level: info
  format: console
  output: stdout
```

## 12. 实施顺序

项目按照“工程基础 → 身份权限 → 笔记主链路 → 数据生命周期 → 交付保障”的顺序实施。每个阶段均包含明确的前置条件、开发任务、交付物和验收条件；当前阶段未通过验收前，不进入下一阶段的大规模开发。

### 第一阶段：工程基础设施

#### 阶段目标

建立后续模块共同依赖的配置、数据库、Redis、日志、HTTP 和安全基础，消除当前代码中的全局依赖、明文密码和不统一响应问题。

#### 1.1 配置与启动流程

1. 扩展配置结构，覆盖 Server、PostgreSQL、Redis、Session、鉴权、文件存储、回收站和日志配置。
2. 启动时检查数据库 DSN、Redis 地址和会话有效期等必填项。
3. 监听退出信号，优雅关闭 HTTP Server、数据库连接池和 Redis 客户端。

#### 1.2 Redis 基础能力

1. 独立初始化 Redis，不与数据库初始化过程耦合。
2. 统一 Redis Key 命名，会话使用 `auth:session:{tokenHash}`；接口权限直接保存在会话的 `APIPermissions` 中，不再单独维护权限缓存。
3. 封装 JSON 读写、TTL、删除和批量失效操作。
4. 为 Redis 操作设置超时，禁止使用无限期后台 Context。
5. 明确 Redis 不可用时的处理策略：认证会话依赖 Redis，因此受保护接口应拒绝访问而不是绕过校验。

#### 1.3 HTTP 公共组件

1. 定义统一成功响应、错误响应和分页响应。
2. 定义参数错误、未登录、无权限、资源不存在、数据冲突和系统错误等业务错误码。
3. 增加 Request ID 中间件，并写入响应头和系统日志。
4. 增加 Recovery 中间件，捕获 Panic 并返回统一错误。
5. 增加参数校验器和统一校验错误转换。
6. 配置 CORS、可信代理、安全响应头和请求体大小限制。
7. 将业务路由统一调整为 `/api/v1` 前缀。
8. 增加 `/health/live` 和 `/health/ready` 健康检查接口。

#### 1.4 密码和随机 Token 基础组件

1. 使用 Argon2id 或 bcrypt 实现密码哈希和密码校验。
2. `users.password` 只保存哈希值，禁止保存或记录明文密码。
3. 请求 DTO、响应 DTO 和数据库 Model 分离，任何响应都不能包含密码。
4. 使用 `crypto/rand` 生成至少 32 字节随机 Token，并使用 Base64URL 编码。
5. 提供 Token SHA-256 哈希函数，Redis 只使用哈希值定位会话。
6. 随机 Token 不携带任何业务信息，用户身份和权限只能从 Redis 会话中读取。

#### 阶段交付物

- 可启动、健康检查和优雅关闭的服务骨架。
- PostgreSQL、Redis、Zap、统一响应和错误处理组件。
- 密码哈希、随机 Token、分页和事务基础组件。

#### 阶段验收

- 缺少必要配置时服务启动失败并给出明确原因。
- 数据库或 Redis 不可用时就绪检查失败。
- Panic 不会导致服务进程退出。
- 所有 API 响应格式一致并包含 Request ID。
- 空数据库能够通过迁移创建完整结构。
- 密码哈希、随机 Token 和 Token 哈希组件具有单元测试。

### 第二阶段：用户、会话和 RBAC 权限

#### 阶段目标

完成从用户登录到接口授权的完整安全链路，使后续业务模块能够获得可信的当前用户、在线会话和权限信息。

#### 2.1 用户管理

1. 按 `schema.sql` 实现 User 模型、Repository、Service、Controller 和 DTO。
2. 实现用户注册、管理员新增、分页查询、详情查询和资料修改。
3. 账号按不区分大小写规则校验唯一性，并与数据库唯一索引保持一致。
4. 实现正常、锁定和注销状态变更。
5. 普通用户只能修改自己的资料；管理接口使用独立的 API 权限记录。


#### 2.2 登录、会话和退出

1. 登录时先按账号查询用户，再使用密码哈希函数校验密码。
2. 登录前校验用户状态，锁定或注销用户不能登录。
3. 登录成功后使用 `crypto/rand` 生成至少 32 字节随机 Token。
4. 对原始 Token 计算 SHA-256，并以 `auth:session:{tokenHash}` 作为 Redis Key。
5. Redis 会话保存用户 ID、账号、可访问接口列表和登录 IP。
6. Redis TTL 是会话过期的唯一依据，并与配置的会话有效期保持一致。
7. 更新用户最近登录时间和最近登录 IP。
8. 只向客户端返回原始 Token，服务端日志和 Redis Key 不记录原始 Token。
9. 退出登录时计算当前 Token 哈希并删除对应 Redis 会话。
10. 实现当前用户和修改密码接口；修改密码后使所有旧会话失效。
11. 根据 `maxSessions` 限制单用户同时在线会话数量。
12. 锁定或注销用户时删除该用户所有 Redis 会话。


#### 2.3 认证中间件

1. 从 `Authorization: <token>` 读取 Token，请求头内容直接是 Token。
2. 校验 Token 格式并计算 SHA-256 哈希。
3. 根据 `auth:session:{tokenHash}` 查询 Redis 会话；会话不存在时拒绝访问。
4. 校验 Redis 会话是否过期以及必要字段是否完整。
5. 查询或缓存用户状态，被锁定或注销的用户立即失去访问权限。
6. 根据配置校验当前 IP 与登录 IP 是否一致。
7. 将用户信息和token写入 Gin Context。

#### 2.4 角色和菜单权限

1. 实现角色新增、修改、查询、启用、禁用和删除。
2. 实现目录、页面、按钮和 API 权限资源的树形管理。
3. 校验菜单父子关系，禁止循环引用。
4. 实现用户角色分配，维护 `user_roles`。
5. 实现角色权限分配，维护 `role_menus`。
6. 删除角色前检查用户绑定；删除菜单前检查子节点。
7. 保护 `SUPER_ADMIN` 等内置角色，禁止普通管理员删除。

#### 2.5 接口权限校验和会话失效

1. API 类型菜单必须配置 `http_method` 和 `api_path`，例如 `GET` 和 `/api/v1/notes/:id`。
2. 登录时通过 `roles`、`user_roles`、`role_menus` 和 `menus` 查询用户可访问的 API 类型菜单。
3. 将方法和路由模板组合为 `GET:/api/v1/notes/:id`，写入 Redis 会话的 `apiPermissions`。
4. 认证中间件使用当前请求方法和 `c.FullPath()` 生成待校验的接口权限键。
5. 会话包含相同接口权限键时允许访问，否则返回 403。
6. 超级管理员会话写入通配符 `*`，但仍不能跳过登录和用户状态校验。
7. 用户角色、角色 API 权限、角色状态或菜单状态变化时删除受影响用户的全部会话。
8. 初始化 `SUPER_ADMIN`、`USER` 角色和基础 API 权限资源。
9. 提供创建首个超级管理员的幂等初始化命令。

#### 阶段交付物

- 用户管理、注册、登录、退出、当前用户和修改密码接口。
- 随机 Token 与 Redis 服务端会话认证中间件。
- 角色、菜单、用户角色和角色菜单管理接口。
- 基于 HTTP 方法和路由模板的接口权限拦截及会话失效机制。
- 系统角色、权限和管理员初始化能力。

#### 阶段验收

- 密码错误、用户锁定或用户注销时不能登录。
- 随机 Token 对应的 Redis 会话不存在时不能访问接口。
- 修改密码、锁定或注销用户后，旧 Token 立即失效。
- 开启 IP 绑定后，IP 不一致的请求被拒绝。
- 无权限用户访问接口时返回明确的无权限错误。
- 角色 API 权限变化后旧会话失效，重新登录后使用新的接口权限快照。

### 第三阶段：笔记核心业务

#### 阶段目标

完成笔记簿、笔记和草稿的业务闭环，确保用户数据隔离、事务一致性和并发更新安全。

#### 3.1 数据归属约束

1. 所有业务接口从认证上下文获取当前用户 ID，不接收前端提交的 `owner_id`。
2. 笔记簿、笔记和草稿查询统一附加 `owner_id = current_user_id`。
3. 修改和删除时同时使用资源 ID 与 `owner_id` 查询，防止越权。
4. 不向调用方泄露某个资源是否属于其他用户。
5. 管理员跨用户查询必须使用独立接口和独立 API 权限记录。

#### 3.2 笔记簿

1. 实现新增、分页列表、详情、修改、排序和状态修改。
2. 校验同一用户下笔记簿名不重复。
3. 支持名称、标签和状态查询。
4. 创建或修改时校验字段长度和状态值。
5. 删除前统计笔记簿下仍然存在的全部笔记数量。
6. 第一版采用“存在任何笔记时禁止删除笔记簿”的策略。

#### 3.3 笔记

1. 实现新增、分页列表、详情、修改和移动笔记簿。
2. 创建和移动前校验目标笔记簿存在、状态正常且属于当前用户。
3. 列表支持笔记簿、标签、关键字和时间范围筛选。
4. 普通列表默认只返回 `status = 1` 的笔记。
5. 列表只查询摘要字段，正文仅在详情接口返回。
6. 对标题、标签、摘要和正文设置长度限制。
7. 明确正文存储规则：第一版将 Markdown 正文保存到 `content`，`content_uri` 预留给外部存储。

#### 3.4 乐观锁

1. 修改笔记时要求前端提交当前 `version`。
2. 更新条件包含 `id`、`owner_id` 和 `version`。
3. 更新成功后执行 `version = version + 1`。
4. 更新行数为零时区分资源不存在和版本冲突。
5. 版本冲突返回独立错误码，由前端提示重新加载或合并内容。

#### 3.5 草稿自动保存

1. 打开新建笔记编辑器时由前端生成一个 UUID/ULID 作为稳定的 `draft_key`；同一个编辑器生命周期内始终复用该值，不同新建操作必须生成不同值。
2. 保存和修改统一使用 `PUT /api/v1/drafts/:draftKey`，后端按当前用户和编辑上下文定位草稿，首次保存和后续定时保存不区分接口。
3. 新建笔记按 `owner_id + draft_key` 定位；编辑已有笔记提交 `note_id`，按 `owner_id + note_id` 保证同一用户同一笔记最多一份草稿。
4. 草稿自动保存不要求客户端提交草稿版本号，服务端保存时递增内部修订号并返回草稿 ID、修订号和服务端更新时间。
5. 自动保存采用最后一次写入生效策略；前端通过稳定 `draft_key` 区分不同新建操作，并通过防抖、取消旧请求或请求序号避免同一编辑器的请求乱序。
6. 实现草稿分页、详情和手动删除接口；列表只返回草稿元数据，不返回正文。

#### 3.6 草稿提交

1. 提交请求按草稿 ID 开启事务并锁定最新草稿，不再要求提交草稿版本号。
2. 校验草稿、目标笔记簿和已有笔记都属于当前用户。
3. 新建场景创建正式笔记；编辑场景提交时提交编辑开始时读取的 `noteVersion`，按正式笔记乐观锁更新。
4. 正式笔记保存成功后删除对应草稿。
5. 任一步骤失败时回滚事务并保留草稿。

说明：`drafts.version` 仅是服务端草稿保存修订号，不作为客户端并发控制字段；发布已有笔记的草稿时，客户端仍需提交编辑开始时获得的正式笔记 `noteVersion`，用于正式笔记乐观锁校验。

#### 阶段交付物

- 笔记簿完整接口。
- 笔记新增、查询、修改和移动接口。
- 笔记乐观锁和正文存储规则。
- 草稿自动保存、查询、删除和提交接口。
- 笔记业务 API 权限资源和数据归属校验组件。

#### 阶段验收

- 用户不能读取或修改其他用户的数据。
- 存在笔记时不能删除笔记簿。
- 两个客户端同时编辑时，旧版本提交产生版本冲突。
- 同一 `draft_key` 重复保存不会产生多份草稿。
- 草稿提交失败时草稿仍然存在。
- 正式保存成功后对应草稿被删除。

### 第四阶段：回收站和扩展能力

#### 阶段目标

完成笔记删除后的完整生命周期，并补充收藏夹、文件管理和审计日志能力。

#### 4.1 回收站

1. 普通删除不执行物理删除，而是设置 `status = 2`。
2. 同时写入 `deleted_at`、`deleted_by` 和 `purge_at`。
3. `purge_at` 根据配置计算，默认是删除时间后 30 天。
4. 回收站列表只查询当前用户 `status = 2` 的笔记。
5. 返回删除时间、计划清理时间和剩余保留天数。
6. 删除已经在回收站中的笔记时返回状态冲突。

#### 4.2 恢复和永久删除

1. 恢复前检查原笔记簿是否仍存在且状态正常。
2. 原笔记簿不可用时要求用户选择新的笔记簿。
3. 恢复时清空删除时间、删除人和清理时间，并恢复正常状态。
4. 永久删除只允许作用于回收站笔记。
5、永久删除接口必须幂等，重复调用不能造成系统异常。

#### 4.3 自动清理回收站任务

1. 按 `status = 2 AND purge_at <= now()` 查询待清理笔记。
2. 使用固定批次大小，避免一次处理大量数据。
3. 自动清理使用定时任务。
4. 单条清理失败不能中断整个批次。
5. 记录扫描数、成功数、失败数和执行耗时。

#### 4.4 收藏夹

1. 实现新增、分页、详情、修改和删除。
2. 校验 URL 格式和字段长度。
3. 支持标题、标签、重点收藏状态和时间范围查询。
4. 所有查询和修改强制使用当前用户 `owner_id`。

#### 4.5 文件管理

1. 实现文件上传（multipart/form-data 和 Base64 图片）、列表、详情和删除。
2. 配置允许的 MIME 类型、扩展名和最大文件大小。
3. 使用随机 `object_key`，不能直接使用客户端文件名作为存储路径。
4. 保存文件大小、类型和校验和。
5. 抽象本地存储和对象存储接口（当前只实现本地存储），业务层不依赖具体实现。

#### 4.6 审计日志

1. 对登录、退出、用户状态、角色授权、权限变更、笔记新增、修改、删除、恢复和永久删除记录日志。
2. `operator_account` 保存操作发生时的账号快照。
3. `operated_at` 使用数据库时间生成。
4. `ip` 从可信代理处理后的客户端地址获取。
5. `location` 可保存城市、机房或“局域网”等描述。
6. `operation` 使用可读文本，例如“将笔记《标题》移入回收站”。
7. 提供按账号、时间范围、IP 和操作关键字查询的管理接口。
8. 审计日志只允许查询，不提供普通修改和删除接口。

#### 阶段交付物

- 回收站列表、恢复和永久删除接口。
- 30 天自动清理任务和手动清理入口。
- 收藏夹完整接口。
- 文件上传、鉴权访问和删除能力。
- 审计日志写入和查询接口。

#### 阶段验收

- 普通删除不会物理删除笔记。
- 回收站笔记可以恢复到有效笔记簿。
- 超过保留期的笔记能够被分批清理。
- 多实例清理不会重复处理同一笔记。
- 用户不能访问其他用户的文件和收藏。
- 关键操作能查询到操作人、时间、IP、地点和操作内容。

### 第五阶段：测试、交付和运行保障

#### 阶段目标

建立自动化测试、接口文档、容器化和持续集成流程，使项目能够重复验证、部署、回滚和排障。

#### 5.1 单元测试

1. 测试密码哈希、随机 Token、Token 哈希、Redis Key、分页和错误转换工具。
2. 测试用户、认证、权限、笔记、草稿和回收站 Service。
3. 使用 Repository Mock 覆盖事务、归属校验和错误分支。
4. 测试随机 Token 会话、权限和 Recovery 中间件。
5. 覆盖正常流程、参数错误、无权限、唯一键冲突和版本冲突。

#### 5.2 集成与端到端测试

1. 使用独立 PostgreSQL 和 Redis 测试实例。
2. 测试前执行迁移，测试后回滚事务或清理数据。
3. 验证唯一约束、外键、检查约束和关键索引查询。
4. 验证 Redis 会话删除、用户锁定和接口权限变更后的会话失效。
5. 验证草稿 Upsert、乐观锁和回收站自动清理。
6. 覆盖登录、创建笔记簿、保存草稿、发布、删除、恢复和永久删除完整链路。
7. 使用两个测试用户验证跨用户访问被拒绝。

#### 5.3 API 文档

1. 为全部接口编写 OpenAPI 文档。
2. 文档包含请求参数、响应结构、错误码、接口方法、路由模板和调用示例。
3. 标明公开接口和需要登录或权限的接口。
4. 标明分页、时间格式、状态枚举和版本字段。
5. API 变更时必须同步更新文档。

#### 5.4 容器化与部署

1. 提供多阶段构建 Dockerfile。
2. 提供包含应用、PostgreSQL 和 Redis 的 Docker Compose。
3. 提供不含真实密钥的 `.env.example`。
4. 配置数据卷、健康检查、日志和服务依赖。
5. 编写首次启动、数据库迁移、管理员初始化、备份和恢复说明。
6. 部署前验证迁移，并准备数据库和应用回滚方案。

#### 5.5 持续集成

1. CI 执行格式检查、静态检查、单元测试和集成测试。
2. 执行 `go test -race ./...` 检查并发安全。
3. 验证空数据库能够执行全部迁移。
4. 构建镜像并执行启动和健康检查。
5. 任一检查失败时禁止生成发布制品。

#### 5.6 安全与性能检查

1. 检查日志和响应是否泄露密码、Token 或连接信息。
2. 为登录接口增加频率限制和连续失败控制。
3. 限制文件大小、请求体大小和最大分页数量。
4. 检查所有用户资源操作是否包含 `owner_id` 条件。
5. 使用 `EXPLAIN ANALYZE` 检查笔记、草稿、回收站和审计日志查询。
6. 监控慢请求、数据库连接池耗尽和 Redis 异常。

#### 阶段交付物

- 单元测试、集成测试和端到端测试。
- OpenAPI 文档。
- Dockerfile、Docker Compose 和环境变量示例。
- CI 流程、部署文档、备份恢复文档和发布清单。

#### 阶段验收

- 核心业务流程能够在 CI 中自动执行。
- 空数据库可以通过迁移创建完整结构。
- Docker Compose 可以启动完整开发环境。
- API 文档与实际接口方法和路由模板一致。
- 发布不依赖手工修改数据库或源码配置。
- 问题可以通过 Request ID、系统日志和审计日志定位。

## 13. 验收标准

- 未登录请求不能访问受保护接口。
- Token 格式正确但 Redis 会话不存在时，接口仍然拒绝访问。
- 被锁定用户不能继续使用已有 Token。
- 用户不能读取、修改或删除其他用户的数据。
- 笔记删除后进入回收站，不会立即物理删除。
- 回收站笔记可以恢复和永久删除。
- 超过 30 天的回收站笔记可以被后台任务清理。
- 自动保存草稿重复请求不会产生重复数据。
- 新增、修改、删除、恢复和权限操作都能查到审计日志。
- 角色 API 权限变更后，相关用户的旧会话能够及时失效。
