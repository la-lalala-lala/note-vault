package repository

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"note-vault/app/config"
	"note-vault/app/dto"
	"note-vault/app/generated"
	"note-vault/app/model"
	"note-vault/app/utils"

	"gorm.io/cli/gorm/typed"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DraftRepository struct {
	db *gorm.DB
}

func NewDraftRepository(db ...*gorm.DB) *DraftRepository {
	repository := &DraftRepository{}
	if len(db) > 0 {
		repository.db = db[0]
	}
	return repository
}

func (r *DraftRepository) database() (*gorm.DB, error) {
	if r.db != nil {
		return r.db, nil
	}
	if config.DB == nil {
		return nil, errors.New("database is unavailable")
	}
	return config.DB, nil
}

// FindByID 使用草稿 ID 和当前用户 ID 查询草稿。
func (r *DraftRepository) FindByID(ctx context.Context, ownerID, id uint) (*model.Draft, error) {
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	draft, err := typed.G[model.Draft](db).
		Where(generated.Draft.ID.Eq(id)).
		Where(generated.Draft.OwnerID.Eq(ownerID)).
		Take(ctx)
	if err != nil {
		return nil, err
	}
	return &draft, nil
}

// List 查询当前用户的草稿分页列表，不加载正文和正文外部存储地址。
func (r *DraftRepository) List(ctx context.Context, ownerID uint, page, pageSize int) ([]model.Draft, int64, error) {
	db, err := r.database()
	if err != nil {
		return nil, 0, err
	}
	query := typed.G[model.Draft](db).Where(generated.Draft.OwnerID.Eq(ownerID))
	total, err := query.Count(ctx, "*")
	if err != nil {
		return nil, 0, err
	}
	drafts, err := query.
		Select(
			generated.Draft.ID,
			generated.Draft.DraftKey,
			generated.Draft.NoteID,
			generated.Draft.NotebookID,
			generated.Draft.Title,
			generated.Draft.Label,
			generated.Draft.ContentType,
			generated.Draft.Version,
			generated.Draft.ExpiresAt,
			generated.Draft.CreatedAt,
			generated.Draft.UpdatedAt,
		).
		Order(generated.Draft.UpdatedAt.OrderExpr("? DESC", generated.Draft.UpdatedAt)).
		Order(generated.Draft.ID.OrderExpr("? DESC", generated.Draft.ID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(ctx)
	return drafts, total, err
}

// Save 使用草稿会话标识执行幂等保存。
// 新建笔记按 owner_id + draft_key 定位；已有笔记优先按 owner_id + note_id 定位。
// 保存采用最后一次写入生效策略，Version 只作为服务端修订号递增，不做客户端 CAS。
func (r *DraftRepository) Save(ctx context.Context, ownerID uint, draftKey string, input dto.DraftRequest) (*model.Draft, error) {
	// 优先使用显式注入的数据库连接；未注入时使用全局数据库连接。
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	// 保存和更新必须在同一个事务中完成，避免并发请求产生不一致结果。
	var savedDraft model.Draft
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 新建笔记按 draftKey 查找，编辑已有笔记还会按 noteID 查找已有草稿。
		currentDraft, err := findDraftForSave(ctx, tx, ownerID, draftKey, input.NoteID)
		if err == nil {
			// 已存在草稿时，禁止把新建草稿切换成已有笔记草稿，或切换到其他笔记。
			if err := validateDraftAssociation(currentDraft, input); err != nil {
				return err
			}
			// 采用最后一次写入生效策略，并递增服务端修订号。
			if err := updateDraft(ctx, tx, ownerID, currentDraft.ID, input); err != nil {
				return err
			}
			// 重新查询更新后的完整草稿，确保返回最新的修订号和更新时间。
			savedDraft, err = typed.G[model.Draft](tx).
				Where(generated.Draft.ID.Eq(currentDraft.ID)).
				Where(generated.Draft.OwnerID.Eq(ownerID)).
				Take(ctx)
			return err
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// 没有找到已有草稿时，按照当前用户和 draftKey 创建新草稿。
		newDraft := newDraftModel(ownerID, draftKey, input)
		// 冲突时不报错，后续重新查询并更新，以兼容并发首次保存请求。
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&newDraft)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			// 当前请求成功创建草稿，直接返回刚创建的对象。
			savedDraft = newDraft
			return nil
		}

		// 创建未生效通常表示并发请求先创建了草稿，发生了冲突，此时重新获取实际记录。
		currentDraft, err = findDraftForSave(ctx, tx, ownerID, draftKey, input.NoteID)
		if err != nil {
			return err
		}
		// 并发创建冲突后重新查询也必须得到有效草稿，避免后续解引用空指针。
		if currentDraft == nil {
			return errors.New("draft query returned nil without error")
		}
		if err := validateDraftAssociation(currentDraft, input); err != nil {
			return err
		}
		// 将当前请求的数据更新到并发请求已经创建的草稿上。
		if err := updateDraft(ctx, tx, ownerID, currentDraft.ID, input); err != nil {
			return err
		}
		savedDraft, err = typed.G[model.Draft](tx).
			Where(generated.Draft.ID.Eq(currentDraft.ID)).
			Where(generated.Draft.OwnerID.Eq(ownerID)).
			Take(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	// 事务提交成功后返回最终保存的草稿。
	return &savedDraft, nil
}

// Delete 只物理删除当前用户自己的草稿。
func (r *DraftRepository) Delete(ctx context.Context, ownerID, id uint) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	rowsAffected, err := typed.G[model.Draft](db).
		Where(generated.Draft.ID.Eq(id)).
		Where(generated.Draft.OwnerID.Eq(ownerID)).
		Delete(ctx)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Submit 在同一事务中将草稿转换为正式笔记，并在笔记保存成功后删除草稿。
func (r *DraftRepository) Submit(ctx context.Context, ownerID, id uint, expectedNoteVersion *uint) (*model.Note, error) {
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	var submittedNote model.Note
	// 草稿写入笔记和草稿删除必须在同一事务中完成，确保失败时草稿不会丢失。
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 锁定当前用户自己的草稿，防止自动保存或其他提交同时修改它。
		draft, err := typed.G[model.Draft](tx).
			Scopes(func(statement *gorm.Statement) {
				statement.AddClause(clause.Locking{Strength: "UPDATE"})
			}).
			Where(generated.Draft.ID.Eq(id)).
			Where(generated.Draft.OwnerID.Eq(ownerID)).
			Take(ctx)
		if err != nil {
			return err
		}
		// 草稿标题是正式笔记的必填字段，提交前统一校验并使用规范化后的值。
		draft.Title = strings.TrimSpace(draft.Title)
		if draft.Title == "" {
			return utils.NewAppError(utils.KindInvalidArgument, "草稿标题不能为空，填写标题后才能提交")
		}
		if utf8.RuneCountInString(draft.Title) > 200 {
			return utils.NewAppError(utils.KindInvalidArgument, "草稿标题长度不能大于 200")
		}
		// note_id 为空表示新建笔记；不为空表示将草稿内容覆盖到已有笔记。
		var currentNote *model.Note
		if draft.NoteID != nil {
			// 编辑已有笔记时必须携带正式笔记版本，用于防止覆盖其他人的新修改。
			if expectedNoteVersion == nil || *expectedNoteVersion == 0 {
				return utils.ErrNoteVersionRequired
			}
			// 只读取后续需要的字段，避免为确定目标笔记簿加载整篇正文。
			note, err := typed.G[model.Note](tx).
				Select(generated.Note.ID, generated.Note.NotebookID).
				Where(generated.Note.ID.Eq(*draft.NoteID)).
				Where(generated.Note.OwnerID.Eq(ownerID)).
				Where(generated.Note.Status.Eq(model.NoteStatusNormal)).
				Take(ctx)
			if err != nil {
				return err
			}
			currentNote = &note
		}

		// 草稿指定了目标笔记簿时使用草稿值；编辑场景未指定时沿用原笔记簿。
		targetNotebookID := draft.NotebookID
		if targetNotebookID == nil && currentNote != nil {
			targetNotebookID = &currentNote.NotebookID
		}
		if targetNotebookID == nil {
			return utils.ErrTargetNotebookRequired
		}
		// 这里只校验归属、删除标记和状态，不锁定笔记簿，因为本事务不会修改它。
		targetNotebook, err := typed.G[model.Notebook](tx).
			Where(generated.Notebook.ID.Eq(*targetNotebookID)).
			Where(generated.Notebook.OwnerID.Eq(ownerID)).
			Where(generated.Notebook.DeletedAt.IsNull()).
			Take(ctx)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return utils.ErrNotebookUnavailable
			}
			return err
		}
		if targetNotebook.Status != model.NotebookStatusNormal {
			return utils.ErrNotebookUnavailable
		}

		if currentNote == nil {
			// 新建场景：把草稿字段组装为正式笔记并插入 notes 表。
			submittedNote = model.Note{
				OwnerID: ownerID, NotebookID: *targetNotebookID,
				Title: draft.Title, Label: draft.Label,
				Content: draft.Content, ContentURI: draft.ContentURI,
				ContentType: draft.ContentType, Status: model.NoteStatusNormal, Version: 1,
			}
			if err := typed.G[model.Note](tx).Create(ctx, &submittedNote); err != nil {
				return err
			}
		} else {
			// 编辑场景：只更新草稿能够表达的笔记字段，并递增正式笔记版本。
			assignments := []clause.Assigner{
				generated.Note.NotebookID.Set(*targetNotebookID),
				generated.Note.Title.Set(draft.Title),
				generated.Note.ContentType.Set(draft.ContentType),
				generated.Note.Version.SetExpr(generated.Note.Version.Incr(1)),
				generated.Note.UpdatedAt.Set(time.Now().UTC()),
			}
			if draft.Label != nil {
				assignments = append(assignments, generated.Note.Label.Set(*draft.Label))
			}
			if draft.Content != nil {
				assignments = append(
					assignments,
					generated.Note.Content.Set(*draft.Content),
					generated.Note.ContentURI.SetExpr(clause.Expr{SQL: "NULL"}),
				)
			} else if draft.ContentURI != nil {
				assignments = append(
					assignments,
					generated.Note.ContentURI.Set(*draft.ContentURI),
					generated.Note.Content.SetExpr(clause.Expr{SQL: "NULL"}),
				)
			}
			rowsAffected, err := typed.G[model.Note](tx).
				Where(generated.Note.ID.Eq(*draft.NoteID)).
				Where(generated.Note.OwnerID.Eq(ownerID)).
				Where(generated.Note.Status.Eq(model.NoteStatusNormal)).
				Where(generated.Note.Version.Eq(*expectedNoteVersion)).
				Set(assignments...).Update(ctx)
			if err != nil {
				return err
			}
			if rowsAffected == 0 {
				// 更新行数为零时再次查询，用于区分笔记不存在和正式笔记版本冲突。
				_, err := typed.G[model.Note](tx).
					Where(generated.Note.ID.Eq(*draft.NoteID)).
					Where(generated.Note.OwnerID.Eq(ownerID)).
					Where(generated.Note.Status.Eq(model.NoteStatusNormal)).
					Take(ctx)
				if err != nil {
					return err
				}
				return utils.ErrOptimisticLockConflict
			}
			// 查询更新后的笔记，返回数据库生成的 ID、时间和最新版本。
			submittedNote, err = typed.G[model.Note](tx).
				Where(generated.Note.ID.Eq(*draft.NoteID)).
				Where(generated.Note.OwnerID.Eq(ownerID)).
				Take(ctx)
			if err != nil {
				return err
			}
		}

		// 正式笔记保存成功后，删除当前用户自己的草稿。
		rowsAffected, err := typed.G[model.Draft](tx).
			Where(generated.Draft.ID.Eq(id)).
			Where(generated.Draft.OwnerID.Eq(ownerID)).
			Delete(ctx)
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		// 事务成功提交后，调用方才能看到正式笔记且草稿已不存在。
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &submittedNote, nil
}

