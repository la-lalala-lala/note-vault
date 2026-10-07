package storage

import (
	"context"
	"io"
)

// StoredObject 是对象存储保存文件后返回的文件元数据。
type StoredObject struct {
	URL      string
	Size     int64
	Checksum string
}

// ObjectStorage 抽象文件对象的保存和删除操作。
// Service 只依赖该接口，不依赖本地磁盘、S3 或其他具体存储实现。
type ObjectStorage interface {
	Save(ctx context.Context, objectKey string, source io.Reader) (StoredObject, error)
	Open(ctx context.Context, objectKey string) (io.ReadCloser, error)
	Delete(ctx context.Context, objectKey string) error
}
