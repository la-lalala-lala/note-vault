CREATE SCHEMA IF NOT EXISTS note-vault;

SET search_path TO note-vault;

CREATE TABLE IF NOT EXISTS users
(
    id
    BIGINT
    GENERATED
    BY
    DEFAULT AS
    IDENTITY
    PRIMARY
    KEY,
    account
    VARCHAR
(
    50
) NOT NULL,
    name VARCHAR
(
    100
) NOT NULL,
    password VARCHAR
(
    255
) NOT NULL,
    open_id VARCHAR
(
    100
),
    sex SMALLINT,
    email VARCHAR
(
    150
),
    phone VARCHAR
(
    30
),
    avatar_url VARCHAR
(
    500
),
    status SMALLINT NOT NULL DEFAULT 1,
    last_login_at TIMESTAMPTZ,
    last_login_ip INET,
    login_method VARCHAR
(
    32
) NOT NULL DEFAULT 'PASSWORD',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT ck_users_status CHECK
(
    status
    IN
(
    1,
    2,
    3
)),
    CONSTRAINT ck_users_sex CHECK
(
    sex
    IS
    NULL
    OR
    sex
    IN
(
    0,
    1,
    2
))
    );

CREATE UNIQUE INDEX IF NOT EXISTS uq_users_account_lower
    ON users (LOWER (account));

CREATE TABLE IF NOT EXISTS roles
(
    id
    BIGINT
    GENERATED
    BY
    DEFAULT AS
    IDENTITY
    PRIMARY
    KEY,
    name
    VARCHAR
(
    100
) NOT NULL,
    code VARCHAR
(
    100
) NOT NULL,
    description VARCHAR
(
    500
),
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_roles_code UNIQUE
(
    code
),
    CONSTRAINT ck_roles_status CHECK
(
    status
    IN
(
    1,
    2
))
    );

CREATE TABLE IF NOT EXISTS menus
(
    id
    BIGINT
    GENERATED
    BY
    DEFAULT AS
    IDENTITY
    PRIMARY
    KEY,
    parent_id
    BIGINT,
    name
    VARCHAR
(
    100
) NOT NULL,
    code VARCHAR
(
    150
) NOT NULL,
    type SMALLINT NOT NULL,
    route_path VARCHAR
(
    300
),
    component VARCHAR
(
    300
),
    http_method VARCHAR
(
    10
),
    api_path VARCHAR
(
    500
),
    sort_order INTEGER NOT NULL DEFAULT 0,
    visible BOOLEAN NOT NULL DEFAULT TRUE,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_menus_code UNIQUE
(
    code
),
    CONSTRAINT fk_menus_parent
    FOREIGN KEY
(
    parent_id
) REFERENCES menus
(
    id
) ON DELETE RESTRICT,
    CONSTRAINT ck_menus_type CHECK
(
    type
    IN
(
    1,
    2,
    3,
    4
)),
    CONSTRAINT ck_menus_status CHECK
(
    status
    IN
(
    1,
    2
)),
    CONSTRAINT ck_menus_http_method CHECK
(
    http_method
    IS
    NULL
    OR
    http_method
    IN
(
    'GET',
    'POST',
    'PUT',
    'PATCH',
    'DELETE',
    'OPTIONS'
)
    ),
    CONSTRAINT ck_menus_api_definition CHECK
(
    type
    <>
    4
    OR
(
    http_method
    IS
    NOT
    NULL
    AND
    api_path
    IS
    NOT
    NULL
)
    )
    );

CREATE UNIQUE INDEX IF NOT EXISTS uq_menus_api_method_path
    ON menus (http_method, api_path)
    WHERE type = 4;

CREATE TABLE IF NOT EXISTS user_roles
(
    user_id
    BIGINT
    NOT
    NULL,
    role_id
    BIGINT
    NOT
    NULL,
    created_at
    TIMESTAMPTZ
    NOT
    NULL
    DEFAULT
    CURRENT_TIMESTAMP,
    PRIMARY
    KEY
(
    user_id,
    role_id
),
    CONSTRAINT fk_user_roles_user
    FOREIGN KEY
(
    user_id
) REFERENCES users
(
    id
) ON DELETE CASCADE,
    CONSTRAINT fk_user_roles_role
    FOREIGN KEY
(
    role_id
) REFERENCES roles
(
    id
)
  ON DELETE CASCADE
    );

CREATE TABLE IF NOT EXISTS role_menus
(
    role_id
    BIGINT
    NOT
    NULL,
    menu_id
    BIGINT
    NOT
    NULL,
    created_at
    TIMESTAMPTZ
    NOT
    NULL
    DEFAULT
    CURRENT_TIMESTAMP,
    PRIMARY
    KEY
(
    role_id,
    menu_id
),
    CONSTRAINT fk_role_menus_role
    FOREIGN KEY
(
    role_id
) REFERENCES roles
(
    id
) ON DELETE CASCADE,
    CONSTRAINT fk_role_menus_menu
    FOREIGN KEY
(
    menu_id
) REFERENCES menus
(
    id
)
  ON DELETE CASCADE
    );

CREATE TABLE IF NOT EXISTS notebooks
(
    id
    BIGINT
    GENERATED
    BY
    DEFAULT AS
    IDENTITY
    PRIMARY
    KEY,
    owner_id
    BIGINT
    NOT
    NULL,
    name
    VARCHAR
(
    100
) NOT NULL,
    icon VARCHAR(100) NOT NULL DEFAULT '',
    tag VARCHAR
(
    100
),
    description TEXT,
    status SMALLINT NOT NULL DEFAULT 1,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_notebooks_owner
    FOREIGN KEY
(
    owner_id
) REFERENCES users
(
    id
) ON DELETE RESTRICT,
    CONSTRAINT ck_notebooks_status CHECK
(
    status
    IN
(
    1,
    2,
    3
))
    );

