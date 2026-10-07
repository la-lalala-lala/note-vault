package service

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"note-vault/app/config"
	"note-vault/app/dto"
	"note-vault/app/model"
	"note-vault/app/repository"
	"note-vault/app/storage"
	"note-vault/app/utils"
)

const (
	maxFileNameLength = 255
	objectKeyBytes    = 24
	publicKeyBytes    = 32
)

// FileService 负责文件上传校验、数据归属和文件对象与元数据的一致性流程。
// 文件对象的具体保存方式通过 ObjectStorage 接口注入。
type FileService struct {
	files         *repository.FileRepository
	objectStorage storage.ObjectStorage
}

// NewFileService 创建文件服务；可选传入对象存储实现，未传入时按当前配置延迟创建。
func NewFileService(objectStorages ...storage.ObjectStorage) *FileService {
	fileService := &FileService{files: repository.NewFileRepository()}
	if len(objectStorages) > 0 {
		fileService.objectStorage = objectStorages[0]
	}
	return fileService
}

// UploadFile 上传当前用户的文件，并保存文件元数据。
func (s *FileService) UploadFile(ctx context.Context, ownerID uint, header *multipart.FileHeader) (*dto.FileResponse, error) {
	if header == nil {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "上传文件不能为空")
	}
	file, err := header.Open()
	if err != nil {
		return nil, utils.WrapAppError(utils.KindInternal, "打开上传文件失败", err)
	}
	defer file.Close()
	return s.uploadContent(ctx, ownerID, header.Filename, file, header.Size, false, "")
}

// UploadBase64Image 解码并上传 Base64 图片，文件内容仍经过与普通上传相同的校验和存储流程。
func (s *FileService) UploadBase64Image(ctx context.Context, ownerID uint, input dto.Base64ImageRequest) (*dto.FileResponse, error) {
	storageConfig := config.AppConfig.Storage
	if storageConfig.MaxFileSize <= 0 {
		return nil, utils.NewAppError(utils.KindInternal, "文件存储配置无效")
	}
	fileName := normalizeFileName(input.FileName)
	if fileName == "" {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "文件名不能为空")
	}
	if utf8.RuneCountInString(fileName) > maxFileNameLength {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "文件名长度不能大于 255")
	}
	if !storageConfig.AllowsExtension(strings.ToLower(filepath.Ext(fileName))) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "文件扩展名不被允许")
	}

	encoded, declaredMIME, err := splitBase64Image(input.Base64)
	if err != nil {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "Base64 图片数据格式无效")
	}
	maxEncodedLength := ((storageConfig.MaxFileSize + 2) / 3) * 4
	if int64(len(encoded)) > maxEncodedLength {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "文件大小超过允许的最大值")
	}
	if declaredMIME != "" && !storageConfig.AllowsMIMEType(declaredMIME) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "文件 MIME 类型不被允许")
	}

	imageBytes, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil {
		imageBytes, err = base64.RawStdEncoding.Strict().DecodeString(encoded)
		if err != nil {
			return nil, utils.NewAppError(utils.KindInvalidArgument, "Base64 图片数据格式无效")
		}
	}
	if int64(len(imageBytes)) > storageConfig.MaxFileSize {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "文件大小超过允许的最大值")
	}
	return s.uploadContent(ctx, ownerID, fileName, bytes.NewReader(imageBytes), int64(len(imageBytes)), true, declaredMIME)
}

