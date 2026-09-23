package domain_test

import (
	"testing"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

func TestNewCategory_Success(t *testing.T) {
	c, err := domain.NewCategory("cat-01", "Elektronik", nil, nil)
	if err != nil {
		t.Fatalf("seharusnya berhasil membuat category, error: %v", err)
	}

	if c.Name != "Elektronik" {
		t.Errorf("ekspektasi nama = 'Elektronik', didapat: %s", c.Name)
	}

	if c.ParentID != nil {
		t.Errorf("ekspektasi ParentID nil, didapat: %v", c.ParentID)
	}
}

func TestNewCategory_ValidationError(t *testing.T) {
	_, err := domain.NewCategory("cat-01", "   ", nil, nil)
	if err == nil {
		t.Errorf("seharusnya error ketika nama kategori kosong")
	}
}

func TestCategory_SetParent_CircularReference(t *testing.T) {
	c, err := domain.NewCategory("cat-01", "Elektronik", nil, nil)
	if err != nil {
		t.Fatalf("gagal inisialisasi: %v", err)
	}

	// Tidak boleh menjadikan dirinya sendiri sebagai parent
	selfID := "cat-01"
	err = c.SetParent(&selfID)
	if err == nil {
		t.Errorf("seharusnya error ketika parent ID sama dengan category ID")
	}
}
