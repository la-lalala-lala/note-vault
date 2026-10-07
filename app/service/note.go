package service

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"note-vault/app/config"
	"note-vault/app/dto"
	"note-vault/app/model"
	"note-vault/app/repository"
	"note-vault/app/utils"
)

const (
	maxNoteTitleLength        = 200
	noteTrashCleanupBatchSize = 100
)

// NoteService 负责笔记业务规则、数据归属校验、乐观锁和 DTO/Model 转换。
type NoteService struct {
	notes     *repository.NoteRepository
	notebooks *repository.NotebookRepository
}

func NewNoteService() *NoteService {
	return &NoteService{
		notes:     repository.NewNoteRepository(),
		notebooks: repository.NewNotebookRepository(),
	}
}

func (s *NoteService) CreateNote(ctx context.Context, ownerID uint, input dto.NoteRequest) (*dto.NoteResponse, error) {
	if input.NotebookID == nil || *input.NotebookID == 0 {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "笔记簿 ID 不能为空")
	}
	title, err := utils.RequiredText(input.Title, "笔记标题")
	if err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(title) > maxNoteTitleLength {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "笔记标题长度不能大于 200")
	}
	*input.Title = title
	if err := utils.ValidNoteOptionalFields(&input); err != nil {
		return nil, err
	}
	if err := s.validateTargetNotebook(ctx, ownerID, *input.NotebookID); err != nil {
		return nil, err
	}

	contentType := "markdown"
	if input.ContentType != nil {
		contentType = *input.ContentType
	}
	note := model.Note{
		OwnerID: ownerID, NotebookID: *input.NotebookID, Title: title,
		Label: input.Label, Summary: input.Summary, CoverURL: input.CoverURL,
		Content: input.Content, ContentType: contentType,
		Status: model.NoteStatusNormal, Version: 1,
	}
	if err := s.notes.Create(ctx, &note); err != nil {
		return nil, utils.BuildNoteRepositoryError(err, "创建笔记失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("新增笔记《%s》（ID %d）", note.Title, note.ID))
	response := dto.NewNoteResponse(note)
	return &response, nil
}

func (s *NoteService) ListNotes(ctx context.Context, ownerID uint, input dto.NoteRequest) (*dto.PageResponse[dto.NoteResponse], error) {
	page, pageSize, err := utils.NormalizePage(input.Page, input.PageSize)
	if err != nil {
		return nil, err
	}
	if input.UpdatedFrom != nil && input.UpdatedTo != nil && input.UpdatedFrom.After(*input.UpdatedTo) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "更新时间起始值不能晚于结束值")
	}
	input.Page = &page
	input.PageSize = &pageSize
	notes, total, err := s.notes.List(ctx, ownerID, input)
	if err != nil {
		return nil, utils.BuildNoteRepositoryError(err, "查询笔记列表失败")
	}
	return &dto.PageResponse[dto.NoteResponse]{
		List: dto.NewNoteResponses(notes), Page: page, PageSize: pageSize, Total: total,
	}, nil
}

func (s *NoteService) GetNote(ctx context.Context, ownerID, id uint) (*dto.NoteResponse, error) {
	note, err := s.notes.FindByID(ctx, ownerID, id)
	if err != nil {
		return nil, utils.BuildNoteRepositoryError(err, "查询笔记详情失败")
	}
	response := dto.NewNoteResponse(*note)
	return &response, nil
}