func (s *FileService) uploadContent(ctx context.Context, ownerID uint, originalName string, source io.Reader, declaredSize int64, imageOnly bool, declaredMIME string) (*dto.FileResponse, error) {
	storageConfig := config.AppConfig.Storage
	if storageConfig.MaxFileSize <= 0 {
		return nil, utils.NewAppError(utils.KindInternal, "文件存储配置无效")
	}
	fileName := normalizeFileName(originalName)
	if fileName == "" {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "文件名不能为空")
	}
	if utf8.RuneCountInString(fileName) > maxFileNameLength {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "文件名长度不能大于 255")
	}
	if !storageConfig.AllowsExtension(strings.ToLower(filepath.Ext(fileName))) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "文件扩展名不被允许")
	}
	if declaredSize > storageConfig.MaxFileSize {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "文件大小超过允许的最大值")
	}

	sample := make([]byte, 512)
	readSize, readErr := io.ReadFull(source, sample)
	if errors.Is(readErr, io.ErrUnexpectedEOF) || errors.Is(readErr, io.EOF) {
		readErr = nil
	}
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return nil, utils.WrapAppError(utils.KindInternal, "读取上传文件失败", readErr)
	}
	detectedType := normalizeMIMEType(http.DetectContentType(sample[:readSize]))
	if !storageConfig.AllowsMIMEType(detectedType) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "文件 MIME 类型不被允许")
	}
	if imageOnly && !strings.HasPrefix(detectedType, "image/") {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "上传内容不是有效图片")
	}
	if imageOnly && !imageExtensionMatchesMIME(filepath.Ext(fileName), detectedType) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "图片扩展名与实际内容不匹配")
	}
	if declaredMIME != "" && declaredMIME != detectedType {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "图片声明类型与实际内容不一致")
	}

	objectKey, err := generateObjectKey()
	if err != nil {
		return nil, utils.WrapAppError(utils.KindInternal, "生成文件对象键失败", err)
	}
	publicKey, err := generatePublicKey()
	if err != nil {
		return nil, utils.WrapAppError(utils.KindInternal, "生成文件公开访问标识失败", err)
	}
	objectStorage, err := s.getObjectStorage()
	if err != nil {
		return nil, utils.WrapAppError(utils.KindInternal, "初始化文件存储失败", err)
	}

	content := io.MultiReader(bytes.NewReader(sample[:readSize]), source)
	storedObject, err := objectStorage.Save(
		ctx,
		objectKey,
		io.LimitReader(content, storageConfig.MaxFileSize+1),
	)
	if err != nil {
		return nil, utils.WrapAppError(utils.KindInternal, "保存文件对象失败", err)
	}
	if storedObject.Size > storageConfig.MaxFileSize {
		cleanupErr := objectStorage.Delete(ctx, objectKey)
		invalidFileSizeError := utils.NewAppError(utils.KindInvalidArgument, "文件大小超过允许的最大值")
		if cleanupErr != nil {
			return nil, utils.WrapAppError(utils.KindInvalidArgument, invalidFileSizeError.Message, cleanupErr)
		}
		return nil, invalidFileSizeError
	}

	fileType := detectedType
	checksum := storedObject.Checksum
	fileModel := model.File{
		OwnerID: ownerID, FileName: fileName, ObjectKey: objectKey, PublicKey: publicKey,
		FileType: &fileType, FileSize: storedObject.Size, Checksum: &checksum,
		Status: model.FileStatusNormal,
	}
	if storedObject.URL != "" {
		fileURL := storedObject.URL
		fileModel.FileURL = &fileURL
	}

	if err := s.files.Create(ctx, &fileModel); err != nil {
		cleanupErr := objectStorage.Delete(ctx, objectKey)
		if cleanupErr != nil {
			return nil, utils.WrapAppError(utils.KindInternal, "创建文件元数据失败，且清理文件对象失败", errors.Join(utils.BuildFileRepositoryError(err, "创建文件元数据失败"), cleanupErr))
		}
		return nil, utils.BuildFileRepositoryError(err, "创建文件元数据失败")
	}

	// 普通文件和 Base64 图片最终均通过本方法成功落库，在此统一记录上传审计日志。
	auditRecorder.Record(ctx, fmt.Sprintf("上传文件《%s》（ID %d）", fileModel.FileName, fileModel.ID))
	response := s.newFileResponse(fileModel)
	return &response, nil
}

// ListFiles 分页查询当前用户自己的正常文件。
func (s *FileService) ListFiles(ctx context.Context, ownerID uint, input dto.FileRequest) (*dto.PageResponse[dto.FileResponse], error) {
	page, pageSize, err := utils.NormalizePage(input.Page, input.PageSize)
	if err != nil {
		return nil, err
	}
	input.Page = &page
	input.PageSize = &pageSize
	files, total, err := s.files.List(ctx, ownerID, input)
	if err != nil {
		return nil, utils.BuildFileRepositoryError(err, "查询文件列表失败")
	}
	responses := make([]dto.FileResponse, 0, len(files))
	for _, file := range files {
		responses = append(responses, s.newFileResponse(file))
	}
	return &dto.PageResponse[dto.FileResponse]{List: responses, Page: page, PageSize: pageSize, Total: total}, nil
}

// GetFile 查询当前用户自己的正常文件详情。
func (s *FileService) GetFile(ctx context.Context, ownerID, id uint) (*dto.FileResponse, error) {
	file, err := s.files.FindByID(ctx, ownerID, id)
	if err != nil {
		return nil, utils.BuildFileRepositoryError(err, "查询文件详情失败")
	}
	response := s.newFileResponse(*file)
	return &response, nil
}

// OpenFile 打开当前用户自己的文件对象，供下载接口返回文件内容。
func (s *FileService) OpenFile(ctx context.Context, ownerID, id uint) (*model.File, io.ReadCloser, error) {
	file, err := s.files.FindByID(ctx, ownerID, id)
	if err != nil {
		return nil, nil, utils.BuildFileRepositoryError(err, "查询下载文件失败")
	}
	objectStorage, err := s.getObjectStorage()
	if err != nil {
		return nil, nil, utils.WrapAppError(utils.KindInternal, "初始化文件存储失败", err)
	}
	reader, err := objectStorage.Open(ctx, file.ObjectKey)
	if err != nil {
		return nil, nil, utils.WrapAppError(utils.KindNotFound, "打开下载文件失败", err)
	}
	return file, reader, nil
}

