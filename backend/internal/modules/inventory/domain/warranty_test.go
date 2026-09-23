package domain_test

import (
	"testing"
	"time"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

func TestNewWarrantyPolicy(t *testing.T) {
	t.Run("sukses membuat kebijakan garansi toko valid", func(t *testing.T) {
		policy, err := domain.NewWarrantyPolicy(
			"pol-1",
			"Garansi Toko Tukar Baru 7 Hari",
			domain.WarrantyTypeToko,
			0,
			7,
			"Tukar unit baru jika cacat pabrik",
			"Bawa struk dan dus lengkap ke toko",
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if policy.ID != "pol-1" {
			t.Errorf("expected ID 'pol-1', got '%s'", policy.ID)
		}
		if policy.Name != "Garansi Toko Tukar Baru 7 Hari" {
			t.Errorf("expected Name 'Garansi Toko Tukar Baru 7 Hari', got '%s'", policy.Name)
		}
		if policy.Type != domain.WarrantyTypeToko {
			t.Errorf("expected Type 'toko', got '%s'", policy.Type)
		}
		if policy.DurationMonths != 0 || policy.DurationDays != 7 {
			t.Errorf("expected 0m 7d, got %dm %dd", policy.DurationMonths, policy.DurationDays)
		}
		if !policy.IsActive {
			t.Errorf("expected IsActive to be true")
		}

		// Test kalkulasi expiry date
		start := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
		expiry := policy.CalculateExpiryDate(start)
		expectedExpiry := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
		if !expiry.Equal(expectedExpiry) {
			t.Errorf("expected expiry %v, got %v", expectedExpiry, expiry)
		}
	})

	t.Run("sukses membuat kebijakan garansi pabrik 12 bulan", func(t *testing.T) {
		policy, err := domain.NewWarrantyPolicy(
			"pol-2",
			"Garansi Resmi Samsung Indonesia 12 Bulan",
			domain.WarrantyTypePabrik,
			12,
			0,
			"Servis dan suku cadang gratis di Authorized Service Center",
			"Kunjungi Samsung Service Center dengan membawa kartu garansi",
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if policy.Type != domain.WarrantyTypePabrik {
			t.Errorf("expected Type 'pabrik', got '%s'", policy.Type)
		}
		if policy.DurationMonths != 12 {
			t.Errorf("expected 12 months, got %d", policy.DurationMonths)
		}

		start := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
		expiry := policy.CalculateExpiryDate(start)
		expectedExpiry := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)
		if !expiry.Equal(expectedExpiry) {
			t.Errorf("expected expiry %v, got %v", expectedExpiry, expiry)
		}
	})

	t.Run("gagal jika nama kosong", func(t *testing.T) {
		_, err := domain.NewWarrantyPolicy("pol-3", "", domain.WarrantyTypeToko, 1, 0, "", "")
		if err != domain.ErrInvalidWarrantyName {
			t.Errorf("expected ErrInvalidWarrantyName, got %v", err)
		}
	})

	t.Run("gagal jika tipe tidak valid", func(t *testing.T) {
		_, err := domain.NewWarrantyPolicy("pol-4", "Garansi X", domain.WarrantyType("distributor_gelap"), 1, 0, "", "")
		if err != domain.ErrInvalidWarrantyType {
			t.Errorf("expected ErrInvalidWarrantyType, got %v", err)
		}
	})

	t.Run("gagal jika durasi 0 bulan dan 0 hari", func(t *testing.T) {
		_, err := domain.NewWarrantyPolicy("pol-5", "Garansi 0", domain.WarrantyTypeToko, 0, 0, "", "")
		if err != domain.ErrInvalidWarrantyDuration {
			t.Errorf("expected ErrInvalidWarrantyDuration, got %v", err)
		}
	})
}

func TestNewProductWarranty(t *testing.T) {
	t.Run("sukses membuat product warranty valid", func(t *testing.T) {
		pw, err := domain.NewProductWarranty("pw-1", "prod-1", "pol-1", domain.WarrantyTypeToko)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if pw.ID != "pw-1" || pw.ProductID != "prod-1" || pw.WarrantyPolicyID != "pol-1" {
			t.Errorf("mismatch in fields: %+v", pw)
		}
		if pw.Type != domain.WarrantyTypeToko {
			t.Errorf("expected Type 'toko', got '%s'", pw.Type)
		}
		if !pw.IsActive {
			t.Errorf("expected IsActive to be true")
		}

		pw.Deactivate()
		if pw.IsActive {
			t.Errorf("expected IsActive to be false after Deactivate")
		}
	})

	t.Run("gagal jika ID kosong", func(t *testing.T) {
		_, err := domain.NewProductWarranty("", "prod-1", "pol-1", domain.WarrantyTypeToko)
		if err != domain.ErrInvalidProductWarrantyID {
			t.Errorf("expected ErrInvalidProductWarrantyID, got %v", err)
		}
	})

	t.Run("gagal jika product ID kosong", func(t *testing.T) {
		_, err := domain.NewProductWarranty("pw-1", "", "pol-1", domain.WarrantyTypeToko)
		if err != domain.ErrInvalidProductID {
			t.Errorf("expected ErrInvalidProductID, got %v", err)
		}
	})

	t.Run("gagal jika policy ID kosong", func(t *testing.T) {
		_, err := domain.NewProductWarranty("pw-1", "prod-1", "", domain.WarrantyTypeToko)
		if err != domain.ErrInvalidPolicyID {
			t.Errorf("expected ErrInvalidPolicyID, got %v", err)
		}
	})
}
