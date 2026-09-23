package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/pkg/uid"
)

var ErrCategoryNotFound = errors.New("kategori tidak ditemukan")

// CreateCategoryCommand membawa data input untuk pembuatan kategori baru.
type CreateCategoryCommand struct {
	Name     string
	ParentID *string
	ImageURL *string
}

// CreateCategoryUseCase menangani proses pembuatan kategori baru.
type CreateCategoryUseCase struct {
	repo domain.CategoryRepository
}

func NewCreateCategoryUseCase(repo domain.CategoryRepository) *CreateCategoryUseCase {
	return &CreateCategoryUseCase{repo: repo}
}

func (uc *CreateCategoryUseCase) Execute(ctx context.Context, cmd CreateCategoryCommand) (string, error) {
	// Jika parent_id diisi, pastikan parent category benar-benar ada di database
	if cmd.ParentID != nil && *cmd.ParentID != "" {
		parent, err := uc.repo.FindByID(ctx, *cmd.ParentID)
		if err != nil {
			return "", fmt.Errorf("gagal memvalidasi parent category: %w", err)
		}
		if parent == nil {
			return "", errors.New("parent kategori tidak ditemukan")
		}
	}

	categoryID := uid.New()
	category, err := domain.NewCategory(categoryID, cmd.Name, cmd.ParentID, cmd.ImageURL)
	if err != nil {
		return "", err
	}

	if err := uc.repo.Save(ctx, category); err != nil {
		return "", fmt.Errorf("gagal menyimpan kategori: %w", err)
	}

	return categoryID, nil
}

// ListCategoriesUseCase menangani pengambilan semua kategori.
type ListCategoriesUseCase struct {
	repo domain.CategoryRepository
}

func NewListCategoriesUseCase(repo domain.CategoryRepository) *ListCategoriesUseCase {
	return &ListCategoriesUseCase{repo: repo}
}

func (uc *ListCategoriesUseCase) Execute(ctx context.Context) ([]*domain.Category, error) {
	return uc.repo.List(ctx)
}

// UpdateCategoryCommand membawa data pembaruan kategori.
type UpdateCategoryCommand struct {
	ID       string
	Name     string
	ParentID *string
	ImageURL *string
}

// UpdateCategoryUseCase menangani pengubahan data kategori.
type UpdateCategoryUseCase struct {
	repo domain.CategoryRepository
}

func NewUpdateCategoryUseCase(repo domain.CategoryRepository) *UpdateCategoryUseCase {
	return &UpdateCategoryUseCase{repo: repo}
}

func (uc *UpdateCategoryUseCase) Execute(ctx context.Context, cmd UpdateCategoryCommand) error {
	if cmd.ID == "" {
		return errors.New("ID kategori wajib diisi")
	}

	category, err := uc.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		return fmt.Errorf("gagal mencari kategori: %w", err)
	}
	if category == nil {
		return ErrCategoryNotFound
	}

	// Cegah circular reference: parent_id tidak boleh diri sendiri
	if cmd.ParentID != nil && *cmd.ParentID == cmd.ID {
		return errors.New("kategori tidak boleh menjadi induk untuk dirinya sendiri")
	}

	if cmd.ParentID != nil && *cmd.ParentID != "" {
		parent, err := uc.repo.FindByID(ctx, *cmd.ParentID)
		if err != nil {
			return fmt.Errorf("gagal memvalidasi parent category: %w", err)
		}
		if parent == nil {
			return errors.New("parent kategori tidak ditemukan")
		}
	}

	if err := category.UpdateName(cmd.Name); err != nil {
		return err
	}
	if err := category.SetParent(cmd.ParentID); err != nil {
		return err
	}
	category.SetImageURL(cmd.ImageURL)

	if err := uc.repo.Update(ctx, category); err != nil {
		return fmt.Errorf("gagal memperbarui kategori: %w", err)
	}

	return nil
}

// DeleteCategoryUseCase menangani penghapusan kategori.
type DeleteCategoryUseCase struct {
	repo domain.CategoryRepository
}

func NewDeleteCategoryUseCase(repo domain.CategoryRepository) *DeleteCategoryUseCase {
	return &DeleteCategoryUseCase{repo: repo}
}

func (uc *DeleteCategoryUseCase) Execute(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("ID kategori wajib diisi")
	}

	category, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("gagal mencari kategori: %w", err)
	}
	if category == nil {
		return ErrCategoryNotFound
	}

	return uc.repo.Delete(ctx, id)
}