// OpenPublicFile 打开正常文件对象，供无需登录的预览接口返回文件内容。
func (s *FileService) OpenPublicFile(ctx context.Context, publicKey string) (*model.File, io.ReadCloser, error) {
	file, err := s.files.FindPublicByKey(ctx, publicKey)
	if err != nil {
		return nil, nil, utils.BuildFileRepositoryError(err, "查询公开预览文件失败")
	}
	objectStorage, err := s.getObjectStorage()
	if err != nil {
		return nil, nil, utils.WrapAppError(utils.KindInternal, "初始化文件存储失败", err)
	}
	reader, err := objectStorage.Open(ctx, file.ObjectKey)
	if err != nil {
		return nil, nil, utils.WrapAppError(utils.KindNotFound, "打开公开预览文件失败", err)
	}
	return file, reader, nil
}

// DeleteFile 删除文件对象并将文件元数据标记为已删除。
func (s *FileService) DeleteFile(ctx context.Context, ownerID, id uint) error {
	file, err := s.files.FindByID(ctx, ownerID, id)
	if err != nil {
		return utils.BuildFileRepositoryError(err, "查询待删除文件失败")
	}
	objectStorage, err := s.getObjectStorage()
	if err != nil {
		return utils.WrapAppError(utils.KindInternal, "初始化文件存储失败", err)
	}
	if err := objectStorage.Delete(ctx, file.ObjectKey); err != nil {
		return utils.WrapAppError(utils.KindInternal, "删除文件对象失败", err)
	}
	if err := s.files.SoftDelete(ctx, ownerID, id, time.Now().UTC()); err != nil {
		return utils.BuildFileRepositoryError(err, "删除文件元数据失败")
	}
	// 文件对象删除且元数据成功标记删除后，才记录删除审计日志。
	auditRecorder.Record(ctx, fmt.Sprintf("删除文件《%s》（ID %d）", file.FileName, file.ID))
	return nil
}

func (s *FileService) getObjectStorage() (storage.ObjectStorage, error) {
	if s.objectStorage != nil {
		return s.objectStorage, nil
	}
	return storage.NewFromConfig(config.AppConfig.Storage)
}

func (s *FileService) newFileResponse(file model.File) dto.FileResponse {
	response := dto.NewFileResponse(file)
	fileURL := fmt.Sprintf("/api/v1/files/public/%s/preview", file.PublicKey)
	response.FileURL = &fileURL
	return response
}

func normalizeFileName(fileName string) string {
	fileName = strings.TrimSpace(strings.ReplaceAll(fileName, "\\", "/"))
	if fileName == "" {
		return ""
	}
	fileName = filepath.Base(fileName)
	if fileName == "." || fileName == string(filepath.Separator) {
		return ""
	}
	return fileName
}

func normalizeMIMEType(value string) string {
	mediaType, _, err := mime.ParseMediaType(value)
	if err == nil {
		return strings.ToLower(mediaType)
	}
	return strings.ToLower(strings.TrimSpace(strings.SplitN(value, ";", 2)[0]))
}

// splitBase64Image 拆分纯 Base64 或 data URL 格式的图片内容。
func splitBase64Image(value string) (string, string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", errors.New("base64 image is empty")
	}

	declaredMIME := ""
	encoded := value
	if strings.HasPrefix(strings.ToLower(value), "data:") {
		separator := strings.IndexByte(value, ',')
		if separator < 0 {
			return "", "", errors.New("invalid data URL")
		}
		metadata := value[len("data:"):separator]
		metadataParts := strings.Split(metadata, ";")
		if len(metadataParts) < 2 || !strings.EqualFold(strings.TrimSpace(metadataParts[len(metadataParts)-1]), "base64") {
			return "", "", errors.New("data URL must use base64 encoding")
		}
		declaredMIME = normalizeMIMEType(metadataParts[0])
		if declaredMIME == "" {
			return "", "", errors.New("data URL MIME type is empty")
		}
		encoded = value[separator+1:]
	}

	encoded = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(encoded, "\r", ""), "\n", ""))
	if encoded == "" {
		return "", "", errors.New("base64 image is empty")
	}
	return encoded, declaredMIME, nil
}

func imageExtensionMatchesMIME(extension, mimeType string) bool {
	extension = strings.ToLower(strings.TrimSpace(extension))
	allowedExtensions, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return false
	}
	for _, allowedExtension := range allowedExtensions {
		if extension == strings.ToLower(allowedExtension) {
			return true
		}
	}
	return false
}

func generateObjectKey() (string, error) {
	randomBytes := make([]byte, objectKeyBytes)
	if _, err := cryptorand.Read(randomBytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(randomBytes), nil
}

func generatePublicKey() (string, error) {
	randomBytes := make([]byte, publicKeyBytes)
	if _, err := cryptorand.Read(randomBytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(randomBytes), nil
}