func (s *NoteService) UpdateNote(ctx context.Context, ownerID, id uint, input dto.NoteRequest) (*dto.NoteResponse, error) {
	if input.Version == nil || *input.Version == 0 {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "笔记版本号不能为空")
	}
	input.NotebookID = nil
	if input.Title == nil && input.Label == nil && input.Summary == nil && input.CoverURL == nil && input.Content == nil && input.ContentType == nil {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "至少需要提交一个可修改字段")
	}
	if input.Title != nil {
		title, err := utils.RequiredText(input.Title, "笔记标题")
		if err != nil {
			return nil, err
		}
		if utf8.RuneCountInString(title) > maxNoteTitleLength {
			return nil, utils.NewAppError(utils.KindInvalidArgument, "笔记标题长度不能大于 200")
		}
		*input.Title = title
	}
	if err := utils.ValidNoteOptionalFields(&input); err != nil {
		return nil, err
	}
	if err := s.notes.Update(ctx, ownerID, id, *input.Version, input); err != nil {
		return nil, utils.BuildNoteRepositoryError(err, "更新笔记失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("修改笔记 ID %d", id))
	return s.GetNote(ctx, ownerID, id)
}

func (s *NoteService) MoveNote(ctx context.Context, ownerID, id uint, input dto.NoteRequest) (*dto.NoteResponse, error) {
	if input.NotebookID == nil || *input.NotebookID == 0 {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "目标笔记簿 ID 不能为空")
	}
	if input.Version == nil || *input.Version == 0 {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "笔记版本号不能为空")
	}
	if err := s.validateTargetNotebook(ctx, ownerID, *input.NotebookID); err != nil {
		return nil, err
	}
	moveInput := dto.NoteRequest{NotebookID: input.NotebookID}
	if err := s.notes.Update(ctx, ownerID, id, *input.Version, moveInput); err != nil {
		return nil, utils.BuildNoteRepositoryError(err, "移动笔记失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("将笔记 ID %d 移动到笔记簿 ID %d", id, *input.NotebookID))
	return s.GetNote(ctx, ownerID, id)
}

func (s *NoteService) DeleteNote(ctx context.Context, ownerID, id uint) error {
	deletedAt := time.Now().UTC()
	retentionDays := config.AppConfig.Trash.RetentionDays
	if retentionDays <= 0 {
		retentionDays = 30
	}
	purgeAt := deletedAt.AddDate(0, 0, retentionDays)
	if err := s.notes.MoveToTrash(ctx, ownerID, id, ownerID, deletedAt, purgeAt); err != nil {
		return utils.BuildNoteRepositoryError(err, "将笔记移入回收站失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("将笔记 ID %d 移入回收站", id))
	return nil
}

func (s *NoteService) ListTrashNotes(ctx context.Context, ownerID uint, input dto.NoteRequest) (*dto.PageResponse[dto.NoteResponse], error) {
	page, pageSize, err := utils.NormalizePage(input.Page, input.PageSize)
	if err != nil {
		return nil, err
	}
	notes, total, err := s.notes.ListTrash(ctx, ownerID, page, pageSize)
	if err != nil {
		return nil, utils.BuildNoteRepositoryError(err, "查询回收站笔记失败")
	}
	responses := dto.NewNoteResponses(notes)
	now := time.Now().UTC()
	for index := range responses {
		if responses[index].PurgeAt == nil {
			continue
		}
		remaining := responses[index].PurgeAt.Sub(now)
		remainingDays := 0
		if remaining > 0 {
			remainingDays = int((remaining + 24*time.Hour - 1) / (24 * time.Hour))
		}
		responses[index].RemainingRetentionDays = &remainingDays
	}
	return &dto.PageResponse[dto.NoteResponse]{
		List: responses, Page: page, PageSize: pageSize, Total: total,
	}, nil
}

func (s *NoteService) RestoreNote(ctx context.Context, ownerID, id uint, input dto.NoteRequest) (*dto.NoteResponse, error) {
	trashedNote, err := s.notes.FindTrashedByID(ctx, ownerID, id)
	if err != nil {
		return nil, utils.BuildNoteRepositoryError(err, "查询待恢复笔记失败")
	}
	targetNotebookID := trashedNote.NotebookID
	if input.NotebookID != nil {
		targetNotebookID = *input.NotebookID
	} else if err := s.validateTargetNotebook(ctx, ownerID, targetNotebookID); err != nil {
		var applicationError *utils.AppError
		if !errors.As(err, &applicationError) || applicationError.Kind == utils.KindInternal {
			return nil, err
		}
		return nil, utils.NewAppError(utils.KindInvalidArgument, "原笔记簿不可用，请指定新的正常笔记簿")
	}
	if input.NotebookID != nil {
		if err := s.validateTargetNotebook(ctx, ownerID, targetNotebookID); err != nil {
			return nil, err
		}
	}
	if err := s.notes.Restore(ctx, ownerID, id, targetNotebookID); err != nil {
		return nil, utils.BuildNoteRepositoryError(err, "恢复笔记失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("恢复笔记 ID %d", id))
	return s.GetNote(ctx, ownerID, id)
}

func (s *NoteService) PermanentlyDeleteNote(ctx context.Context, ownerID, id uint) error {
	if err := s.notes.PermanentDelete(ctx, ownerID, id); err != nil {
		return utils.BuildNoteRepositoryError(err, "永久删除回收站笔记失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("永久删除笔记 ID %d", id))
	return nil
}

// CleanupExpiredTrash 清理到期的回收站笔记，并返回本次任务的处理统计。
func (s *NoteService) CleanupExpiredTrash(ctx context.Context) (int, int, int, error) {
	scanned, succeeded, failed, err := s.notes.CleanupExpiredTrash(
		ctx,
		time.Now().UTC(),
		noteTrashCleanupBatchSize,
	)
	if succeeded > 0 {
		auditContext := ctx
		if _, ok := utils.CurrentAuditActor(ctx); !ok {
			auditContext = utils.WithAuditActor(ctx, "SYSTEM", "")
		}
		auditRecorder.Record(auditContext, fmt.Sprintf("自动永久删除 %d 条过期回收站笔记", succeeded))
	}
	if err != nil {
		return scanned, succeeded, failed, utils.BuildNoteRepositoryError(err, "清理到期回收站笔记失败")
	}
	return scanned, succeeded, failed, nil
}

func (s *NoteService) validateTargetNotebook(ctx context.Context, ownerID, notebookID uint) error {
	notebook, err := s.notebooks.FindByID(ctx, ownerID, notebookID)
	if err != nil {
		return utils.BuildNotebookRepositoryError(err, "查询目标笔记簿失败")
	}
	if notebook.Status != model.NotebookStatusNormal {
		return utils.NewAppError(utils.KindConflict, "目标笔记簿当前不可用")
	}
	return nil
}
