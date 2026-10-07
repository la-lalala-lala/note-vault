package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"note-vault/app/config"
)

// LocalStorage 将文件对象保存到本地文件系统。
type LocalStorage struct {
	basePath string
}

// NewLocalStorage 创建本地文件存储，并确保根目录存在。
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	basePath = strings.TrimSpace(basePath)
	if basePath == "" {
		return nil, errors.New("local storage base path is empty")
	}
	absolutePath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("resolve local storage base path: %w", err)
	}
	if err := os.MkdirAll(absolutePath, 0750); err != nil {
		return nil, fmt.Errorf("create local storage base path: %w", err)
	}
	return &LocalStorage{basePath: absolutePath}, nil
}

// NewFromConfig 根据配置创建当前启用的对象存储实现。
// 当前仅实现 local，后续增加对象存储实现时只需扩展这里的工厂分支。
func NewFromConfig(storageConfig config.StorageConfig) (ObjectStorage, error) {
	switch strings.ToLower(strings.TrimSpace(storageConfig.Type)) {
	case "local":
		return NewLocalStorage(storageConfig.BasePath)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageConfig.Type)
	}
}

// Save 将输入流写入临时文件，计算 SHA-256 后原子移动到目标对象键。
// objectKey 由业务层随机生成，客户端文件名不会参与路径拼接。
func (s *LocalStorage) Save(ctx context.Context, objectKey string, source io.Reader) (StoredObject, error) {
	path, err := s.objectPath(objectKey)
	if err != nil {
		return StoredObject{}, err
	}
	if err := contextError(ctx); err != nil {
		return StoredObject{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return StoredObject{}, fmt.Errorf("create object directory: %w", err)
	}

	temporaryFile, err := os.CreateTemp(filepath.Dir(path), ".upload-*")
	if err != nil {
		return StoredObject{}, fmt.Errorf("create temporary object: %w", err)
	}
	temporaryPath := temporaryFile.Name()
	keepTemporaryFile := false
	defer func() {
		_ = temporaryFile.Close()
		if !keepTemporaryFile {
			_ = os.Remove(temporaryPath)
		}
	}()

	digest := sha256.New()
	bytesWritten, err := io.Copy(io.MultiWriter(temporaryFile, digest), contextReader{ctx: ctx, reader: source})
	if err != nil {
		return StoredObject{}, fmt.Errorf("write object: %w", err)
	}
	if err := contextError(ctx); err != nil {
		return StoredObject{}, err
	}
	if err := temporaryFile.Sync(); err != nil {
		return StoredObject{}, fmt.Errorf("sync temporary object: %w", err)
	}
	if err := temporaryFile.Close(); err != nil {
		return StoredObject{}, fmt.Errorf("close temporary object: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return StoredObject{}, fmt.Errorf("move object into storage: %w", err)
	}
	keepTemporaryFile = true

	return StoredObject{
		Size:     bytesWritten,
		Checksum: hex.EncodeToString(digest.Sum(nil)),
	}, nil
}

// Open 打开已保存的文件对象，供预览和下载接口读取。
func (s *LocalStorage) Open(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	path, err := s.objectPath(objectKey)
	if err != nil {
		return nil, err
	}
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open object: %w", err)
	}
	return file, nil
}

// Delete 删除对象；对象已经不存在时按幂等成功处理。
func (s *LocalStorage) Delete(ctx context.Context, objectKey string) error {
	path, err := s.objectPath(objectKey)
	if err != nil {
		return err
	}
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

func (s *LocalStorage) objectPath(objectKey string) (string, error) {
	if s == nil || strings.TrimSpace(s.basePath) == "" {
		return "", errors.New("local storage is unavailable")
	}
	if strings.TrimSpace(objectKey) == "" {
		return "", errors.New("object key is empty")
	}

	cleanKey := filepath.Clean(filepath.FromSlash(objectKey))
	if cleanKey == "." || filepath.IsAbs(cleanKey) || cleanKey == ".." || strings.HasPrefix(cleanKey, ".."+string(os.PathSeparator)) {
		return "", errors.New("object key contains an invalid path")
	}
	return filepath.Join(s.basePath, cleanKey), nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(buffer []byte) (int, error) {
	if err := contextError(r.ctx); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