/*
*
  - 在保存草稿前，查找当前应该被保存或更新的草稿，并对找到的记录加行级排他锁。
  - 查找顺序：
  - 优先按 owner_id + draftKey 查询
  - 用于新建笔记的编辑器会话。
  - 同一个编辑器持续使用同一个 draftKey。
  - 如果没有找到，并且请求包含 noteId，再按 owner_id + noteId 查询
  - 用于编辑已有笔记。
  - 保证同一个用户同一篇笔记只使用一份草稿。
  - 查询时使用 FOR UPDATE 行锁
  - 防止多个保存请求同时修改同一条草稿。
  - 这个函数只负责查找和加锁，不负责新增或更新草稿
*/
func findDraftForSave(ctx context.Context, tx *gorm.DB, ownerID uint, draftKey string, noteID *uint) (*model.Draft, error) {
	// 优先按照当前用户和编辑器生成的 draftKey 查找草稿。
	// 找到了，err返回nil 否则（没找到，或者DB异常）返回异常
	draft, err := typed.G[model.Draft](tx).
		// 加行级排他锁，避免读取后被其他保存请求同时修改。
		Scopes(func(statement *gorm.Statement) {
			statement.AddClause(clause.Locking{Strength: "UPDATE"})
		}).
		Where(generated.Draft.OwnerID.Eq(ownerID)).
		Where(generated.Draft.DraftKey.Eq(draftKey)).
		Take(ctx)
	if err == nil {
		// 按 draftKey 找到记录后直接返回，新增和后续自动保存都使用这条路径。
		return &draft, nil
	}
	if noteID == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		// 没有 noteID 时不能继续按笔记查找；其他数据库错误也直接返回。
		return nil, err
	}
	// 编辑已有笔记时，如果 draftKey 尚未对应草稿，再按照当前用户和 noteID 查找。
	draft, err = typed.G[model.Draft](tx).
		// 同样加行级排他锁，保证后续更新与本次查找处于同一事务保护下。
		Scopes(func(statement *gorm.Statement) {
			statement.AddClause(clause.Locking{Strength: "UPDATE"})
		}).
		Where(generated.Draft.OwnerID.Eq(ownerID)).
		Where(generated.Draft.NoteID.Eq(*noteID)).
		Take(ctx)
	if err != nil {
		// 草稿不存在或查询失败时交由上层统一处理。
		return nil, err
	}
	// 返回按 noteID 找到的已有笔记草稿。
	return &draft, nil
}