CREATE UNIQUE INDEX IF NOT EXISTS uq_notebooks_owner_name
    ON notebooks (owner_id, LOWER (name))
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS notes
(
    id
    BIGINT
    GENERATED
    BY
    DEFAULT AS
    IDENTITY
    PRIMARY
    KEY,
    owner_id
    BIGINT
    NOT
    NULL,
    notebook_id
    BIGINT
    NOT
    NULL,
    title
    VARCHAR
(
    200
) NOT NULL DEFAULT '',
    label VARCHAR
(
    200
),
    summary TEXT,
    cover_url VARCHAR
(
    500
),
    content TEXT,
    content_uri VARCHAR
(
    1000
),
    content_type VARCHAR
(
    50
) NOT NULL DEFAULT 'markdown',
    status SMALLINT NOT NULL DEFAULT 1,
    version BIGINT NOT NULL DEFAULT 1,
    deleted_at TIMESTAMPTZ,
    deleted_by BIGINT,
    purge_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_notes_owner
    FOREIGN KEY
(
    owner_id
) REFERENCES users
(
    id
) ON DELETE RESTRICT,
    CONSTRAINT fk_notes_notebook
    FOREIGN KEY
(
    notebook_id
) REFERENCES notebooks
(
    id
)
  ON DELETE RESTRICT,
    CONSTRAINT fk_notes_deleted_by
    FOREIGN KEY
(
    deleted_by
) REFERENCES users
(
    id
)
  ON DELETE SET NULL,
    CONSTRAINT ck_notes_status CHECK
(
    status
    IN
(
    1,
    2
)),
    CONSTRAINT ck_notes_version CHECK
(
    version >
    0
),
    CONSTRAINT ck_notes_trash_time CHECK
(
    status =
    1
    OR
(
    deleted_at
    IS
    NOT
    NULL
    AND
    purge_at
    IS
    NOT
    NULL
)
    )
    );

CREATE TABLE IF NOT EXISTS drafts
(
    id
    BIGINT
    GENERATED
    BY
    DEFAULT AS
    IDENTITY
    PRIMARY
    KEY,
    owner_id
    BIGINT
    NOT
    NULL,
    draft_key
    VARCHAR
(
    100
) NOT NULL,
    note_id BIGINT,
    notebook_id BIGINT,
    title VARCHAR
(
    200
) NOT NULL DEFAULT '',
    label VARCHAR
(
    200
),
    content TEXT,
    content_uri VARCHAR
(
    1000
),
    content_type VARCHAR
(
    50
) NOT NULL DEFAULT 'markdown',
    version BIGINT NOT NULL DEFAULT 1,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_drafts_owner
    FOREIGN KEY
(
    owner_id
) REFERENCES users
(
    id
) ON DELETE CASCADE,
    CONSTRAINT fk_drafts_note
    FOREIGN KEY
(
    note_id
) REFERENCES notes
(
    id
)
  ON DELETE SET NULL,
    CONSTRAINT fk_drafts_notebook
    FOREIGN KEY
(
    notebook_id
) REFERENCES notebooks
(
    id
)
  ON DELETE SET NULL,
    CONSTRAINT ck_drafts_version CHECK
(
    version >
    0
)
    );

CREATE UNIQUE INDEX IF NOT EXISTS uq_drafts_owner_key
    ON drafts (owner_id, draft_key);

CREATE UNIQUE INDEX IF NOT EXISTS uq_drafts_owner_note
    ON drafts (owner_id, note_id)
    WHERE note_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS bookmarks
(
    id
    BIGINT
    GENERATED
    BY
    DEFAULT AS
    IDENTITY
    PRIMARY
    KEY,
    owner_id
    BIGINT
    NOT
    NULL,
    topic
    VARCHAR
(
    200
),
    label VARCHAR
(
    100
),
    href TEXT NOT NULL,
    icon TEXT,
    favorite BOOLEAN NOT NULL DEFAULT TRUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_bookmarks_owner
    FOREIGN KEY
(
    owner_id
) REFERENCES users
(
    id
) ON DELETE CASCADE
    );

CREATE UNIQUE INDEX IF NOT EXISTS uq_bookmarks_owner_href
    ON bookmarks (owner_id, href);

CREATE TABLE IF NOT EXISTS files
(
    id
    BIGINT
    GENERATED
    BY
    DEFAULT AS
    IDENTITY
    PRIMARY
    KEY,
    owner_id
    BIGINT
    NOT
    NULL,
    file_name
    VARCHAR
(
    255
) NOT NULL,
    object_key VARCHAR
(
    1000
) NOT NULL,
    public_key VARCHAR
(
    64
) NOT NULL,
    file_url VARCHAR
(
    1500
),
    file_type VARCHAR
(
    100
),
    file_size BIGINT NOT NULL DEFAULT 0,
    checksum VARCHAR
(
    128
),
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_files_owner
    FOREIGN KEY
(
    owner_id
) REFERENCES users
(
    id
) ON DELETE CASCADE,
    CONSTRAINT ck_files_status CHECK
(
    status
    IN
(
    1,
    2
)),
    CONSTRAINT ck_files_size CHECK
(
    file_size
    >=
    0
),
    CONSTRAINT uq_files_object_key UNIQUE
(
    object_key
),
    CONSTRAINT uq_files_public_key UNIQUE
(
    public_key
)
    );

