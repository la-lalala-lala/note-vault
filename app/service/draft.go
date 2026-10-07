package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"note-vault/app/dto"
	"note-vault/app/model"
	"note-vault/app/repository"
	"note-vault/app/utils"
)

const maxDraftKeyLength = 100

// DraftService 负责草稿自动保存、数据归属校验和事务发布流程。
type DraftService struct {
	drafts    *repository.DraftRepository
	notes     *repository.NoteRepository
	notebooks *repository.NotebookRepository
}

func NewDraftService() *DraftService {
	return &DraftService{
		drafts:    repository.NewDraftRepository(),
		notes:     repository.NewNoteRepository(),
		notebooks: repository.NewNotebookRepository(),
	}
}

func (s *DraftService) SaveDraft(ctx context.Context, ownerID uint, draftKey string, input dto.DraftRequest) (*dto.DraftResponse, error) {
	draftKey = strings.TrimSpace(draftKey)
	if draftKey == "" {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "草稿 key 不能为空")
	}
	if utf8.RuneCountInString(draftKey) > maxDraftKeyLength {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "草稿 key 长度不能大于 100")
	}
	if err := utils.ValidDraftFields(&input); err != nil {
		return nil, err
	}
	if input.NoteID != nil {
		if _, err := s.notes.FindByID(ctx, ownerID, *input.NoteID); err != nil {
			return nil, utils.BuildNoteRepositoryError(err, "校验草稿关联笔记失败")
		}
	}
	if input.NotebookID != nil {
		if err := s.validateTargetNotebook(ctx, ownerID, *input.NotebookID); err != nil {
			return nil, err
		}
	}
	draft, err := s.drafts.Save(ctx, ownerID, draftKey, input)
	if err != nil {
		return nil, utils.BuildDraftRepositoryError(err, "保存草稿失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("保存草稿（ID %d，Key %s）", draft.ID, draftKey))
	response := dto.NewDraftResponse(*draft)
	return &response, nil
}

func (s *DraftService) ListDrafts(ctx context.Context, ownerID uint, input dto.DraftRequest) (*dto.PageResponse[dto.DraftResponse], error) {
	page, pageSize, err := utils.NormalizePage(input.Page, input.PageSize)
	if err != nil {
		return nil, err
	}
	drafts, total, err := s.drafts.List(ctx, ownerID, page, pageSize)
	if err != nil {
		return nil, utils.BuildDraftRepositoryError(err, "查询草稿列表失败")
	}
	return &dto.PageResponse[dto.DraftResponse]{
		List: dto.NewDraftResponses(drafts), Page: page, PageSize: pageSize, Total: total,
	}, nil
}

func (s *DraftService) GetDraft(ctx context.Context, ownerID, id uint) (*dto.DraftResponse, error) {
	draft, err := s.drafts.FindByID(ctx, ownerID, id)
	if err != nil {
		return nil, utils.BuildDraftRepositoryError(err, "查询草稿详情失败")
	}
	response := dto.NewDraftResponse(*draft)
	return &response, nil
}

func (s *DraftService) SubmitDraft(ctx context.Context, ownerID, id uint, input dto.DraftRequest) (*dto.NoteResponse, error) {
	// 草稿读取、关联资源校验、笔记保存和草稿删除由仓储层在同一事务中完成。
	note, err := s.drafts.Submit(ctx, ownerID, id, input.NoteVersion)
	if err != nil {
		return nil, utils.BuildDraftRepositoryError(err, "提交草稿失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("提交草稿 %d 为笔记 %d", id, note.ID))
	response := dto.NewNoteResponse(*note)
	return &response, nil
}

func (s *DraftService) DeleteDraft(ctx context.Context, ownerID, id uint) error {
	if err := s.drafts.Delete(ctx, ownerID, id); err != nil {
		return utils.BuildDraftRepositoryError(err, "删除草稿失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("删除草稿 ID %d", id))
	return nil
}

func (s *DraftService) validateTargetNotebook(ctx context.Context, ownerID, notebookID uint) error {
	notebook, err := s.notebooks.FindByID(ctx, ownerID, notebookID)
	if err != nil {
		return utils.BuildNotebookRepositoryError(err, "查询草稿目标笔记簿失败")
	}
	if notebook.Status != model.NotebookStatusNormal {
		return utils.BuildDraftRepositoryError(utils.ErrNotebookUnavailable, "校验草稿目标笔记簿失败")
	}
	return nil
}
