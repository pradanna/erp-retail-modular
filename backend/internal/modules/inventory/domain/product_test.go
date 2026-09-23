package domain_test

import (
	"testing"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

func TestNewProduct_Success(t *testing.T) {
	p, err := domain.NewProduct(
		"0191eb7a-0000-7000-8000-000000000001",
		"SKU-TV-001",
		"CAT-ELEKTRONIK",
		"Smart TV 43 Inch",
		"Samsung",
		"unit",
		3500000,
		4200000,
	)
	if err != nil {
		t.Fatalf("seharusnya berhasil membuat product, tetapi dapat error: %v", err)
	}

	if p.Status != domain.ProductStatusActive {
		t.Errorf("status awal produk harus active, didapat: %s", p.Status)
	}

	if !p.IsPPN {
		t.Errorf("default IsPPN harus true")
	}
}

func TestProduct_UpdateDetails_WeightGram(t *testing.T) {
	p, err := domain.NewProduct(
		"0191eb7a-0000-7000-8000-000000000001",
		"SKU-TV-001",
		"CAT-ELEKTRONIK",
		"Smart TV 43 Inch",
		"Samsung",
		"unit",
		3500000,
		4200000,
	)
	if err != nil {
		t.Fatalf("error inisialisasi product: %v", err)
	}

	// Update dengan berat 8500 gram (8.5 kg)
	err = p.UpdateDetails("Smart TV 43 Inch 4K", "Samsung", "unit", "Resolusi 4K UHD", 8500)
	if err != nil {
		t.Fatalf("gagal update details: %v", err)
	}

	if p.WeightGram != 8500 {
		t.Errorf("ekspektasi WeightGram = 8500, didapat: %d", p.WeightGram)
	}

	// Uji Invariant: berat tidak boleh negatif
	err = p.UpdateDetails("Smart TV 43 Inch 4K", "Samsung", "unit", "Resolusi 4K UHD", -500)
	if err == nil {
		t.Errorf("seharusnya return error saat berat negatif, tetapi nil")
	}
}