CREATE TABLE IF NOT EXISTS audit_logs
(
    id
    BIGINT
    GENERATED
    BY
    DEFAULT AS
    IDENTITY
    PRIMARY
    KEY,
    operator_account
    VARCHAR
(
    50
) NOT NULL,
    operation VARCHAR
(
    1000
) NOT NULL,
    ip INET,
    location VARCHAR
(
    255
),
    operated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

CREATE INDEX IF NOT EXISTS idx_user_roles_role_id
    ON user_roles (role_id);

CREATE INDEX IF NOT EXISTS idx_role_menus_menu_id
    ON role_menus (menu_id);

CREATE INDEX IF NOT EXISTS idx_menus_parent_sort
    ON menus (parent_id, sort_order, id);

CREATE INDEX IF NOT EXISTS idx_notebooks_owner_status
    ON notebooks (owner_id, status, sort_order, id);

CREATE INDEX IF NOT EXISTS idx_notes_owner_status_updated
    ON notes (owner_id, status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_notes_owner_notebook_status
    ON notes (owner_id, notebook_id, status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_notes_purge_at
    ON notes (purge_at)
    WHERE status = 2;

CREATE INDEX IF NOT EXISTS idx_drafts_owner_updated
    ON drafts (owner_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_bookmarks_owner_updated
    ON bookmarks (owner_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_files_owner_created
    ON files (owner_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_account_operated
    ON audit_logs (operator_account, operated_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_operated_at
    ON audit_logs (operated_at DESC);

CREATE
OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at
= CURRENT_TIMESTAMP;
RETURN NEW;
END;
$$
LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE
    ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_roles_updated_at ON roles;
CREATE TRIGGER trg_roles_updated_at
    BEFORE UPDATE
    ON roles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_menus_updated_at ON menus;
CREATE TRIGGER trg_menus_updated_at
    BEFORE UPDATE
    ON menus
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_notebooks_updated_at ON notebooks;
CREATE TRIGGER trg_notebooks_updated_at
    BEFORE UPDATE
    ON notebooks
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_notes_updated_at ON notes;
CREATE TRIGGER trg_notes_updated_at
    BEFORE UPDATE
    ON notes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_drafts_updated_at ON drafts;
CREATE TRIGGER trg_drafts_updated_at
    BEFORE UPDATE
    ON drafts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_bookmarks_updated_at ON bookmarks;
CREATE TRIGGER trg_bookmarks_updated_at
    BEFORE UPDATE
    ON bookmarks
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_files_updated_at ON files;
CREATE TRIGGER trg_files_updated_at
    BEFORE UPDATE
    ON files
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 初始化用户。
INSERT INTO users (id, account, name, password, open_id, sex, email, phone, avatar_url, status,
                   last_login_at, last_login_ip, login_method, created_at, updated_at)
VALUES (1, 'lalala', 'lalala', '$2a$10$6v5Ad.BmpZNlJb1yQkn.E.dyBVKE/f3co5mrIFDT.V.3Ootankc9.',
        NULL, NULL, 'saya@saya.ac.cn', '17345225420', NULL, 1,
        '2026-09-28 03:06:14.623866 +00:00', '127.0.0.1', 'PASSWORD',
        '2026-09-27 13:50:01.833512 +00:00', '2026-10-01 07:26:49.827356 +00:00')
ON CONFLICT (id) DO UPDATE SET
    account = EXCLUDED.account,
    name = EXCLUDED.name,
    password = EXCLUDED.password,
    open_id = EXCLUDED.open_id,
    sex = EXCLUDED.sex,
    email = EXCLUDED.email,
    phone = EXCLUDED.phone,
    avatar_url = EXCLUDED.avatar_url,
    status = EXCLUDED.status,
    last_login_at = EXCLUDED.last_login_at,
    last_login_ip = EXCLUDED.last_login_ip,
    login_method = EXCLUDED.login_method,
    created_at = EXCLUDED.created_at,
    updated_at = EXCLUDED.updated_at;

-- 同步用户表 identity 序列，避免初始化用户的显式 ID 与后续自动生成 ID 冲突。
SELECT setval(pg_get_serial_sequence('users', 'id'), GREATEST((SELECT MAX(id) FROM users), 1), TRUE);

INSERT INTO roles (name, code, description)
VALUES ('超级管理员', 'SUPER_ADMIN', '拥有全部系统权限'),
       ('普通用户', 'USER', '拥有个人笔记和收藏相关权限') ON CONFLICT (code) DO NOTHING;

-- 为初始化用户授予超级管理员角色。
INSERT INTO user_roles (user_id, role_id, created_at)
VALUES (1, 1, '2026-10-01 07:32:52.051780 +00:00')
ON CONFLICT (user_id, role_id) DO UPDATE SET
    created_at = EXCLUDED.created_at;

-- 菜单目录和 API 权限初始化数据。
-- type=1 为顶级目录，type=2 为页面目录，type=4 为接口权限。
INSERT INTO menus (id, parent_id, name, code, type, route_path, component, http_method, api_path,
                   sort_order, visible, status, created_at, updated_at)
VALUES
    (11000, NULL, '系统管理', 'SYSTEM_SPACE', 1, NULL, NULL, NULL, NULL, 30, TRUE, 1,
     '2026-10-01 07:21:51.747667 +00:00', '2026-10-01 07:21:51.747667 +00:00'),
    (12000, NULL, '知识工作区', 'WORK_SPACE', 1, NULL, NULL, NULL, NULL, 20, TRUE, 1,
     '2026-10-01 07:21:51.747667 +00:00', '2026-10-01 07:21:51.747667 +00:00'),
    (13000, null, '主页', 'HOME_PAGE', 2, '/home', 'Home', null, null, 10, true, 1, '2026-10-01 07:21:51.747667 +00:00', '2026-10-01 10:03:33.761256 +00:00'),
    (11100, 11000, '用户管理', 'GROUP_USERS', 2, '/admin/users', 'UserList', NULL, NULL, 10, TRUE, 1,
     '2026-10-01 07:21:51.747667 +00:00', '2026-10-01 08:02:11.539903 +00:00'),
    (11200, 11000, '菜单管理', 'GROUP_MENUS', 2, '/admin/menus', 'MenuTree', NULL, NULL, 20, TRUE, 1,
     '2026-10-01 07:21:51.747667 +00:00', '2026-10-01 08:02:11.539903 +00:00'),
    (11300, 11000, '角色管理', 'GROUP_ROLES', 2, '/admin/roles', 'RoleList', NULL, NULL, 30, TRUE, 1,
     '2026-10-01 07:21:51.747667 +00:00', '2026-10-01 08:02:11.539903 +00:00'),
    (11400, 11000, '审计日志', 'GROUP_AUDIT_LOGS', 2, '/admin/audit-logs', 'AuditLogList', NULL, NULL, 40, TRUE, 1,
     '2026-10-01 07:21:51.747667 +00:00', '2026-10-01 07:21:51.747667 +00:00'),
    (12100, 12000, '笔记簿', 'GROUP_NOTEBOOKS', 2, '/notebooks', 'NotebookList', NULL, NULL, 10, TRUE, 1,
     '2026-10-01 07:21:51.747667 +00:00', '2026-10-01 07:21:51.747667 +00:00'),
    (12200, 12000, '笔记', 'GROUP_NOTES', 2, '/notes', 'NoteWorkspace', NULL, NULL, 20, TRUE, 1,
     '2026-10-01 07:21:51.747667 +00:00', '2026-10-01 07:21:51.747667 +00:00'),
    (12300, 12000, '废纸篓', 'GROUP_TRASH', 2, '/trash', 'TrashNoteList', NULL, NULL, 30, TRUE, 1,
     '2026-10-01 07:21:51.747667 +00:00', '2026-10-01 08:02:11.539903 +00:00'),
    (12400, 12000, '草稿箱', 'GROUP_DRAFTS', 2, '/drafts', 'DraftList', NULL, NULL, 40, TRUE, 1,
     '2026-10-01 07:21:51.747667 +00:00', '2026-10-01 07:21:51.747667 +00:00'),
    (12500, 12000, '收藏夹', 'GROUP_BOOKMARKS', 2, '/bookmarks', 'BookmarkList', NULL, NULL, 50, TRUE, 1,
     '2026-10-01 07:21:51.747667 +00:00', '2026-10-01 07:21:51.747667 +00:00'),
    (12600, 12000, '文件', 'GROUP_FILES', 2, '/files', 'FileList', NULL, NULL, 60, TRUE, 1,
     '2026-10-01 07:21:51.747667 +00:00', '2026-10-01 07:21:51.747667 +00:00'),
    (11101, 11100, '退出登录', 'AUTH_LOGOUT', 4, NULL, NULL, 'GET', '/api/v1/auth/logout', 10, FALSE, 1,
     '2026-10-01 07:21:51.898216 +00:00', '2026-10-01 07:21:51.898216 +00:00'),
    (11102, 11100, '修改本人密码', 'USER_PASSWORD_UPDATE', 4, NULL, NULL, 'PUT', '/api/v1/users/password', 20, FALSE, 1,
     '2026-10-01 07:21:51.898216 +00:00', '2026-10-01 07:21:51.898216 +00:00'),
    (11103, 11100, '查看本人资料', 'USER_SELF_DETAIL', 4, NULL, NULL, 'GET', '/api/v1/users/profile', 30, FALSE, 1,
     '2026-10-01 07:21:51.898216 +00:00', '2026-10-01 07:21:51.898216 +00:00'),
    (11104, 11100, '修改本人资料', 'USER_SELF_UPDATE', 4, NULL, NULL, 'PUT', '/api/v1/users/profile', 40, FALSE, 1,
     '2026-10-01 07:21:51.898216 +00:00', '2026-10-01 07:21:51.898216 +00:00'),
    (11112, 11100, '查询本周新增统计', 'USER_SELF_WEEKLY_STATISTICS', 4, NULL, NULL, 'GET', '/api/v1/users/statistics/weekly', 45, FALSE, 1,
     '2026-10-07 00:00:00 +08:00', '2026-10-07 00:00:00 +08:00'),
    (11105, 11100, '管理员新增用户', 'USER_ADMIN_CREATE', 4, NULL, NULL, 'POST', '/api/v1/users', 50, FALSE, 1,
     '2026-10-01 07:21:51.898216 +00:00', '2026-10-01 07:21:51.898216 +00:00'),
    (11106, 11100, '管理员查询用户列表', 'USER_ADMIN_LIST', 4, NULL, NULL, 'GET', '/api/v1/users', 60, FALSE, 1,
     '2026-10-01 07:21:51.898216 +00:00', '2026-10-01 07:21:51.898216 +00:00'),
    (11107, 11100, '管理员查询用户详情', 'USER_ADMIN_DETAIL', 4, NULL, NULL, 'GET', '/api/v1/users/:id', 70, FALSE, 1,
     '2026-10-01 07:21:51.898216 +00:00', '2026-10-01 07:21:51.898216 +00:00'),
    (11108, 11100, '管理员修改用户资料', 'USER_ADMIN_UPDATE', 4, NULL, NULL, 'PUT', '/api/v1/users/:id', 80, FALSE, 1,
     '2026-10-01 07:21:51.898216 +00:00', '2026-10-01 07:21:51.898216 +00:00'),
    (11109, 11100, '管理员修改用户状态', 'USER_ADMIN_STATUS', 4, NULL, NULL, 'PUT', '/api/v1/users/:id/status', 90, FALSE, 1,
     '2026-10-01 07:21:51.898216 +00:00', '2026-10-01 07:21:51.898216 +00:00'),
    (11110, 11100, '管理员注销用户', 'USER_ADMIN_CANCEL', 4, NULL, NULL, 'DELETE', '/api/v1/users/:id', 100, FALSE, 1,
     '2026-10-01 07:21:51.898216 +00:00', '2026-10-01 07:21:51.898216 +00:00'),
    (11111, 11100, '管理员分配用户角色', 'USER_ADMIN_ASSIGN_ROLES', 4, NULL, NULL, 'PUT', '/api/v1/users/:id/roles', 110, FALSE, 1,
     '2026-10-01 07:21:51.898216 +00:00', '2026-10-01 07:21:51.898216 +00:00'),
    (11201, 11200, '查询菜单树', 'MENU_TREE_LIST', 4, NULL, NULL, 'GET', '/api/v1/menus/tree', 10, FALSE, 1,
     '2026-10-01 07:21:52.197673 +00:00', '2026-10-01 07:21:52.197673 +00:00'),
    (11202, 11200, '新增菜单', 'MENU_CREATE', 4, NULL, NULL, 'POST', '/api/v1/menus', 20, FALSE, 1,
     '2026-10-01 07:21:52.197673 +00:00', '2026-10-01 07:21:52.197673 +00:00'),
    (11203, 11200, '修改菜单', 'MENU_UPDATE', 4, NULL, NULL, 'PUT', '/api/v1/menus/:id', 30, FALSE, 1,
     '2026-10-01 07:21:52.197673 +00:00', '2026-10-01 07:21:52.197673 +00:00'),
    (11204, 11200, '删除菜单', 'MENU_DELETE', 4, NULL, NULL, 'DELETE', '/api/v1/menus/:id', 40, FALSE, 1,
     '2026-10-01 07:21:52.197673 +00:00', '2026-10-01 07:21:52.197673 +00:00'),
    (11301, 11300, '查询角色列表', 'ROLE_LIST', 4, NULL, NULL, 'GET', '/api/v1/roles', 10, FALSE, 1,
     '2026-10-01 07:21:52.049447 +00:00', '2026-10-01 07:21:52.049447 +00:00'),
    (11302, 11300, '新增角色', 'ROLE_CREATE', 4, NULL, NULL, 'POST', '/api/v1/roles', 20, FALSE, 1,
     '2026-10-01 07:21:52.049447 +00:00', '2026-10-01 07:21:52.049447 +00:00'),
    (11303, 11300, '查询角色详情', 'ROLE_DETAIL', 4, NULL, NULL, 'GET', '/api/v1/roles/:id', 30, FALSE, 1,
     '2026-10-01 07:21:52.049447 +00:00', '2026-10-01 07:21:52.049447 +00:00'),
    (11304, 11300, '修改角色', 'ROLE_UPDATE', 4, NULL, NULL, 'PUT', '/api/v1/roles/:id', 40, FALSE, 1,
     '2026-10-01 07:21:52.049447 +00:00', '2026-10-01 07:21:52.049447 +00:00'),
    (11305, 11300, '修改角色状态', 'ROLE_STATUS_UPDATE', 4, NULL, NULL, 'PUT', '/api/v1/roles/:id/status', 50, FALSE, 1,
     '2026-10-01 07:21:52.049447 +00:00', '2026-10-01 07:21:52.049447 +00:00'),
    (11306, 11300, '删除角色', 'ROLE_DELETE', 4, NULL, NULL, 'DELETE', '/api/v1/roles/:id', 60, FALSE, 1,
     '2026-10-01 07:21:52.049447 +00:00', '2026-10-01 07:21:52.049447 +00:00'),
    (11307, 11300, '分配角色菜单', 'ROLE_ASSIGN_MENUS', 4, NULL, NULL, 'PUT', '/api/v1/roles/:id/menus', 70, FALSE, 1,
     '2026-10-01 07:21:52.049447 +00:00', '2026-10-01 07:21:52.049447 +00:00'),
    (11401, 11400, '查询审计日志', 'AUDIT_LOG_LIST', 4, NULL, NULL, 'GET', '/api/v1/audit-logs', 10, FALSE, 1,
     '2026-10-01 07:21:55.087669 +00:00', '2026-10-01 07:21:55.087669 +00:00'),
    (12101, 12100, '查询笔记簿列表', 'NOTEBOOK_LIST', 4, NULL, NULL, 'GET', '/api/v1/notebooks', 10, FALSE, 1,
     '2026-10-01 07:21:52.398183 +00:00', '2026-10-01 07:21:52.398183 +00:00'),
    (12102, 12100, '新增笔记簿', 'NOTEBOOK_CREATE', 4, NULL, NULL, 'POST', '/api/v1/notebooks', 20, FALSE, 1,
     '2026-10-01 07:21:52.398183 +00:00', '2026-10-01 07:21:52.398183 +00:00'),
    (12103, 12100, '查询笔记簿详情', 'NOTEBOOK_DETAIL', 4, NULL, NULL, 'GET', '/api/v1/notebooks/:id', 30, FALSE, 1,
     '2026-10-01 07:21:52.398183 +00:00', '2026-10-01 07:21:52.398183 +00:00'),
    (12104, 12100, '修改笔记簿', 'NOTEBOOK_UPDATE', 4, NULL, NULL, 'PUT', '/api/v1/notebooks/:id', 40, FALSE, 1,
     '2026-10-01 07:21:52.398183 +00:00', '2026-10-01 07:21:52.398183 +00:00'),
    (12105, 12100, '修改笔记簿排序', 'NOTEBOOK_SORT_UPDATE', 4, NULL, NULL, 'PUT', '/api/v1/notebooks/:id/sort', 50, FALSE, 1,
     '2026-10-01 07:21:52.398183 +00:00', '2026-10-01 07:21:52.398183 +00:00'),
    (12106, 12100, '修改笔记簿状态', 'NOTEBOOK_STATUS_UPDATE', 4, NULL, NULL, 'PUT', '/api/v1/notebooks/:id/status', 60, FALSE, 1,
     '2026-10-01 07:21:52.398183 +00:00', '2026-10-01 07:21:52.398183 +00:00'),
    (12107, 12100, '删除笔记簿', 'NOTEBOOK_DELETE', 4, NULL, NULL, 'DELETE', '/api/v1/notebooks/:id', 70, FALSE, 1,
     '2026-10-01 07:21:52.398183 +00:00', '2026-10-01 07:21:52.398183 +00:00'),
    (12201, 12200, '查询笔记列表', 'NOTE_LIST', 4, NULL, NULL, 'GET', '/api/v1/notes', 10, FALSE, 1,
     '2026-10-01 07:21:52.968212 +00:00', '2026-10-01 07:21:52.968212 +00:00'),
    (12202, 12200, '新增笔记', 'NOTE_CREATE', 4, NULL, NULL, 'POST', '/api/v1/notes', 20, FALSE, 1,
     '2026-10-01 07:21:52.968212 +00:00', '2026-10-01 07:21:52.968212 +00:00'),
    (12203, 12200, '查询笔记详情', 'NOTE_DETAIL', 4, NULL, NULL, 'GET', '/api/v1/notes/:id', 30, FALSE, 1,
     '2026-10-01 07:21:52.968212 +00:00', '2026-10-01 07:21:52.968212 +00:00'),
    (12204, 12200, '修改笔记', 'NOTE_UPDATE', 4, NULL, NULL, 'PUT', '/api/v1/notes/:id', 40, FALSE, 1,
     '2026-10-01 07:21:52.968212 +00:00', '2026-10-01 07:21:52.968212 +00:00'),
    (12205, 12200, '移动笔记', 'NOTE_MOVE', 4, NULL, NULL, 'PUT', '/api/v1/notes/:id/move', 50, FALSE, 1,
     '2026-10-01 07:21:52.968212 +00:00', '2026-10-01 07:21:52.968212 +00:00'),
    (12206, 12200, '删除笔记', 'NOTE_DELETE', 4, NULL, NULL, 'DELETE', '/api/v1/notes/:id', 60, FALSE, 1,
     '2026-10-01 07:21:52.968212 +00:00', '2026-10-01 07:21:52.968212 +00:00'),
    (12207, 12200, '恢复笔记', 'NOTE_RESTORE', 4, NULL, NULL, 'PUT', '/api/v1/notes/:id/restore', 70, FALSE, 1,
     '2026-10-01 07:21:52.968212 +00:00', '2026-10-01 07:21:52.968212 +00:00'),
    (12208, 12200, '永久删除笔记', 'NOTE_PERMANENT_DELETE', 4, NULL, NULL, 'DELETE', '/api/v1/notes/:id/permanent', 80, FALSE, 1,
     '2026-10-01 07:21:52.968212 +00:00', '2026-10-01 07:21:52.968212 +00:00'),
    (12301, 12300, '查询回收站笔记', 'TRASH_NOTE_LIST', 4, NULL, NULL, 'GET', '/api/v1/trash/notes', 10, FALSE, 1,
     '2026-10-01 07:21:53.637685 +00:00', '2026-10-01 07:21:53.637685 +00:00'),
    (12302, 12300, '恢复回收站笔记', 'TRASH_NOTE_RESTORE', 4, NULL, NULL, 'PUT', '/api/v1/trash/notes/:id/restore', 20, FALSE, 1,
     '2026-10-01 07:21:53.637685 +00:00', '2026-10-01 07:21:53.637685 +00:00'),
    (12303, 12300, '永久删除回收站笔记', 'TRASH_NOTE_PERMANENT_DELETE', 4, NULL, NULL, 'DELETE', '/api/v1/trash/notes/:id', 30, FALSE, 1,
     '2026-10-01 07:21:53.637685 +00:00', '2026-10-01 07:21:53.637685 +00:00'),
    (12401, 12400, '查询草稿列表', 'DRAFT_LIST', 4, NULL, NULL, 'GET', '/api/v1/drafts', 10, FALSE, 1,
     '2026-10-01 07:21:54.259476 +00:00', '2026-10-01 07:21:54.259476 +00:00'),
    (12402, 12400, '保存草稿', 'DRAFT_SAVE', 4, NULL, NULL, 'PUT', '/api/v1/drafts/:draftKey', 20, FALSE, 1,
     '2026-10-01 07:21:54.259476 +00:00', '2026-10-01 07:21:54.259476 +00:00'),
    (12403, 12400, '查询草稿详情', 'DRAFT_DETAIL', 4, NULL, NULL, 'GET', '/api/v1/drafts/:id', 30, FALSE, 1,
     '2026-10-01 07:21:54.259476 +00:00', '2026-10-01 07:21:54.259476 +00:00'),
    (12404, 12400, '提交草稿为笔记', 'DRAFT_SUBMIT', 4, NULL, NULL, 'POST', '/api/v1/drafts/:id/submit', 40, FALSE, 1,
     '2026-10-01 07:21:54.259476 +00:00', '2026-10-01 07:21:54.259476 +00:00'),
    (12405, 12400, '删除草稿', 'DRAFT_DELETE', 4, NULL, NULL, 'DELETE', '/api/v1/drafts/:id', 50, FALSE, 1,
     '2026-10-01 07:21:54.259476 +00:00', '2026-10-01 07:21:54.259476 +00:00'),
    (12501, 12500, '查询收藏列表', 'BOOKMARK_LIST', 4, NULL, NULL, 'GET', '/api/v1/bookmarks', 10, FALSE, 1,
     '2026-10-01 07:21:53.947520 +00:00', '2026-10-01 07:21:53.947520 +00:00'),
    (12502, 12500, '新增收藏', 'BOOKMARK_CREATE', 4, NULL, NULL, 'POST', '/api/v1/bookmarks', 20, FALSE, 1,
     '2026-10-01 07:21:53.947520 +00:00', '2026-10-01 07:21:53.947520 +00:00'),
    (12503, 12500, '查询收藏详情', 'BOOKMARK_DETAIL', 4, NULL, NULL, 'GET', '/api/v1/bookmarks/:id', 30, FALSE, 1,
     '2026-10-01 07:21:53.947520 +00:00', '2026-10-01 07:21:53.947520 +00:00'),
    (12504, 12500, '修改收藏', 'BOOKMARK_UPDATE', 4, NULL, NULL, 'PUT', '/api/v1/bookmarks/:id', 40, FALSE, 1,
     '2026-10-01 07:21:53.947520 +00:00', '2026-10-01 07:21:53.947520 +00:00'),
    (12505, 12500, '删除收藏', 'BOOKMARK_DELETE', 4, NULL, NULL, 'DELETE', '/api/v1/bookmarks/:id', 50, FALSE, 1,
     '2026-10-01 07:21:53.947520 +00:00', '2026-10-01 07:21:53.947520 +00:00'),
    (12601, 12600, '上传文件', 'FILE_UPLOAD', 4, NULL, NULL, 'POST', '/api/v1/files', 10, FALSE, 1,
     '2026-10-01 07:21:54.778179 +00:00', '2026-10-01 07:21:54.778179 +00:00'),
    (12602, 12600, '上传 Base64 图片', 'FILE_BASE64_IMAGE_UPLOAD', 4, NULL, NULL, 'POST', '/api/v1/files/base64', 20, FALSE, 1,
     '2026-10-01 07:21:54.778179 +00:00', '2026-10-01 07:21:54.778179 +00:00'),
    (12603, 12600, '查询文件列表', 'FILE_LIST', 4, NULL, NULL, 'GET', '/api/v1/files', 30, FALSE, 1,
     '2026-10-01 07:21:54.778179 +00:00', '2026-10-01 07:21:54.778179 +00:00'),
    (12604, 12600, '查询文件详情', 'FILE_DETAIL', 4, NULL, NULL, 'GET', '/api/v1/files/:id', 40, FALSE, 1,
     '2026-10-01 07:21:54.778179 +00:00', '2026-10-01 07:21:54.778179 +00:00'),
    (12605, 12600, '删除文件', 'FILE_DELETE', 4, NULL, NULL, 'DELETE', '/api/v1/files/:id', 50, FALSE, 1,
     '2026-10-01 07:21:54.778179 +00:00', '2026-10-01 07:21:54.778179 +00:00'),
    (12606, 12600, '预览文件', 'FILE_PREVIEW', 4, NULL, NULL, 'GET', '/api/v1/files/public/:publicKey/preview', 60, FALSE, 1,
     '2026-10-02 00:00:00 +00:00', '2026-10-02 00:00:00 +00:00'),
    (12607, 12600, '下载文件', 'FILE_DOWNLOAD', 4, NULL, NULL, 'GET', '/api/v1/files/:id/download', 70, FALSE, 1,
     '2026-10-02 00:00:00 +00:00', '2026-10-02 00:00:00 +00:00')
ON CONFLICT (code) DO UPDATE SET
    parent_id = EXCLUDED.parent_id,
    name = EXCLUDED.name,
    type = EXCLUDED.type,
    route_path = EXCLUDED.route_path,
    component = EXCLUDED.component,
    http_method = EXCLUDED.http_method,
    api_path = EXCLUDED.api_path,
    sort_order = EXCLUDED.sort_order,
    visible = EXCLUDED.visible,
    status = EXCLUDED.status,
    created_at = EXCLUDED.created_at,
    updated_at = EXCLUDED.updated_at;


-- 普通用户获得个人资料、退出登录及个人知识工作区接口；不授予系统管理 API。
INSERT INTO role_menus (role_id, menu_id)
SELECT roles.id, menus.id
FROM roles
         JOIN menus ON menus.code IN (
                                      'HOME_PAGE','SYSTEM_SPACE', 'GROUP_USERS', 'AUTH_LOGOUT', 'USER_PASSWORD_UPDATE',
                                      'USER_SELF_DETAIL', 'USER_SELF_UPDATE', 'USER_SELF_WEEKLY_STATISTICS',
                                      'WORK_SPACE', 'GROUP_NOTEBOOKS', 'NOTEBOOK_LIST', 'NOTEBOOK_CREATE',
                                      'NOTEBOOK_DETAIL', 'NOTEBOOK_UPDATE',
                                      'NOTEBOOK_SORT_UPDATE', 'NOTEBOOK_STATUS_UPDATE', 'NOTEBOOK_DELETE',
                                      'GROUP_NOTES', 'NOTE_LIST', 'NOTE_CREATE', 'NOTE_DETAIL', 'NOTE_UPDATE',
                                      'NOTE_MOVE', 'NOTE_DELETE', 'NOTE_RESTORE', 'NOTE_PERMANENT_DELETE',
                                      'GROUP_TRASH', 'TRASH_NOTE_LIST', 'TRASH_NOTE_RESTORE',
                                      'TRASH_NOTE_PERMANENT_DELETE',
                                      'GROUP_BOOKMARKS', 'BOOKMARK_LIST', 'BOOKMARK_CREATE', 'BOOKMARK_DETAIL',
                                      'BOOKMARK_UPDATE', 'BOOKMARK_DELETE',
                                      'GROUP_DRAFTS', 'DRAFT_LIST', 'DRAFT_SAVE', 'DRAFT_DETAIL', 'DRAFT_SUBMIT',
                                      'DRAFT_DELETE',
                                      'GROUP_FILES', 'FILE_UPLOAD', 'FILE_BASE64_IMAGE_UPLOAD', 'FILE_LIST',
                                      'FILE_DETAIL', 'FILE_DELETE', 'FILE_PREVIEW', 'FILE_DOWNLOAD'
    )
WHERE roles.code = 'USER';

-- 超级管理员获得本次初始化的全部菜单和 API 权限；接口鉴权仍由会话中的通配权限处理。
INSERT INTO role_menus (role_id, menu_id)
SELECT roles.id, menus.id
FROM roles
         CROSS JOIN menus
WHERE roles.code = 'SUPER_ADMIN'
  AND menus.id BETWEEN 11000 AND 12607;

-- 显式指定菜单 ID 后同步 identity 序列，避免后续自动生成的 ID 与初始化数据冲突。
SELECT setval(pg_get_serial_sequence('menus', 'id'), GREATEST((SELECT MAX(id) FROM menus), 1), TRUE);

COMMENT
ON TABLE users IS '用户表';
COMMENT
ON COLUMN users.id IS '用户主键';
COMMENT
ON COLUMN users.account IS '登录账号';
COMMENT
ON COLUMN users.name IS '用户名称';
COMMENT
ON COLUMN users.password IS '用户密码，仅保存密码哈希值，禁止保存明文';
COMMENT
ON COLUMN users.open_id IS '第三方平台 OpenID';
COMMENT
ON COLUMN users.sex IS '性别：0 未知，1 男，2 女';
COMMENT
ON COLUMN users.email IS '电子邮箱';
COMMENT
ON COLUMN users.phone IS '手机号码';
COMMENT
ON COLUMN users.avatar_url IS '头像地址';
COMMENT
ON COLUMN users.status IS '用户状态：1 正常，2 锁定，3 注销';
COMMENT
ON COLUMN users.last_login_at IS '最近登录时间';
COMMENT
ON COLUMN users.last_login_ip IS '最近登录 IP';
COMMENT
ON COLUMN users.login_method IS '登录方式';
COMMENT
ON COLUMN users.created_at IS '创建时间';
COMMENT
ON COLUMN users.updated_at IS '最后更新时间';

COMMENT
ON TABLE roles IS '角色表';
COMMENT
ON COLUMN roles.id IS '角色主键';
COMMENT
ON COLUMN roles.name IS '角色名称';
COMMENT
ON COLUMN roles.code IS '角色编码';
COMMENT
ON COLUMN roles.description IS '角色描述';
COMMENT
ON COLUMN roles.status IS '角色状态：1 启用，2 禁用';
COMMENT
ON COLUMN roles.created_at IS '创建时间';
COMMENT
ON COLUMN roles.updated_at IS '最后更新时间';

COMMENT
ON TABLE menus IS '菜单和权限资源表';
COMMENT
ON COLUMN menus.id IS '菜单权限主键';
COMMENT
ON COLUMN menus.parent_id IS '父级菜单 ID';
COMMENT
ON COLUMN menus.name IS '菜单或权限名称';
COMMENT
ON COLUMN menus.code IS '权限编码';
COMMENT
ON COLUMN menus.type IS '资源类型：1 目录，2 页面，3 按钮，4 API';
COMMENT
ON COLUMN menus.route_path IS '前端路由地址';
COMMENT
ON COLUMN menus.component IS '前端组件路径';
COMMENT
ON COLUMN menus.http_method IS '接口 HTTP 方法，API 类型资源必填';
COMMENT
ON COLUMN menus.api_path IS 'Gin 路由模板，API 类型资源必填，例如 /api/v1/users/:id';
COMMENT
ON COLUMN menus.sort_order IS '同级排序值';
COMMENT
ON COLUMN menus.visible IS '是否在菜单中显示';
COMMENT
ON COLUMN menus.status IS '资源状态：1 启用，2 禁用';
COMMENT
ON COLUMN menus.created_at IS '创建时间';
COMMENT
ON COLUMN menus.updated_at IS '最后更新时间';

COMMENT
ON TABLE user_roles IS '用户角色关联表';
COMMENT
ON COLUMN user_roles.user_id IS '用户 ID';
COMMENT
ON COLUMN user_roles.role_id IS '角色 ID';
COMMENT
ON COLUMN user_roles.created_at IS '授权时间';

COMMENT
ON TABLE role_menus IS '角色菜单权限关联表';
COMMENT
ON COLUMN role_menus.role_id IS '角色 ID';
COMMENT
ON COLUMN role_menus.menu_id IS '菜单或权限资源 ID';
COMMENT
ON COLUMN role_menus.created_at IS '授权时间';

COMMENT
ON TABLE notebooks IS '笔记簿表';
COMMENT
ON COLUMN notebooks.id IS '笔记簿主键';
COMMENT
ON COLUMN notebooks.owner_id IS '所属用户 ID';
COMMENT
ON COLUMN notebooks.name IS '笔记簿名';
COMMENT
ON COLUMN notebooks.icon IS '封面图标名';
COMMENT
ON COLUMN notebooks.tag IS '笔记簿标签';
COMMENT
ON COLUMN notebooks.description IS '笔记簿描述';
COMMENT
ON COLUMN notebooks.status IS '笔记簿状态：1 正常，2 禁用，3 删除';
COMMENT
ON COLUMN notebooks.sort_order IS '笔记簿排序值';
COMMENT
ON COLUMN notebooks.created_at IS '创建时间';
COMMENT
ON COLUMN notebooks.updated_at IS '最后更新时间';
COMMENT
ON COLUMN notebooks.deleted_at IS '删除时间';

COMMENT
ON TABLE notes IS '笔记表';
COMMENT
ON COLUMN notes.id IS '笔记主键';
COMMENT
ON COLUMN notes.owner_id IS '所属用户 ID';
COMMENT
ON COLUMN notes.notebook_id IS '所属笔记簿 ID';
COMMENT
ON COLUMN notes.title IS '笔记标题';
COMMENT
ON COLUMN notes.label IS '笔记标签';
COMMENT
ON COLUMN notes.summary IS '笔记摘要';
COMMENT
ON COLUMN notes.cover_url IS '笔记封面地址';
COMMENT
ON COLUMN notes.content IS '笔记正文内容';
COMMENT
ON COLUMN notes.content_uri IS '正文外部存储地址';
COMMENT
ON COLUMN notes.content_type IS '正文格式，例如 markdown 或 html';
COMMENT
ON COLUMN notes.status IS '笔记状态：1 正常，2 回收站';
COMMENT
ON COLUMN notes.version IS '乐观锁版本号';
COMMENT
ON COLUMN notes.deleted_at IS '进入回收站时间';
COMMENT
ON COLUMN notes.deleted_by IS '执行删除的用户 ID';
COMMENT
ON COLUMN notes.purge_at IS '计划永久清理时间';
COMMENT
ON COLUMN notes.created_at IS '创建时间';
COMMENT
ON COLUMN notes.updated_at IS '最后更新时间';

COMMENT
ON TABLE drafts IS '笔记草稿表';
COMMENT
ON COLUMN drafts.id IS '草稿主键';
COMMENT
ON COLUMN drafts.owner_id IS '所属用户 ID';
COMMENT
ON COLUMN drafts.draft_key IS '客户端草稿业务标识，用于幂等保存';
COMMENT
ON COLUMN drafts.note_id IS '对应的正式笔记 ID，新建笔记时为空';
COMMENT
ON COLUMN drafts.notebook_id IS '目标笔记簿 ID';
COMMENT
ON COLUMN drafts.title IS '草稿标题';
COMMENT
ON COLUMN drafts.label IS '草稿标签';
COMMENT
ON COLUMN drafts.content IS '草稿正文内容';
COMMENT
ON COLUMN drafts.content_uri IS '草稿正文外部存储地址';
COMMENT
ON COLUMN drafts.content_type IS '正文格式，例如 markdown 或 html';
COMMENT
ON COLUMN drafts.version IS '草稿乐观锁版本号';
COMMENT
ON COLUMN drafts.expires_at IS '草稿过期时间';
COMMENT
ON COLUMN drafts.created_at IS '创建时间';
COMMENT
ON COLUMN drafts.updated_at IS '最后更新时间';

COMMENT
ON TABLE bookmarks IS '收藏夹表';
COMMENT
ON COLUMN bookmarks.id IS '收藏主键';
COMMENT
ON COLUMN bookmarks.owner_id IS '所属用户 ID';
COMMENT
ON COLUMN bookmarks.topic IS '收藏标题';
COMMENT
ON COLUMN bookmarks.label IS '收藏标签';
COMMENT
ON COLUMN bookmarks.href IS '收藏链接地址';
COMMENT
ON COLUMN bookmarks.icon IS '网站图标地址';
COMMENT
ON COLUMN bookmarks.favorite IS '是否标记为重点收藏';
COMMENT
ON COLUMN bookmarks.description IS '收藏描述';
COMMENT
ON COLUMN bookmarks.created_at IS '创建时间';
COMMENT
ON COLUMN bookmarks.updated_at IS '最后更新时间';

COMMENT
ON TABLE files IS '文件和附件元数据表';
COMMENT
ON COLUMN files.id IS '文件主键';
COMMENT
ON COLUMN files.owner_id IS '所属用户 ID';
COMMENT
ON COLUMN files.file_name IS '原始文件名';
COMMENT
ON COLUMN files.object_key IS '文件存储对象键';
COMMENT
ON COLUMN files.public_key IS '文件公开访问标识';
COMMENT
ON COLUMN files.file_url IS '文件访问地址';
COMMENT
ON COLUMN files.file_type IS '文件类型或 MIME 类型';
COMMENT
ON COLUMN files.file_size IS '文件大小，单位为字节';
COMMENT
ON COLUMN files.checksum IS '文件校验和';
COMMENT
ON COLUMN files.status IS '文件状态：1 正常，2 删除';
COMMENT
ON COLUMN files.created_at IS '上传时间';
COMMENT
ON COLUMN files.updated_at IS '最后更新时间';
COMMENT
ON COLUMN files.deleted_at IS '删除时间';

COMMENT
ON TABLE audit_logs IS '业务审计日志表';
COMMENT
ON COLUMN audit_logs.id IS '审计日志主键';
COMMENT
ON COLUMN audit_logs.operator_account IS '操作用户账号快照，用于记录谁执行了操作';
COMMENT
ON COLUMN audit_logs.operation IS '执行的操作内容';
COMMENT
ON COLUMN audit_logs.ip IS '操作发生的 IP 地址';
COMMENT
ON COLUMN audit_logs.location IS '操作发生的地点';
COMMENT
ON COLUMN audit_logs.operated_at IS '操作发生时间';
