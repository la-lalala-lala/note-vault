# Note Vault API（前端对接）

本文件记录当前 `app/router/router.go` 注册的接口。下方代码块是 **OpenAPI 3.0.3** 规范；前端可从 ` ```yaml ` 代码块提取 YAML 后导入 OpenAPI 工具。接口变更时须同步更新本文件。当前不提供公开注册接口；`app/controller.Register` 未注册路由。

## 调用约定

- 服务地址默认 `http://localhost:8080`；业务接口前缀 `/api/v1`。除 `/health/live`、`/health/ready` 和 `POST /api/v1/auth/login` 外均需登录，并检查登录时生成的接口权限快照；超级管理员使用通配权限。
- 登录后将 `data.token` **原样**放入 `Authorization` 请求头，**不添加 `Bearer `**。会话由 Redis 保存，会续期；密码修改、角色或菜单权限调整等操作可能使旧会话失效。登录响应中的 `data.menuTree` 用于生成侧边栏及按钮权限；接口仍以服务端鉴权为准。
- JSON 请求设置 `Content-Type: application/json`；文件上传使用 `multipart/form-data`，文件字段名为 `file`。`GET /auth/logout` 无请求体；退出后清除前端 Token。
- 所有业务响应使用 `{ "code": 0, "message": "success", "data": ..., "requestId": "..." }`，HTTP 成功状态为 200；删除及授权成功时 `data` 为 `null`。`/health/ready` 未就绪返回 503 且 `data` 包含组件状态。
- 分页从 1 开始，默认 `page=1&pageSize=20`、最大 `pageSize=100`；列表返回 `data: {list, page, pageSize, total}`。时间使用带时区的 RFC 3339（例如 `2026-09-30T10:30:00+08:00`），查询时间需对 `+` URL 编码或使用 `Z`。路径中的 `:id` 在 OpenAPI 中写为 `{id}`，调用时替换为正整数。
- 常用状态：用户 `1` 正常 / `2` 锁定 / `3` 注销；角色、菜单、笔记簿 `1` 启用 / `2` 禁用；笔记 `1` 正常 / `2` 回收站。菜单类型 `1` 目录 / `2` 页面 / `3` 按钮 / `4` API。响应的可选字段可能省略，不要假设总是 `null`。
- 常见 HTTP 状态与业务码：`400/40001` 参数不合法；`401/40101` 未登录或凭证错误；`403/40301` 无权限；`404/40401` 资源不存在；`409/40901` 冲突；`409/40902` 笔记版本冲突；`413/41301` 请求体过大；`503/50301` 依赖服务不可用；`500/50001` 内部错误。HTTP 405 的业务码也是 `40001`。前端先判断 HTTP 状态及 `code`，错误 `message` 供提示，`requestId` 供排查；不要将错误响应的 `data` 当成成功数据。客户端也可传 `X-Request-ID` 以关联服务日志。
- 写笔记时 `version` 是必填乐观锁版本：创建返回 `version=1`，更新/移动须提交当前版本；收到 `40902` 时重新拉取或提示合并。草稿自动保存用固定 `draftKey` 调用同一个 `PUT /drafts/{draftKey}`，返回 `id`；提交草稿后删除草稿并生成/更新正式笔记，若关联已有笔记，提交时提供 `noteVersion`。草稿响应 `version` 不需要回传。正常笔记删除先进入回收站，永久删除不可恢复。
- 数据归属由登录会话确定；前端不要提交 `ownerId`。文件的大小、允许 MIME/后缀以部署配置为准（当前默认最大 10 MiB）。审计日志只有查询接口，没有普通写入/删除接口。
- 草稿 `PUT /drafts/{draftKey}` 的路径参数是编辑器持久化的键（最长 100 字符）；草稿 `GET/DELETE /drafts/{draftKey}` **同一 OpenAPI 路径**要求路径参数是数字草稿 ID。实际 Gin 权限路由分别使用 `:draftKey` 和 `:id`；二者在 OpenAPI 中不能作为两个仅参数名称不同的同级路径定义。
- 当前实现的 `DELETE /api/v1/users/{id}` **不会根据路径 ID 注销指定用户**，Controller 实际注销的是当前登录用户；前端切勿将它作为管理员注销他人的接口使用。此处仅记录现有行为，不改变后端。

### 调用示例

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' -d '{"account":"alice","password":"YourPassword"}'
curl -sS 'http://localhost:8080/api/v1/notes?page=1&pageSize=20' -H 'Authorization: <登录返回的data.token>'
curl -sS -X PUT http://localhost:8080/api/v1/drafts/editor-abc -H 'Authorization: <token>' -H 'Content-Type: application/json' -d '{"notebookId":1,"title":"新笔记","content":"正文"}'
curl -sS -X POST http://localhost:8080/api/v1/drafts/12/submit -H 'Authorization: <token>' -H 'Content-Type: application/json' -d '{}'
curl -sS -X PUT http://localhost:8080/api/v1/notes/42 -H 'Authorization: <token>' -H 'Content-Type: application/json' -d '{"version":1,"title":"更新后的标题"}'
curl -sS -X POST http://localhost:8080/api/v1/files -H 'Authorization: <token>' -F 'file=@picture.png'
```

正常笔记修改成功的响应形如 `{"code":0,"message":"更新笔记成功","data":{"id":42,"version":2,"notebookId":1,"title":"更新后的标题","contentType":"markdown","status":1,"createdAt":"2026-09-30T10:00:00+08:00","updatedAt":"2026-09-30T10:30:00+08:00"},"requestId":"abc"}`；版本冲突返回 HTTP 409、`code=40902`。

## OpenAPI 3.0.3

```yaml
openapi: 3.0.3
info:
  title: Note Vault API
  version: 1.0.0
  description: 当前已注册接口；成功 HTTP 200，非零业务 code 表示失败。x-permission 表示会话需包含的“HTTP方法:路由模板”权限键。
servers:
  - url: http://localhost:8080
    description: 本地默认服务地址，部署时替换
security:
  - sessionToken: []
tags:
  - {name: Health, description: 公开服务监控}
  - {name: Auth, description: 登录及会话}
  - {name: Users, description: 用户}
  - {name: Roles, description: 角色}
  - {name: Menus, description: 菜单与接口权限}
  - {name: Notebooks, description: 笔记簿}
  - {name: Notes, description: 正式笔记}
  - {name: Trash, description: 笔记回收站}
  - {name: Bookmarks, description: 收藏夹}
  - {name: Drafts, description: 草稿箱}
  - {name: Files, description: 文件}
  - {name: Audit, description: 审计日志，仅有读接口}
