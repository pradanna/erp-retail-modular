package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/pkg/uid"
)

var (
	ErrLocationNotFound = errors.New("lokasi tidak ditemukan")
	ErrDuplicateCode    = errors.New("kode lokasi sudah terdaftar")
)

// CreateLocationCommand membawa data input untuk pembuatan lokasi/cabang baru.
type CreateLocationCommand struct {
	Code    string
	Name    string
	Type    domain.LocationType
	Address string
}

// CreateLocationUseCase menangani proses pembuatan lokasi baru.
type CreateLocationUseCase struct {
	repo domain.LocationRepository
}

func NewCreateLocationUseCase(repo domain.LocationRepository) *CreateLocationUseCase {
	return &CreateLocationUseCase{repo: repo}
}

func (uc *CreateLocationUseCase) Execute(ctx context.Context, cmd CreateLocationCommand) (string, error) {
	// 1. Validasi keunikan kode cabang di database
	existing, err := uc.repo.FindByCode(ctx, cmd.Code)
	if err != nil {
		return "", fmt.Errorf("gagal memeriksa kode lokasi: %w", err)
	}
	if existing != nil {
		return "", fmt.Errorf("%w: %s", ErrDuplicateCode, cmd.Code)
	}

	// 2. Generate UUIDv7 untuk entitas baru
	locationID := uid.New()

	// 3. Buat entity domain dengan validasi invariant internal
	location, err := domain.NewLocation(locationID, cmd.Code, cmd.Name, cmd.Type, cmd.Address)
	if err != nil {
		return "", err
	}

	// 4. Simpan ke database
	if err := uc.repo.Save(ctx, location); err != nil {
		return "", fmt.Errorf("gagal menyimpan lokasi: %w", err)
	}

	return locationID, nil
}

// ListLocationsUseCase menangani pengambilan daftar seluruh lokasi.
type ListLocationsUseCase struct {
	repo domain.LocationRepository
}

func NewListLocationsUseCase(repo domain.LocationRepository) *ListLocationsUseCase {
	return &ListLocationsUseCase{repo: repo}
}

func (uc *ListLocationsUseCase) Execute(ctx context.Context, activeOnly bool) ([]*domain.Location, error) {
	return uc.repo.List(ctx, activeOnly)
}

// GetLocationUseCase menangani pencarian detail 1 lokasi berdasarkan ID.
type GetLocationUseCase struct {
	repo domain.LocationRepository
}

func NewGetLocationUseCase(repo domain.LocationRepository) *GetLocationUseCase {
	return &GetLocationUseCase{repo: repo}
}

func (uc *GetLocationUseCase) Execute(ctx context.Context, id string) (*domain.Location, error) {
	if id == "" {
		return nil, errors.New("ID lokasi wajib diisi")
	}

	loc, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("gagal mencari lokasi: %w", err)
	}
	if loc == nil {
		return nil, ErrLocationNotFound
	}

	return loc, nil
}

// UpdateLocationCommand membawa data pembaruan informasi lokasi.
type UpdateLocationCommand struct {
	ID      string
	Code    string
	Name    string
	Type    domain.LocationType
	Address string
}

// UpdateLocationUseCase menangani pembaruan nama, kode, tipe, dan alamat lokasi.
type UpdateLocationUseCase struct {
	repo domain.LocationRepository
}

func NewUpdateLocationUseCase(repo domain.LocationRepository) *UpdateLocationUseCase {
	return &UpdateLocationUseCase{repo: repo}
}

func (uc *UpdateLocationUseCase) Execute(ctx context.Context, cmd UpdateLocationCommand) error {
	if cmd.ID == "" {
		return errors.New("ID lokasi wajib diisi")
	}

	loc, err := uc.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		return fmt.Errorf("gagal mencari lokasi: %w", err)
	}
	if loc == nil {
		return ErrLocationNotFound
	}

	// Jika code diubah, pastikan code baru belum digunakan oleh lokasi lain
	if cmd.Code != "" && cmd.Code != loc.Code {
		existing, err := uc.repo.FindByCode(ctx, cmd.Code)
		if err != nil {
			return fmt.Errorf("gagal memeriksa kode lokasi: %w", err)
		}
		if existing != nil && existing.ID != loc.ID {
			return fmt.Errorf("%w: %s", ErrDuplicateCode, cmd.Code)
		}
	}

	if err := loc.UpdateDetails(cmd.Code, cmd.Name, cmd.Type, cmd.Address); err != nil {
		return err
	}

	if err := uc.repo.Update(ctx, loc); err != nil {
		return fmt.Errorf("gagal memperbarui lokasi: %w", err)
	}

	return nil
}

// SetLocationStatusUseCase menangani aktivasi dan deaktivasi (soft delete) lokasi.
type SetLocationStatusUseCase struct {
	repo domain.LocationRepository
}

func NewSetLocationStatusUseCase(repo domain.LocationRepository) *SetLocationStatusUseCase {
	return &SetLocationStatusUseCase{repo: repo}
}

func (uc *SetLocationStatusUseCase) Execute(ctx context.Context, id string, isActive bool) error {
	if id == "" {
		return errors.New("ID lokasi wajib diisi")
	}

	loc, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("gagal mencari lokasi: %w", err)
	}
	if loc == nil {
		return ErrLocationNotFound
	}

	if isActive {
		loc.Activate()
	} else {
		loc.Deactivate()
	}

	if err := uc.repo.Update(ctx, loc); err != nil {
		return fmt.Errorf("gagal mengubah status lokasi: %w", err)
	}

	return nil
}

// DeleteLocationUseCase menangani penghapusan fisik lokasi.
// Akan gagal jika lokasi masih terikat dengan data stok atau transaksi (ON DELETE RESTRICT).
type DeleteLocationUseCase struct {
	repo domain.LocationRepository
}

func NewDeleteLocationUseCase(repo domain.LocationRepository) *DeleteLocationUseCase {
	return &DeleteLocationUseCase{repo: repo}
}

func (uc *DeleteLocationUseCase) Execute(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("ID lokasi wajib diisi")
	}

	loc, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("gagal mencari lokasi: %w", err)
	}
	if loc == nil {
		return ErrLocationNotFound
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("lokasi tidak dapat dihapus (mungkin masih memiliki data stok atau transaksi terkait): %w", err)
	}

	return nil
}
