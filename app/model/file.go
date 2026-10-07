package model

import "time"

const (
	FileStatusNormal  = 1
	FileStatusDeleted = 2
)

// File 对应文件表（files），字段含义与数据库注释保持一致。
// HTTP 请求和响应使用 dto 包中的结构，禁止直接序列化数据库 Model。
type File struct {
	ID        uint       `gorm:"column:id;primaryKey" json:"-"`                 // 文件主键
	OwnerID   uint       `gorm:"column:owner_id;not null" json:"-"`             // 所属用户 ID
	FileName  string     `gorm:"column:file_name;size:255;not null" json:"-"`   // 原始文件名
	ObjectKey string     `gorm:"column:object_key;size:1000;not null" json:"-"` // 文件存储对象键
	PublicKey string     `gorm:"column:public_key;size:64;not null" json:"-"`   // 文件公开访问标识
	FileURL   *string    `gorm:"column:file_url;size:1500" json:"-"`            // 文件访问地址
	FileType  *string    `gorm:"column:file_type;size:100" json:"-"`            // 文件类型或 MIME 类型
	FileSize  int64      `gorm:"column:file_size;not null;default:0" json:"-"`  // 文件大小，单位为字节
	Checksum  *string    `gorm:"column:checksum;size:128" json:"-"`             // 文件校验和
	Status    int16      `gorm:"column:status;not null;default:1" json:"-"`     // 文件状态：1 正常，2 删除
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime" json:"-"`     // 上传时间
	UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"-"`     // 最后更新时间
	DeletedAt *time.Time `gorm:"column:deleted_at" json:"-"`                    // 删除时间
}

// TableName 明确 File 使用的数据库表名。
func (File) TableName() string {
	return "files"
}