paths:
  /health/live:
    get:
      tags: [Health]
      summary: 存活检查
      operationId: live
      security: []
      responses: {'200': {$ref: '#/components/responses/Health'}, default: {$ref: '#/components/responses/Error'}}
  /health/ready:
    get:
      tags: [Health]
      summary: PostgreSQL 和 Redis 就绪检查
      operationId: ready
      security: []
      responses: {'200': {$ref: '#/components/responses/Health'}, '503': {$ref: '#/components/responses/ReadyUnavailable'}}
  /api/v1/auth/login:
    post:
      tags: [Auth]
      summary: 账号密码登录
      operationId: login
      security: []
      requestBody: {$ref: '#/components/requestBodies/Login'}
      responses: {'200': {$ref: '#/components/responses/Login'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/auth/logout:
    get:
      tags: [Auth]
      summary: 退出当前会话
      operationId: logout
      x-permission: 'GET:/api/v1/auth/logout'
      responses: {'200': {$ref: '#/components/responses/Empty'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/users/password:
    put:
      tags: [Users]
      summary: 修改本人密码，成功后重新登录
      operationId: changePassword
      x-permission: 'PUT:/api/v1/users/password'
      requestBody: {$ref: '#/components/requestBodies/Password'}
      responses: {'200': {$ref: '#/components/responses/Empty'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/users/profile:
    get:
      tags: [Users]
      summary: 获取本人资料
      operationId: getProfile
      x-permission: 'GET:/api/v1/users/profile'
      responses: {'200': {$ref: '#/components/responses/User'}, default: {$ref: '#/components/responses/Error'}}
    put:
      tags: [Users]
      summary: 修改本人资料
      operationId: updateProfile
      x-permission: 'PUT:/api/v1/users/profile'
      requestBody: {$ref: '#/components/requestBodies/UserProfile'}
      responses: {'200': {$ref: '#/components/responses/User'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/users/statistics/weekly:
    get:
      tags: [Users]
      summary: 查询本人本周新增资源及资源总数统计
      operationId: getCurrentUserWeeklyStatistics
      x-permission: 'GET:/api/v1/users/statistics/weekly'
      responses: {'200': {$ref: '#/components/responses/UserWeeklyStatistics'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/users:
    get:
      tags: [Users]
      summary: 分页筛选用户
      operationId: listUsers
      x-permission: 'GET:/api/v1/users'
      parameters: [{$ref: '#/components/parameters/Page'}, {$ref: '#/components/parameters/PageSize'}, {$ref: '#/components/parameters/UserAccount'}, {$ref: '#/components/parameters/UserName'}, {$ref: '#/components/parameters/UserEmail'}, {$ref: '#/components/parameters/UserPhone'}, {$ref: '#/components/parameters/UserStatus'}]
      responses: {'200': {$ref: '#/components/responses/Users'}, default: {$ref: '#/components/responses/Error'}}
    post:
      tags: [Users]
      summary: 管理员新增用户
      operationId: createUser
      x-permission: 'POST:/api/v1/users'
      requestBody: {$ref: '#/components/requestBodies/UserCreate'}
      responses: {'200': {$ref: '#/components/responses/User'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/users/{id}:
    parameters: [{$ref: '#/components/parameters/ID'}]
    get:
      tags: [Users]
      summary: 用户详情
      operationId: getUser
      x-permission: 'GET:/api/v1/users/:id'
      responses: {'200': {$ref: '#/components/responses/User'}, default: {$ref: '#/components/responses/Error'}}
    put:
      tags: [Users]
      summary: 管理员修改用户资料
      operationId: updateUser
      x-permission: 'PUT:/api/v1/users/:id'
      requestBody: {$ref: '#/components/requestBodies/UserProfile'}
      responses: {'200': {$ref: '#/components/responses/User'}, default: {$ref: '#/components/responses/Error'}}
    delete:
      tags: [Users]
      summary: 注销当前登录用户（非物理删除；当前实现忽略路径 id）
      operationId: cancelUser
      x-permission: 'DELETE:/api/v1/users/:id'
      responses: {'200': {$ref: '#/components/responses/User'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/users/{id}/status:
    parameters: [{$ref: '#/components/parameters/ID'}]
    put:
      tags: [Users]
      summary: 更改用户状态
      operationId: updateUserStatus
      x-permission: 'PUT:/api/v1/users/:id/status'
      requestBody: {$ref: '#/components/requestBodies/UserStatus'}
      responses: {'200': {$ref: '#/components/responses/User'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/users/{id}/roles:
    parameters: [{$ref: '#/components/parameters/ID'}]
    put:
      tags: [Users]
      summary: 覆盖用户关联的角色 ID 列表
      operationId: assignUserRoles
      x-permission: 'PUT:/api/v1/users/:id/roles'
      requestBody: {$ref: '#/components/requestBodies/IDs'}
      responses: {'200': {$ref: '#/components/responses/Empty'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/roles:
    get:
      tags: [Roles]
      summary: 分页查询角色
      operationId: listRoles
      x-permission: 'GET:/api/v1/roles'
      parameters: [{$ref: '#/components/parameters/Page'}, {$ref: '#/components/parameters/PageSize'}, {$ref: '#/components/parameters/RoleName'}, {$ref: '#/components/parameters/RoleCode'}, {$ref: '#/components/parameters/RoleStatus'}]
      responses: {'200': {$ref: '#/components/responses/Roles'}, default: {$ref: '#/components/responses/Error'}}
    post:
      tags: [Roles]
      summary: 创建角色
      operationId: createRole
      x-permission: 'POST:/api/v1/roles'
      requestBody: {$ref: '#/components/requestBodies/RoleCreate'}
      responses: {'200': {$ref: '#/components/responses/Role'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/roles/{id}:
    parameters: [{$ref: '#/components/parameters/ID'}]
    get:
      tags: [Roles]
      summary: 角色详情
      operationId: getRole
      x-permission: 'GET:/api/v1/roles/:id'
      responses: {'200': {$ref: '#/components/responses/Role'}, default: {$ref: '#/components/responses/Error'}}
    put:
      tags: [Roles]
      summary: 更新角色
      operationId: updateRole
      x-permission: 'PUT:/api/v1/roles/:id'
      requestBody: {$ref: '#/components/requestBodies/RoleUpdate'}
      responses: {'200': {$ref: '#/components/responses/Role'}, default: {$ref: '#/components/responses/Error'}}
    delete:
      tags: [Roles]
      summary: 删除未绑定用户的非内置角色
      operationId: deleteRole
      x-permission: 'DELETE:/api/v1/roles/:id'
      responses: {'200': {$ref: '#/components/responses/Empty'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/roles/{id}/status:
    parameters: [{$ref: '#/components/parameters/ID'}]
    put:
      tags: [Roles]
      summary: 修改角色状态
      operationId: updateRoleStatus
      x-permission: 'PUT:/api/v1/roles/:id/status'
      requestBody: {$ref: '#/components/requestBodies/RoleStatus'}
      responses: {'200': {$ref: '#/components/responses/Role'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/roles/{id}/menus:
    parameters: [{$ref: '#/components/parameters/ID'}]
    put:
      tags: [Roles]
      summary: 覆盖角色关联的菜单 ID 列表
      operationId: assignRoleMenus
      x-permission: 'PUT:/api/v1/roles/:id/menus'
      requestBody: {$ref: '#/components/requestBodies/IDs'}
      responses: {'200': {$ref: '#/components/responses/Empty'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/menus/tree:
    get:
      tags: [Menus]
      summary: 查询菜单树
      operationId: listMenuTree
      x-permission: 'GET:/api/v1/menus/tree'
      parameters: [{$ref: '#/components/parameters/MenuStatus'}, {$ref: '#/components/parameters/MenuVisible'}, {$ref: '#/components/parameters/MenuType'}]
      responses: {'200': {$ref: '#/components/responses/MenuTree'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/menus:
    post:
      tags: [Menus]
      summary: 创建菜单或 API 权限资源
      operationId: createMenu
      x-permission: 'POST:/api/v1/menus'
      requestBody: {$ref: '#/components/requestBodies/MenuCreate'}
      responses: {'200': {$ref: '#/components/responses/Menu'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/menus/{id}:
    parameters: [{$ref: '#/components/parameters/ID'}]
    put:
      tags: [Menus]
      summary: 更新菜单或 API 权限资源
      operationId: updateMenu
      x-permission: 'PUT:/api/v1/menus/:id'
      requestBody: {$ref: '#/components/requestBodies/MenuUpdate'}
      responses: {'200': {$ref: '#/components/responses/Menu'}, default: {$ref: '#/components/responses/Error'}}
    delete:
      tags: [Menus]
      summary: 删除菜单或 API 权限资源
      operationId: deleteMenu
      x-permission: 'DELETE:/api/v1/menus/:id'
      responses: {'200': {$ref: '#/components/responses/Empty'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/notebooks:
    get:
      tags: [Notebooks]
      summary: 分页筛选本人笔记簿
      description: 列表项的 noteCount 为该笔记簿下笔记总数（包含回收站，不含草稿）；无笔记时为 0。
      operationId: listNotebooks
      x-permission: 'GET:/api/v1/notebooks'
      parameters: [{$ref: '#/components/parameters/Page'}, {$ref: '#/components/parameters/PageSize'}, {$ref: '#/components/parameters/NotebookName'}, {$ref: '#/components/parameters/NotebookStatus'}]
      responses: {'200': {$ref: '#/components/responses/Notebooks'}, default: {$ref: '#/components/responses/Error'}}
    post:
      tags: [Notebooks]
      summary: 创建笔记簿
      operationId: createNotebook
      x-permission: 'POST:/api/v1/notebooks'
      requestBody: {$ref: '#/components/requestBodies/NotebookCreate'}
      responses: {'200': {$ref: '#/components/responses/Notebook'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/notebooks/{id}:
    parameters: [{$ref: '#/components/parameters/ID'}]
    get:
      tags: [Notebooks]
      summary: 获取笔记簿详情
      operationId: getNotebook
      x-permission: 'GET:/api/v1/notebooks/:id'
      responses: {'200': {$ref: '#/components/responses/Notebook'}, default: {$ref: '#/components/responses/Error'}}
    put:
      tags: [Notebooks]
      summary: 修改笔记簿资料
      operationId: updateNotebook
      x-permission: 'PUT:/api/v1/notebooks/:id'
      requestBody: {$ref: '#/components/requestBodies/NotebookUpdate'}
      responses: {'200': {$ref: '#/components/responses/Notebook'}, default: {$ref: '#/components/responses/Error'}}
    delete:
      tags: [Notebooks]
      summary: 删除空笔记簿；包含笔记时返回冲突
      operationId: deleteNotebook
      x-permission: 'DELETE:/api/v1/notebooks/:id'
      responses: {'200': {$ref: '#/components/responses/Empty'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/notebooks/{id}/sort:
    parameters: [{$ref: '#/components/parameters/ID'}]
    put:
      tags: [Notebooks]
      summary: 修改笔记簿排序
      operationId: sortNotebook
      x-permission: 'PUT:/api/v1/notebooks/:id/sort'
      requestBody: {$ref: '#/components/requestBodies/NotebookSort'}
      responses: {'200': {$ref: '#/components/responses/Notebook'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/notebooks/{id}/status:
    parameters: [{$ref: '#/components/parameters/ID'}]
    put:
      tags: [Notebooks]
      summary: 修改笔记簿状态
      operationId: statusNotebook
      x-permission: 'PUT:/api/v1/notebooks/:id/status'
      requestBody: {$ref: '#/components/requestBodies/NotebookStatus'}
      responses: {'200': {$ref: '#/components/responses/Notebook'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/notes:
    get:
      tags: [Notes]
      summary: 分页筛选本人正常笔记，列表不加载正文
      operationId: listNotes
      x-permission: 'GET:/api/v1/notes'
      parameters: [{$ref: '#/components/parameters/Page'}, {$ref: '#/components/parameters/PageSize'}, {$ref: '#/components/parameters/NotebookID'}, {$ref: '#/components/parameters/NoteTitle'}, {$ref: '#/components/parameters/NoteLabel'}, {$ref: '#/components/parameters/UpdatedFrom'}, {$ref: '#/components/parameters/UpdatedTo'}]
      responses: {'200': {$ref: '#/components/responses/Notes'}, default: {$ref: '#/components/responses/Error'}}
    post:
      tags: [Notes]
      summary: 直接创建正式笔记
      operationId: createNote
      x-permission: 'POST:/api/v1/notes'
      requestBody: {$ref: '#/components/requestBodies/NoteCreate'}
      responses: {'200': {$ref: '#/components/responses/Note'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/notes/{id}:
    parameters: [{$ref: '#/components/parameters/ID'}]
    get:
      tags: [Notes]
      summary: 获取正常笔记详情及正文
      operationId: getNote
      x-permission: 'GET:/api/v1/notes/:id'
      responses: {'200': {$ref: '#/components/responses/Note'}, default: {$ref: '#/components/responses/Error'}}
    put:
      tags: [Notes]
      summary: 修改正常笔记，需提交当前版本号
      operationId: updateNote
      x-permission: 'PUT:/api/v1/notes/:id'
      requestBody: {$ref: '#/components/requestBodies/NoteUpdate'}
      responses: {'200': {$ref: '#/components/responses/Note'}, default: {$ref: '#/components/responses/Error'}}
    delete:
      tags: [Notes]
      summary: 将正常笔记移入回收站
      operationId: trashNote
      x-permission: 'DELETE:/api/v1/notes/:id'
      responses: {'200': {$ref: '#/components/responses/Empty'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/notes/{id}/move:
    parameters: [{$ref: '#/components/parameters/ID'}]
    put:
      tags: [Notes]
      summary: 移动正常笔记到其他笔记簿，需提交当前版本号
      operationId: moveNote
      x-permission: 'PUT:/api/v1/notes/:id/move'
      requestBody: {$ref: '#/components/requestBodies/NoteMove'}
      responses: {'200': {$ref: '#/components/responses/Note'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/notes/{id}/restore:
    parameters: [{$ref: '#/components/parameters/ID'}]
    put:
      tags: [Trash]
      summary: 从回收站恢复笔记（兼容笔记路径）
      operationId: restoreNote
      x-permission: 'PUT:/api/v1/notes/:id/restore'
      requestBody: {$ref: '#/components/requestBodies/RestoreNote'}
      responses: {'200': {$ref: '#/components/responses/Note'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/notes/{id}/permanent:
    parameters: [{$ref: '#/components/parameters/ID'}]
    delete:
      tags: [Trash]
      summary: 永久删除回收站笔记（兼容笔记路径）
      operationId: permanentDeleteNote
      x-permission: 'DELETE:/api/v1/notes/:id/permanent'
      responses: {'200': {$ref: '#/components/responses/Empty'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/trash/notes:
    get:
      tags: [Trash]
      summary: 分页查看本人回收站笔记
      operationId: listTrashNotes
      x-permission: 'GET:/api/v1/trash/notes'
      parameters: [{$ref: '#/components/parameters/Page'}, {$ref: '#/components/parameters/PageSize'}]
      responses: {'200': {$ref: '#/components/responses/Notes'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/trash/notes/{id}/restore:
    parameters: [{$ref: '#/components/parameters/ID'}]
    put:
      tags: [Trash]
      summary: 恢复笔记；原笔记簿不可用时需指定 notebookId
      operationId: restoreTrashNote
      x-permission: 'PUT:/api/v1/trash/notes/:id/restore'
      requestBody: {$ref: '#/components/requestBodies/RestoreNote'}
      responses: {'200': {$ref: '#/components/responses/Note'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/trash/notes/{id}:
    parameters: [{$ref: '#/components/parameters/ID'}]
    delete:
      tags: [Trash]
      summary: 永久删除回收站笔记
      operationId: permanentDeleteTrashNote
      x-permission: 'DELETE:/api/v1/trash/notes/:id'
      responses: {'200': {$ref: '#/components/responses/Empty'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/bookmarks:
    get:
      tags: [Bookmarks]
      summary: 分页筛选本人收藏
      operationId: listBookmarks
      x-permission: 'GET:/api/v1/bookmarks'
      parameters: [{$ref: '#/components/parameters/Page'}, {$ref: '#/components/parameters/PageSize'}, {$ref: '#/components/parameters/BookmarkTopic'}, {$ref: '#/components/parameters/BookmarkLabel'}, {$ref: '#/components/parameters/BookmarkHref'}, {$ref: '#/components/parameters/BookmarkFavorite'}, {$ref: '#/components/parameters/CreatedFrom'}, {$ref: '#/components/parameters/CreatedTo'}]
      responses: {'200': {$ref: '#/components/responses/Bookmarks'}, default: {$ref: '#/components/responses/Error'}}
    post:
      tags: [Bookmarks]
      summary: 新增收藏
      operationId: createBookmark
      x-permission: 'POST:/api/v1/bookmarks'
      requestBody: {$ref: '#/components/requestBodies/BookmarkCreate'}
      responses: {'200': {$ref: '#/components/responses/Bookmark'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/bookmarks/{id}:
    parameters: [{$ref: '#/components/parameters/ID'}]
    get:
      tags: [Bookmarks]
      summary: 收藏详情
      operationId: getBookmark
      x-permission: 'GET:/api/v1/bookmarks/:id'
      responses: {'200': {$ref: '#/components/responses/Bookmark'}, default: {$ref: '#/components/responses/Error'}}
    put:
      tags: [Bookmarks]
      summary: 修改收藏
      operationId: updateBookmark
      x-permission: 'PUT:/api/v1/bookmarks/:id'
      requestBody: {$ref: '#/components/requestBodies/BookmarkUpdate'}
      responses: {'200': {$ref: '#/components/responses/Bookmark'}, default: {$ref: '#/components/responses/Error'}}
    delete:
      tags: [Bookmarks]
      summary: 删除收藏
      operationId: deleteBookmark
      x-permission: 'DELETE:/api/v1/bookmarks/:id'
      responses: {'200': {$ref: '#/components/responses/Empty'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/drafts:
    get:
      tags: [Drafts]
      summary: 分页查看本人草稿箱（不加载正文）
      operationId: listDrafts
      x-permission: 'GET:/api/v1/drafts'
      parameters: [{$ref: '#/components/parameters/Page'}, {$ref: '#/components/parameters/PageSize'}]
      responses: {'200': {$ref: '#/components/responses/Drafts'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/drafts/{draftKey}:
    put:
      tags: [Drafts]
      summary: 草稿首次保存及后续更新使用同一接口
      operationId: saveDraft
      x-permission: 'PUT:/api/v1/drafts/:draftKey'
      parameters: [{$ref: '#/components/parameters/DraftKey'}]
      requestBody: {$ref: '#/components/requestBodies/DraftSave'}
      responses: {'200': {$ref: '#/components/responses/Draft'}, default: {$ref: '#/components/responses/Error'}}
    get:
      tags: [Drafts]
      summary: 草稿详情及正文
      operationId: getDraft
      x-permission: 'GET:/api/v1/drafts/:id'
      parameters: [{$ref: '#/components/parameters/DraftID'}]
      responses: {'200': {$ref: '#/components/responses/Draft'}, default: {$ref: '#/components/responses/Error'}}
    delete:
      tags: [Drafts]
      summary: 删除草稿（不删除正式笔记）
      operationId: deleteDraft
      x-permission: 'DELETE:/api/v1/drafts/:id'
      parameters: [{$ref: '#/components/parameters/DraftID'}]
      responses: {'200': {$ref: '#/components/responses/Empty'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/drafts/{id}/submit:
    parameters: [{$ref: '#/components/parameters/ID'}]
    post:
      tags: [Drafts]
      summary: 草稿提交为正式笔记并删除草稿
      operationId: submitDraft
      x-permission: 'POST:/api/v1/drafts/:id/submit'
      requestBody: {$ref: '#/components/requestBodies/DraftSubmit'}
      responses: {'200': {$ref: '#/components/responses/Note'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/files:
    get:
      tags: [Files]
      summary: 分页筛选本人文件元数据
      operationId: listFiles
      x-permission: 'GET:/api/v1/files'
      parameters: [{$ref: '#/components/parameters/Page'}, {$ref: '#/components/parameters/PageSize'}, {$ref: '#/components/parameters/FileName'}, {$ref: '#/components/parameters/FileType'}]
      responses: {'200': {$ref: '#/components/responses/Files'}, default: {$ref: '#/components/responses/Error'}}
    post:
      tags: [Files]
      summary: multipart 上传文件
      description: 上传成功后写入审计日志。
      operationId: uploadFile
      x-permission: 'POST:/api/v1/files'
      requestBody: {$ref: '#/components/requestBodies/FileUpload'}
      responses: {'200': {$ref: '#/components/responses/File'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/files/base64:
    post:
      tags: [Files]
      summary: 上传 Base64 图片（纯 Base64 或 data URL）
      description: 上传成功后写入审计日志。
      operationId: uploadBase64Image
      x-permission: 'POST:/api/v1/files/base64'
      requestBody: {$ref: '#/components/requestBodies/Base64Image'}
      responses: {'200': {$ref: '#/components/responses/File'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/files/{id}:
    parameters: [{$ref: '#/components/parameters/ID'}]
    get:
      tags: [Files]
      summary: 文件元数据详情（不是文件字节流）
      operationId: getFile
      x-permission: 'GET:/api/v1/files/:id'
      responses: {'200': {$ref: '#/components/responses/File'}, default: {$ref: '#/components/responses/Error'}}
    delete:
      tags: [Files]
      summary: 删除本人文件和对象
      description: 文件对象删除且元数据标记删除成功后写入审计日志。
      operationId: deleteFile
      x-permission: 'DELETE:/api/v1/files/:id'
      responses: {'200': {$ref: '#/components/responses/Empty'}, default: {$ref: '#/components/responses/Error'}}
  /api/v1/files/public/{publicKey}/preview:
    parameters: [{$ref: '#/components/parameters/PublicKey'}]
    get:
      tags: [Files]
      summary: 公开预览文件
      description: 返回正常状态文件的字节流，响应地址由文件元数据中的 fileUrl 提供；接口无需携带会话令牌。任何持有 publicKey 的客户端都可以访问该地址，请勿将其用于存储敏感文件。
      operationId: previewFile
      responses:
        '200': {description: 文件内容}
        default: {$ref: '#/components/responses/Error'}
  /api/v1/files/{id}/download:
    parameters: [{$ref: '#/components/parameters/ID'}]
    get:
      tags: [Files]
      summary: 下载本人文件
      description: 返回文件字节流并通过 Content-Disposition: attachment 提示客户端下载；接口需要携带当前用户会话令牌。
      operationId: downloadFile
      x-permission: 'GET:/api/v1/files/:id/download'
      responses:
        '200': {description: 文件下载内容}
        default: {$ref: '#/components/responses/Error'}
  /api/v1/audit-logs:
    get:
      tags: [Audit]
      summary: 分页筛选审计日志（仅有审计权限的用户）
      operationId: listAuditLogs
      x-permission: 'GET:/api/v1/audit-logs'
      parameters: [{$ref: '#/components/parameters/Page'}, {$ref: '#/components/parameters/PageSize'}, {$ref: '#/components/parameters/AuditAccount'}, {$ref: '#/components/parameters/AuditOperation'}, {$ref: '#/components/parameters/AuditIP'}, {$ref: '#/components/parameters/OperatedFrom'}, {$ref: '#/components/parameters/OperatedTo'}]
      responses: {'200': {$ref: '#/components/responses/AuditLogs'}, default: {$ref: '#/components/responses/Error'}}
components:
  securitySchemes:
    sessionToken:
      type: apiKey
      in: header
      name: Authorization
      description: 登录返回的原始 token，不能添加 Bearer 前缀。
  parameters:
    ID: {name: id, in: path, required: true, description: 资源正整数 ID, schema: {type: integer, minimum: 1}}
    PublicKey: {name: publicKey, in: path, required: true, description: 文件公开访问标识，不是数据库 ID, schema: {type: string, minLength: 64, maxLength: 64, pattern: '^[0-9a-f]{64}$'}}
    DraftID: {name: draftKey, in: path, required: true, description: 'GET/DELETE 时传数字草稿 ID；实际 Gin 路由为 :id', schema: {type: string, pattern: '^[1-9][0-9]*$'}}
    DraftKey: {name: draftKey, in: path, required: true, description: 'PUT 时传稳定的编辑器草稿键；实际 Gin 路由为 :draftKey', schema: {type: string, minLength: 1, maxLength: 100}}
    Page: {name: page, in: query, description: 从 1 开始，默认 1, schema: {type: integer, minimum: 1, default: 1}}
    PageSize: {name: pageSize, in: query, description: 默认 20，最大 100, schema: {type: integer, minimum: 1, maximum: 100, default: 20}}
    UserAccount: {name: account, in: query, schema: {type: string, maxLength: 50}, description: 账号筛选（大小写不敏感）}
    UserName: {name: name, in: query, schema: {type: string, maxLength: 100}}
    UserEmail: {name: email, in: query, schema: {type: string, format: email, maxLength: 150}}
    UserPhone: {name: phone, in: query, schema: {type: string, maxLength: 30}}
    UserStatus: {name: status, in: query, schema: {type: integer, enum: [1, 2, 3]}}
    RoleName: {name: name, in: query, schema: {type: string, maxLength: 100}}
    RoleCode: {name: code, in: query, schema: {type: string, maxLength: 100}}
    RoleStatus: {name: status, in: query, schema: {type: integer, enum: [1, 2]}}
    MenuStatus: {name: status, in: query, schema: {type: integer, enum: [1, 2]}}
    MenuVisible: {name: visible, in: query, schema: {type: boolean}}
    MenuType: {name: type, in: query, schema: {type: integer, enum: [1, 2, 3, 4]}}
    NotebookName: {name: name, in: query, schema: {type: string, maxLength: 100}}
    NotebookStatus: {name: status, in: query, schema: {type: integer, enum: [1, 2]}}
    NotebookID: {name: notebookId, in: query, schema: {type: integer, minimum: 1}}
    NoteTitle: {name: title, in: query, schema: {type: string, maxLength: 200}, description: 笔记标题模糊匹配}
    NoteLabel: {name: label, in: query, schema: {type: string, maxLength: 200}}
    UpdatedFrom: {name: updatedFrom, in: query, schema: {type: string, format: date-time}}
    UpdatedTo: {name: updatedTo, in: query, schema: {type: string, format: date-time}}
    BookmarkTopic: {name: topic, in: query, schema: {type: string, maxLength: 200}}
    BookmarkLabel: {name: label, in: query, schema: {type: string, maxLength: 100}}
    BookmarkHref: {name: href, in: query, schema: {type: string, maxLength: 2048}}
    BookmarkFavorite: {name: favorite, in: query, schema: {type: boolean}}
    CreatedFrom: {name: createdFrom, in: query, schema: {type: string, format: date-time}}
    CreatedTo: {name: createdTo, in: query, schema: {type: string, format: date-time}}
    FileName: {name: fileName, in: query, schema: {type: string, maxLength: 255}}
    FileType: {name: fileType, in: query, schema: {type: string, maxLength: 100}}
    AuditAccount: {name: operatorAccount, in: query, schema: {type: string, maxLength: 50}}
    AuditOperation: {name: operation, in: query, schema: {type: string, maxLength: 200}}
    AuditIP: {name: ip, in: query, schema: {type: string}}
    OperatedFrom: {name: operatedFrom, in: query, schema: {type: string, format: date-time}}
    OperatedTo: {name: operatedTo, in: query, schema: {type: string, format: date-time}}
  requestBodies:
    Login: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/LoginInput'}}}}
    Password: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/PasswordInput'}}}}
    UserCreate: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/UserCreateInput'}}}}
    UserProfile: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/UserProfileInput'}}}}
    UserStatus: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/UserStatusInput'}}}}
    IDs: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/IDsInput'}}}}
    RoleCreate: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/RoleCreateInput'}}}}
    RoleUpdate: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/RoleInput'}}}}
    RoleStatus: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/RoleStatusInput'}}}}
    MenuCreate: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/MenuCreateInput'}}}}
    MenuUpdate: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/MenuInput'}}}}
    NotebookCreate: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/NotebookCreateInput'}}}}
    NotebookUpdate: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/NotebookProfileInput'}}}}
    NotebookSort: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/NotebookSortInput'}}}}
    NotebookStatus: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/NotebookStatusInput'}}}}
    NoteCreate: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/NoteCreateInput'}}}}
    NoteUpdate: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/NoteUpdateInput'}}}}
    NoteMove: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/NoteMoveInput'}}}}
    RestoreNote: {required: true, description: 可以传空对象；原笔记簿不可用时必须提供 notebookId, content: {application/json: {schema: {$ref: '#/components/schemas/RestoreNoteInput'}}}}
    BookmarkCreate: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/BookmarkCreateInput'}}}}
    BookmarkUpdate: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/BookmarkInput'}}}}
    DraftSave: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/DraftSaveInput'}}}}
    DraftSubmit: {required: true, description: '新笔记提交空对象；编辑已有笔记时提供 noteVersion', content: {application/json: {schema: {$ref: '#/components/schemas/DraftSubmitInput'}}}}
    FileUpload: {required: true, content: {multipart/form-data: {schema: {type: object, required: [file], properties: {file: {type: string, format: binary}}}}}}
    Base64Image: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/Base64ImageInput'}}}}
  responses:
    Error: {description: '400/40001、401/40101、403/40301、404/40401、409/40901 或 40902、413/41301、500/50001、503/50301；405/40001', content: {application/json: {schema: {$ref: '#/components/schemas/ErrorResponse'}}}}
    ReadyUnavailable: {description: '依赖不可用（HTTP 503，code 50301）', content: {application/json: {schema: {$ref: '#/components/schemas/HealthErrorResponse'}}}}
    Empty: {description: 操作成功，无数据, content: {application/json: {schema: {$ref: '#/components/schemas/EmptyResponse'}}}}
    Health: {description: 健康状态, content: {application/json: {schema: {$ref: '#/components/schemas/HealthResponse'}}}}
    Login: {description: 登录成功, content: {application/json: {schema: {$ref: '#/components/schemas/LoginResponse'}}}}
    User: {description: 用户资料, content: {application/json: {schema: {$ref: '#/components/schemas/UserResponse'}}}}
    UserWeeklyStatistics: {description: 本周新增资源及资源总数统计, content: {application/json: {schema: {$ref: '#/components/schemas/UserWeeklyStatisticsResponse'}}}}
    Users: {description: 用户分页, content: {application/json: {schema: {$ref: '#/components/schemas/UsersResponse'}}}}
    Role: {description: 角色详情, content: {application/json: {schema: {$ref: '#/components/schemas/RoleResponse'}}}}
    Roles: {description: 角色分页, content: {application/json: {schema: {$ref: '#/components/schemas/RolesResponse'}}}}
    Menu: {description: 菜单详情, content: {application/json: {schema: {$ref: '#/components/schemas/MenuResponse'}}}}
    MenuTree: {description: 菜单树, content: {application/json: {schema: {$ref: '#/components/schemas/MenuTreeResponse'}}}}
    Notebook: {description: 笔记簿详情, content: {application/json: {schema: {$ref: '#/components/schemas/NotebookResponse'}}}}
    Notebooks: {description: 笔记簿分页, content: {application/json: {schema: {$ref: '#/components/schemas/NotebooksResponse'}}}}
    Note: {description: 笔记详情, content: {application/json: {schema: {$ref: '#/components/schemas/NoteResponse'}}}}
    Notes: {description: 笔记分页, content: {application/json: {schema: {$ref: '#/components/schemas/NotesResponse'}}}}
    Bookmark: {description: 收藏详情, content: {application/json: {schema: {$ref: '#/components/schemas/BookmarkResponse'}}}}
    Bookmarks: {description: 收藏分页, content: {application/json: {schema: {$ref: '#/components/schemas/BookmarksResponse'}}}}
    Draft: {description: 草稿详情, content: {application/json: {schema: {$ref: '#/components/schemas/DraftResponse'}}}}
    Drafts: {description: 草稿分页, content: {application/json: {schema: {$ref: '#/components/schemas/DraftsResponse'}}}}
    File: {description: 文件元数据, content: {application/json: {schema: {$ref: '#/components/schemas/FileResponse'}}}}
    Files: {description: 文件分页, content: {application/json: {schema: {$ref: '#/components/schemas/FilesResponse'}}}}
    AuditLogs: {description: 审计日志分页, content: {application/json: {schema: {$ref: '#/components/schemas/AuditLogsResponse'}}}}
  schemas:
    LoginInput:
      type: object
      required: [account, password]
      properties:
        account: {type: string, maxLength: 50}
        password: {type: string, format: password, maxLength: 72}
    PasswordInput:
      type: object
      required: [oldPassword, newPassword]
      properties:
        oldPassword: {type: string, format: password, maxLength: 72}
        newPassword: {type: string, format: password, minLength: 8, maxLength: 72}
    UserInput:
      type: object
      properties:
        account: {type: string, minLength: 3, maxLength: 50, description: 仅创建用户可传}
        name: {type: string, minLength: 1, maxLength: 100}
        password: {type: string, format: password, minLength: 8, maxLength: 72, description: 仅创建用户可传}
        openId: {type: string, maxLength: 100}
        sex: {type: integer, enum: [0, 1, 2]}
        email: {type: string, format: email, maxLength: 150}
        phone: {type: string, maxLength: 30}
        avatarUrl: {type: string, maxLength: 500}
        status: {type: integer, enum: [1, 2, 3], description: 仅创建用户及状态接口可传}
    UserCreateInput: {allOf: [{$ref: '#/components/schemas/UserInput'}, {type: object, required: [account, name, password]}]}
    UserProfileInput:
      description: 允许 account、name、sex、email、phone、avatarUrl；至少传一个字段；openId、password、status 会被拒绝。
      type: object
      properties:
        account: {type: string, minLength: 3, maxLength: 50}
        name: {type: string, minLength: 1, maxLength: 100}
        sex: {type: integer, enum: [0, 1, 2]}
        email: {type: string, format: email, maxLength: 150}
        phone: {type: string, maxLength: 30}
        avatarUrl: {type: string, maxLength: 500}
    UserStatusInput: {type: object, required: [status], properties: {status: {type: integer, enum: [1, 2, 3]}}}
    UserWeeklyStatisticsData:
      type: object
      required: [weeklyNewNoteCount, totalNoteCount, weeklyNewNotebookCount, totalNotebookCount, weeklyNewFileCount, totalFileCount, weeklyNewBookmarkCount, totalBookmarkCount]
      properties:
        weeklyNewNoteCount: {type: integer, format: int64, minimum: 0, description: 本周新增笔记数}
        totalNoteCount: {type: integer, format: int64, minimum: 0, description: 当前用户笔记总数，包含回收站中的笔记}
        weeklyNewNotebookCount: {type: integer, format: int64, minimum: 0, description: 本周新增笔记簿数}
        totalNotebookCount: {type: integer, format: int64, minimum: 0, description: 当前用户笔记簿总数，包含已删除的笔记簿记录}
        weeklyNewFileCount: {type: integer, format: int64, minimum: 0, description: 本周新增文件数}
        totalFileCount: {type: integer, format: int64, minimum: 0, description: 当前用户文件总数，包含已删除的文件记录}
        weeklyNewBookmarkCount: {type: integer, format: int64, minimum: 0, description: 本周新增收藏数}
        totalBookmarkCount: {type: integer, format: int64, minimum: 0, description: 当前用户收藏总数}
    IDsInput: {type: object, required: [ids], properties: {ids: {type: array, items: {type: integer, minimum: 1}, description: '完整替换；空数组表示清空'}}}
    RoleInput:
      type: object
      properties:
        name: {type: string, maxLength: 100}
        code: {type: string, maxLength: 100}
        description: {type: string, maxLength: 500}
        status: {type: integer, enum: [1, 2]}
    RoleCreateInput: {allOf: [{$ref: '#/components/schemas/RoleInput'}, {type: object, required: [name, code]}]}
    RoleStatusInput: {type: object, required: [status], properties: {status: {type: integer, enum: [1, 2]}}}
    MenuInput:
      type: object
      properties:
        parentId: {type: integer, minimum: 0, nullable: true, description: 0 或省略表示顶级}
        name: {type: string, maxLength: 100}
        code: {type: string, maxLength: 150}
        type: {type: integer, enum: [1, 2, 3, 4]}
        routePath: {type: string, maxLength: 300}
        component: {type: string, maxLength: 300}
        httpMethod: {type: string, enum: [GET, POST, PUT, PATCH, DELETE, OPTIONS]}
        apiPath: {type: string, maxLength: 500, description: 'API 类型必填，填写 Gin 路由模板（如 /api/v1/users/:id）'}
        sortOrder: {type: integer}
        visible: {type: boolean}
        status: {type: integer, enum: [1, 2]}
    MenuCreateInput: {allOf: [{$ref: '#/components/schemas/MenuInput'}, {type: object, required: [name, code, type]}]}
    NotebookInput:
      type: object
      properties:
        name: {type: string, maxLength: 100}
        icon: {type: string, maxLength: 100, description: 封面图标名；可传空字符串清除}
        tag: {type: string, maxLength: 100}
        description: {type: string, maxLength: 1000}
        status: {type: integer, enum: [1, 2]}
        sortOrder: {type: integer, minimum: 0}
    NotebookCreateInput: {allOf: [{$ref: '#/components/schemas/NotebookInput'}, {type: object, required: [name]}]}
    NotebookProfileInput: {type: object, description: 至少传一个字段, properties: {name: {type: string, maxLength: 100}, icon: {type: string, maxLength: 100, description: 封面图标名；可传空字符串清除}, tag: {type: string, maxLength: 100}, description: {type: string, maxLength: 1000}}}
    NotebookSortInput: {type: object, required: [sortOrder], properties: {sortOrder: {type: integer, minimum: 0}}}
    NotebookStatusInput: {type: object, required: [status], properties: {status: {type: integer, enum: [1, 2]}}}
    NoteInput:
      type: object
      properties:
        notebookId: {type: integer, minimum: 1, description: 创建、移动时需要}
        title: {type: string, maxLength: 200}
        label: {type: string, maxLength: 200}
        summary: {type: string, maxLength: 2000}
        coverUrl: {type: string, maxLength: 500}
        content: {type: string, description: 正文最大 10 MiB}
        contentType: {type: string, enum: [markdown]}
        version: {type: integer, minimum: 1, description: 修改、移动时必填}
    NoteCreateInput: {allOf: [{$ref: '#/components/schemas/NoteInput'}, {type: object, required: [notebookId, title]}]}
    NoteUpdateInput:
      allOf: [{$ref: '#/components/schemas/NoteInput'}, {type: object, required: [version]}]
      description: 除 version 外至少传一个 title/label/summary/coverUrl/content/contentType；notebookId 不生效。
    NoteMoveInput: {type: object, required: [notebookId, version], properties: {notebookId: {type: integer, minimum: 1}, version: {type: integer, minimum: 1}}}
    RestoreNoteInput: {type: object, properties: {notebookId: {type: integer, minimum: 1}}}
    BookmarkInput:
      type: object
      properties:
        topic: {type: string, maxLength: 200}
        label: {type: string, maxLength: 100}
        href: {type: string, maxLength: 2048, description: HTTP 或 HTTPS URL}
        icon: {type: string, maxLength: 1000, description: 提交 href 时由后端抓取并覆盖；抓取失败时为空；未提交 href 时修改接口可单独修改图标}
        favorite: {type: boolean}
        description: {type: string, maxLength: 2000}
    BookmarkCreateInput: {allOf: [{$ref: '#/components/schemas/BookmarkInput'}, {type: object, required: [href]}]}
    DraftSaveInput:
      type: object
      description: 首次或继续自动保存；同一个 draftKey 持续使用。已关联的正式笔记不可切换到另一条。
      properties:
        noteId: {type: integer, minimum: 1, description: 编辑已存在笔记时提供}
        notebookId: {type: integer, minimum: 1}
        title: {type: string, maxLength: 200}
        label: {type: string, maxLength: 200}
        content: {type: string, description: 正文最大 10 MiB}
        contentType: {type: string, enum: [markdown]}
    DraftSubmitInput: {type: object, properties: {noteVersion: {type: integer, minimum: 1, description: 编辑已有正式笔记时必填}}}
    Base64ImageInput: {type: object, required: [fileName, base64], properties: {fileName: {type: string, maxLength: 255}, base64: {type: string, description: 纯 Base64 或带 MIME 声明的 data URL}}}
    Envelope:
      type: object
      required: [code, message, data, requestId]
      properties:
        code: {type: integer, enum: [0]}
        message: {type: string, example: success}
        requestId: {type: string, description: 用于排查的请求 ID}
    ErrorResponse:
      type: object
      required: [code, message, data, requestId]
      properties:
        code: {type: integer, enum: [40001, 40101, 40301, 40401, 40901, 40902, 41301, 50001, 50301]}
        message: {type: string}
        data: {type: object, nullable: true, description: 通常为 null}
        requestId: {type: string}
    EmptyResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, properties: {data: {type: object, nullable: true, example: null}}}]}
    HealthData:
      type: object
      required: [status, components]
      properties:
        status: {type: string, enum: [UP, DOWN]}
        components: {type: object, additionalProperties: {type: string, enum: [UP, DOWN]}}
    HealthResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, properties: {data: {$ref: '#/components/schemas/HealthData'}}}]}
    HealthErrorResponse:
      type: object
      required: [code, message, data, requestId]
      properties:
        code: {type: integer, enum: [50301]}
        message: {type: string}
        data: {$ref: '#/components/schemas/HealthData'}
        requestId: {type: string}
    LoginData:
      type: object
      required: [token, user, menuTree]
      properties:
        token: {type: string, description: 原样写入 Authorization 请求头}
        user: {$ref: '#/components/schemas/UserData'}
        menuTree: {type: array, items: {$ref: '#/components/schemas/MenuData'}}
    UserData:
      type: object
      required: [id, account, name, roles, status, loginMethod, createdAt, updatedAt]
      properties:
        id: {type: integer, minimum: 1}
        account: {type: string}
        name: {type: string}
        roles: {type: array, items: {$ref: '#/components/schemas/RoleData'}, description: 用户当前关联的角色列表；没有角色时返回空数组}
        openId: {type: string}
        sex: {type: integer, enum: [0, 1, 2]}
        email: {type: string, format: email}
        phone: {type: string}
        avatarUrl: {type: string}
        status: {type: integer, enum: [1, 2, 3]}
        lastLoginAt: {type: string, format: date-time}
        lastLoginIp: {type: string}
        loginMethod: {type: string}
        createdAt: {type: string, format: date-time}
        updatedAt: {type: string, format: date-time}
    RoleData:
      type: object
      required: [id, name, code, status, createdAt, updatedAt]
      properties:
        id: {type: integer, minimum: 1}
        name: {type: string}
        code: {type: string}
        description: {type: string}
        status: {type: integer, enum: [1, 2]}
        menus: {type: array, items: {$ref: '#/components/schemas/MenuData'}, description: 角色详情返回的已关联菜单和权限资源；角色列表及用户角色数组中省略}
        createdAt: {type: string, format: date-time}
        updatedAt: {type: string, format: date-time}
    MenuData:
      type: object
      required: [id, name, code, type, sortOrder, visible, status, createdAt, updatedAt]
      properties:
        id: {type: integer, minimum: 1}
        parentId: {type: integer, minimum: 1}
        name: {type: string}
        code: {type: string}
        type: {type: integer, enum: [1, 2, 3, 4]}
        routePath: {type: string}
        component: {type: string}
        httpMethod: {type: string}
        apiPath: {type: string}
        sortOrder: {type: integer}
        visible: {type: boolean}
        status: {type: integer, enum: [1, 2]}
        createdAt: {type: string, format: date-time}
        updatedAt: {type: string, format: date-time}
        children: {type: array, items: {$ref: '#/components/schemas/MenuData'}}
    NotebookData:
      type: object
      required: [id, name, icon, status, sortOrder, noteCount, createdAt, updatedAt]
      properties:
        id: {type: integer, minimum: 1}
        name: {type: string}
        icon: {type: string, description: 封面图标名；未设置时为空字符串}
        tag: {type: string}
        description: {type: string}
        status: {type: integer, enum: [1, 2]}
        sortOrder: {type: integer}
        noteCount: {type: integer, format: int64, minimum: 0, description: 当前笔记簿下笔记总数，包含回收站，不含草稿}
        createdAt: {type: string, format: date-time}
        updatedAt: {type: string, format: date-time}
    NoteData:
      type: object
      required: [id, notebookId, title, contentType, status, version, createdAt, updatedAt]
      properties:
        id: {type: integer, minimum: 1}
        notebookId: {type: integer, minimum: 1}
        title: {type: string}
        label: {type: string}
        summary: {type: string}
        coverUrl: {type: string}
        content: {type: string, description: 列表中省略}
        contentUri: {type: string, description: 列表中省略；正文存储引用}
        contentType: {type: string, enum: [markdown]}
        status: {type: integer, enum: [1, 2]}
        version: {type: integer, minimum: 1}
        createdAt: {type: string, format: date-time}
        updatedAt: {type: string, format: date-time}
        deletedAt: {type: string, format: date-time}
        purgeAt: {type: string, format: date-time}
        remainingRetentionDays: {type: integer, minimum: 0, description: 回收站列表剩余保留天数}
    BookmarkData:
      type: object
      required: [id, href, favorite, createdAt, updatedAt]
      properties:
        id: {type: integer, minimum: 1}
        topic: {type: string}
        label: {type: string}
        href: {type: string}
        icon: {type: string, description: 网站图标 URL；抓取失败时可能为空}
        favorite: {type: boolean}
        description: {type: string}
        createdAt: {type: string, format: date-time}
        updatedAt: {type: string, format: date-time}
    DraftData:
      type: object
      required: [id, draftKey, title, contentType, version, createdAt, updatedAt]
      properties:
        id: {type: integer, minimum: 1}
        draftKey: {type: string}
        noteId: {type: integer, minimum: 1}
        notebookId: {type: integer, minimum: 1}
        title: {type: string}
        label: {type: string}
        content: {type: string, description: 列表不加载正文}
        contentType: {type: string, enum: [markdown]}
        version: {type: integer, minimum: 1, description: 服务端修订号，无需回传}
        expiresAt: {type: string, format: date-time}
        createdAt: {type: string, format: date-time}
        updatedAt: {type: string, format: date-time}
    FileData:
      type: object
      required: [id, fileName, fileSize, createdAt, updatedAt]
      properties:
        id: {type: integer, minimum: 1}
        fileName: {type: string}
        fileUrl: {type: string, description: 文件公开预览 URL，无需 Token；地址中的 publicKey 不是数据库 ID}
        fileType: {type: string}
        fileSize: {type: integer, format: int64}
        checksum: {type: string}
        createdAt: {type: string, format: date-time}
        updatedAt: {type: string, format: date-time}
    AuditLogData:
      type: object
      required: [id, operatorAccount, operation, ip, location, operatedAt]
      properties:
        id: {type: integer, minimum: 1}
        operatorAccount: {type: string}
        operation: {type: string}
        ip: {type: string, nullable: true}
        location: {type: string, nullable: true}
        operatedAt: {type: string, format: date-time}
    PageData:
      type: object
      required: [list, page, pageSize, total]
      properties:
        page: {type: integer, minimum: 1}
        pageSize: {type: integer, minimum: 1, maximum: 100}
        total: {type: integer, format: int64, minimum: 0}
    UsersPage: {allOf: [{$ref: '#/components/schemas/PageData'}, {type: object, properties: {list: {type: array, items: {$ref: '#/components/schemas/UserData'}}}}]}
    RolesPage: {allOf: [{$ref: '#/components/schemas/PageData'}, {type: object, properties: {list: {type: array, items: {$ref: '#/components/schemas/RoleData'}}}}]}
    NotebooksPage: {allOf: [{$ref: '#/components/schemas/PageData'}, {type: object, properties: {list: {type: array, items: {$ref: '#/components/schemas/NotebookData'}}}}]}
    NotesPage: {allOf: [{$ref: '#/components/schemas/PageData'}, {type: object, properties: {list: {type: array, items: {$ref: '#/components/schemas/NoteData'}}}}]}
    BookmarksPage: {allOf: [{$ref: '#/components/schemas/PageData'}, {type: object, properties: {list: {type: array, items: {$ref: '#/components/schemas/BookmarkData'}}}}]}
    DraftsPage: {allOf: [{$ref: '#/components/schemas/PageData'}, {type: object, properties: {list: {type: array, items: {$ref: '#/components/schemas/DraftData'}}}}]}
    FilesPage: {allOf: [{$ref: '#/components/schemas/PageData'}, {type: object, properties: {list: {type: array, items: {$ref: '#/components/schemas/FileData'}}}}]}
    AuditLogsPage: {allOf: [{$ref: '#/components/schemas/PageData'}, {type: object, properties: {list: {type: array, items: {$ref: '#/components/schemas/AuditLogData'}}}}]}
    LoginResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/LoginData'}}}]}
    UserResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/UserData'}}}]}
    UserWeeklyStatisticsResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/UserWeeklyStatisticsData'}}}]}
    UsersResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/UsersPage'}}}]}
    RoleResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/RoleData'}}}]}
    RolesResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/RolesPage'}}}]}
    MenuResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/MenuData'}}}]}
    MenuTreeResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {type: array, items: {$ref: '#/components/schemas/MenuData'}}}}]}
    NotebookResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/NotebookData'}}}]}
    NotebooksResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/NotebooksPage'}}}]}
    NoteResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/NoteData'}}}]}
    NotesResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/NotesPage'}}}]}
    BookmarkResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/BookmarkData'}}}]}
    BookmarksResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/BookmarksPage'}}}]}
    DraftResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/DraftData'}}}]}
    DraftsResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/DraftsPage'}}}]}
    FileResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/FileData'}}}]}
    FilesResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/FilesPage'}}}]}
    AuditLogsResponse: {allOf: [{$ref: '#/components/schemas/Envelope'}, {type: object, required: [data], properties: {data: {$ref: '#/components/schemas/AuditLogsPage'}}}]}
```
