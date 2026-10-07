package service

import (
	"context"
	"fmt"
	"unicode/utf8"

	"note-vault/app/dto"
	"note-vault/app/model"
	"note-vault/app/repository"
	"note-vault/app/utils"
)

const (
	maxNotebookNameLength = 100
)

// NotebookService 负责笔记簿业务规则、数据归属校验和 DTO/Model 转换。
type NotebookService struct {
	notebooks *repository.NotebookRepository
	notes     *repository.NoteRepository
}

func NewNotebookService() *NotebookService {
	return &NotebookService{notebooks: repository.NewNotebookRepository(), notes: repository.NewNoteRepository()}
}

func (s *NotebookService) CreateNotebook(ctx context.Context, ownerID uint, input dto.NotebookRequest) (*dto.NotebookResponse, error) {
	name, err := utils.RequiredText(input.Name, "笔记簿名")
	if err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(name) > maxNotebookNameLength {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "笔记簿名长度不能大于 100")
	}
	if err := utils.ValidNotebookOptionalFields(&input); err != nil {
		return nil, err
	}
	status := model.NotebookStatusNormal
	if input.Status != nil {
		status = *input.Status
	}
	if !utils.ValidNotebookStatus(status) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "笔记簿状态无效")
	}
	sortOrder := 0
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}
	if sortOrder < 0 {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "笔记簿排序值不能小于 0")
	}
	exists, err := s.notebooks.ExistsByName(ctx, ownerID, name, 0)
	if err != nil {
		return nil, utils.BuildNotebookRepositoryError(err, "检查笔记簿名失败")
	}
	if exists {
		return nil, utils.NewAppError(utils.KindConflict, "同一用户下笔记簿名已存在")
	}

	notebook := model.Notebook{
		OwnerID: ownerID, Name: name, Tag: input.Tag, Description: input.Description,
		Status: status, SortOrder: sortOrder,
	}
	if input.Icon != nil {
		notebook.Icon = *input.Icon
	}
	if err := s.notebooks.Create(ctx, &notebook); err != nil {
		return nil, utils.BuildNotebookRepositoryError(err, "创建笔记簿失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("新增笔记簿《%s》（ID %d）", notebook.Name, notebook.ID))
	result := dto.NewNotebookResponse(notebook)
	return &result, nil
}

func (s *NotebookService) ListNotebooks(ctx context.Context, ownerID uint, input dto.NotebookRequest) (*dto.PageResponse[dto.NotebookResponse], error) {
	page, pageSize, err := utils.NormalizePage(input.Page, input.PageSize)
	if err != nil {
		return nil, err
	}
	if input.Status != nil && !utils.ValidNotebookStatus(*input.Status) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "笔记簿状态无效")
	}
	input.Page, input.PageSize = &page, &pageSize
	notebooks, total, err := s.notebooks.List(ctx, ownerID, input)
	if err != nil {
		return nil, utils.BuildNotebookRepositoryError(err, "获取笔记簿列表失败")
	}
	ids := make([]uint, 0, len(notebooks))
	for _, notebook := range notebooks {
		ids = append(ids, notebook.ID)
	}
	counts, err := s.notes.CountByNotebookIDs(ctx, ownerID, ids)
	if err != nil {
		return nil, utils.BuildNotebookRepositoryError(err, "统计笔记簿笔记数量失败")
	}
	return &dto.PageResponse[dto.NotebookResponse]{
		List: dto.NewNotebookResponses(notebooks, counts), Page: page, PageSize: pageSize, Total: total,
	}, nil
}

func (s *NotebookService) GetNotebook(ctx context.Context, ownerID, id uint) (*dto.NotebookResponse, error) {
	notebook, err := s.notebooks.FindByID(ctx, ownerID, id)
	if err != nil {
		return nil, utils.BuildNotebookRepositoryError(err, "获取笔记簿详情失败")
	}
	result := dto.NewNotebookResponse(*notebook)
	counts, err := s.notes.CountByNotebookIDs(ctx, ownerID, []uint{id})
	if err != nil {
		return nil, utils.BuildNotebookRepositoryError(err, "统计笔记簿笔记数量失败")
	}
	result.NoteCount = counts[id]
	return &result, nil
}

func (s *NotebookService) UpdateNotebook(ctx context.Context, ownerID, id uint, input dto.NotebookRequest) (*dto.NotebookResponse, error) {
	if input.Name == nil && input.Icon == nil && input.Tag == nil && input.Description == nil {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "至少提供一个需要修改的笔记簿字段")
	}
	if _, err := s.notebooks.FindByID(ctx, ownerID, id); err != nil {
		return nil, utils.BuildNotebookRepositoryError(err, "获取笔记簿详情失败")
	}
	if input.Name != nil {
		name, err := utils.RequiredText(input.Name, "笔记簿名")
		if err != nil {
			return nil, err
		}
		if utf8.RuneCountInString(name) > maxNotebookNameLength {
			return nil, utils.NewAppError(utils.KindInvalidArgument, "笔记簿名长度不能大于 100")
		}
		*input.Name = name
		exists, err := s.notebooks.ExistsByName(ctx, ownerID, name, id)
		if err != nil {
			return nil, utils.BuildNotebookRepositoryError(err, "检查笔记簿名失败")
		}
		if exists {
			return nil, utils.NewAppError(utils.KindConflict, "同一用户下笔记簿名已存在")
		}
	}
	if err := utils.ValidNotebookOptionalFields(&input); err != nil {
		return nil, err
	}
	if err := s.notebooks.Update(ctx, ownerID, id, input); err != nil {
		return nil, utils.BuildNotebookRepositoryError(err, "更新笔记簿失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("修改笔记簿 ID %d", id))
	return s.GetNotebook(ctx, ownerID, id)
}

func (s *NotebookService) UpdateNotebookSort(ctx context.Context, ownerID, id uint, input dto.NotebookRequest) (*dto.NotebookResponse, error) {
	if input.SortOrder == nil || *input.SortOrder < 0 {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "笔记簿排序值必须是大于等于 0 的整数")
	}
	if err := s.notebooks.UpdateSort(ctx, ownerID, id, *input.SortOrder); err != nil {
		return nil, utils.BuildNotebookRepositoryError(err, "更新笔记簿排序失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("修改笔记簿 ID %d 的排序为 %d", id, *input.SortOrder))
	return s.GetNotebook(ctx, ownerID, id)
}

func (s *NotebookService) UpdateNotebookStatus(ctx context.Context, ownerID, id uint, input dto.NotebookRequest) (*dto.NotebookResponse, error) {
	if input.Status == nil || !utils.ValidNotebookStatus(*input.Status) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "笔记簿状态无效")
	}
	if err := s.notebooks.UpdateStatus(ctx, ownerID, id, *input.Status); err != nil {
		return nil, utils.BuildNotebookRepositoryError(err, "更新笔记簿状态失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("修改笔记簿 ID %d 的状态为 %d", id, *input.Status))
	return s.GetNotebook(ctx, ownerID, id)
}

func (s *NotebookService) DeleteNotebook(ctx context.Context, ownerID, id uint) error {
	noteCount, err := s.notebooks.DeleteIfEmpty(ctx, ownerID, id)
	if err != nil {
		return utils.BuildNotebookRepositoryError(err, "删除笔记簿失败")
	}
	if noteCount > 0 {
		return utils.NewAppError(
			utils.KindConflict,
			fmt.Sprintf("笔记簿中存在 %d 条笔记，不能删除", noteCount),
		)
	}
	auditRecorder.Record(ctx, fmt.Sprintf("删除笔记簿 ID %d", id))
	return nil
}