func validateDraftAssociation(current *model.Draft, input dto.DraftRequest) error {
	if current.NoteID == nil && input.NoteID != nil {
		return utils.ErrDraftAssociationConflict
	}
	if current.NoteID != nil && input.NoteID != nil && *input.NoteID != *current.NoteID {
		return utils.ErrDraftAssociationConflict
	}
	return nil
}

func updateDraft(ctx context.Context, tx *gorm.DB, ownerID, id uint, input dto.DraftRequest) error {
	assignments := draftAssignments(input)
	assignments = append(
		assignments,
		generated.Draft.Version.SetExpr(generated.Draft.Version.Incr(1)),
		generated.Draft.UpdatedAt.Set(time.Now().UTC()),
	)
	rowsAffected, err := typed.G[model.Draft](tx).
		Where(generated.Draft.ID.Eq(id)).
		Where(generated.Draft.OwnerID.Eq(ownerID)).
		Set(assignments...).Update(ctx)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func newDraftModel(ownerID uint, draftKey string, input dto.DraftRequest) model.Draft {
	draft := model.Draft{
		OwnerID: ownerID, DraftKey: draftKey, NoteID: input.NoteID,
		NotebookID:  input.NotebookID,
		ContentType: "markdown", Version: 1,
	}
	if input.Title != nil {
		draft.Title = strings.TrimSpace(*input.Title)
	}
	if input.Label != nil {
		label := strings.TrimSpace(*input.Label)
		draft.Label = &label
	}
	if input.Content != nil {
		draft.Content = input.Content
	}
	if input.ContentType != nil {
		draft.ContentType = strings.ToLower(strings.TrimSpace(*input.ContentType))
	}
	return draft
}

func draftAssignments(input dto.DraftRequest) []clause.Assigner {
	assignments := make([]clause.Assigner, 0, 8)
	if input.NotebookID != nil {
		assignments = append(assignments, generated.Draft.NotebookID.Set(*input.NotebookID))
	}
	if input.Title != nil {
		assignments = append(assignments, generated.Draft.Title.Set(strings.TrimSpace(*input.Title)))
	}
	if input.Label != nil {
		assignments = append(assignments, generated.Draft.Label.Set(strings.TrimSpace(*input.Label)))
	}
	if input.Content != nil {
		assignments = append(
			assignments,
			generated.Draft.Content.Set(*input.Content),
			generated.Draft.ContentURI.SetExpr(clause.Expr{SQL: "NULL"}),
		)
	}
	if input.ContentType != nil {
		assignments = append(assignments, generated.Draft.ContentType.Set(strings.ToLower(strings.TrimSpace(*input.ContentType))))
	}
	return assignments
}
