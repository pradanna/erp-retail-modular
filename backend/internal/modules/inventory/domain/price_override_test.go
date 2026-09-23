package domain_test

import (
	"testing"
	"time"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

func TestNewPriceOverride_Validation(t *testing.T) {
	now := time.Now().UTC()
	start := now.Add(1 * time.Hour)
	end := now.Add(24 * time.Hour)

	tests := []struct {
		name        string
		id          string
		productID   string
		locationID  string
		price       int64
		start       time.Time
		end         time.Time
		reason      string
		expectedErr error
	}{
		{
			name:        "ID kosong harus error",
			id:          "",
			productID:   "prod-1",
			locationID:  "loc-1",
			price:       100000,
			start:       start,
			end:         end,
			reason:      "Promo",
			expectedErr: domain.ErrInvalidOverrideID,
		},
		{
			name:        "ProductID kosong harus error",
			id:          "po-1",
			productID:   "",
			locationID:  "loc-1",
			price:       100000,
			start:       start,
			end:         end,
			reason:      "Promo",
			expectedErr: domain.ErrInvalidProductID,
		},
		{
			name:        "LocationID kosong harus error",
			id:          "po-1",
			productID:   "prod-1",
			locationID:  "",
			price:       100000,
			start:       start,
			end:         end,
			reason:      "Promo",
			expectedErr: domain.ErrInvalidLocationID,
		},
		{
			name:        "Harga promo 0 atau negatif harus error",
			id:          "po-1",
			productID:   "prod-1",
			locationID:  "loc-1",
			price:       0,
			start:       start,
			end:         end,
			reason:      "Promo",
			expectedErr: domain.ErrInvalidPromoPrice,
		},
		{
			name:        "Tanggal end_date sebelum start_date harus error",
			id:          "po-1",
			productID:   "prod-1",
			locationID:  "loc-1",
			price:       100000,
			start:       end,
			end:         start,
			reason:      "Promo",
			expectedErr: domain.ErrInvalidPromoDates,
		},
		{
			name:        "Data valid berhasil dibuat (unlimited promo)",
			id:          "po-1",
			productID:   "prod-1",
			locationID:  "loc-1",
			price:       150000,
			start:       start,
			end:         end,
			reason:      "  Diskon Pembukaan Cabang  ",
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			po, err := domain.NewPriceOverride(tt.id, tt.productID, tt.locationID, tt.price, tt.start, tt.end, tt.reason, nil)
			if tt.expectedErr != nil {
				if err != tt.expectedErr {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if po.Reason != "Diskon Pembukaan Cabang" {
					t.Fatalf("expected trimmed reason, got '%s'", po.Reason)
				}
				if !po.IsActive {
					t.Fatal("expected IsActive to be true initially")
				}
				if po.MaxQuantity != nil {
					t.Fatal("expected MaxQuantity to be nil for unlimited promo")
				}
			}
		})
	}

	// Uji validasi kuota
	t.Run("Kuota 0 atau negatif harus error", func(t *testing.T) {
		zero := 0
		_, err := domain.NewPriceOverride("po-1", "prod-1", "loc-1", 100000, start, end, "Promo", &zero)
		if err != domain.ErrInvalidPromoQuota {
			t.Fatalf("expected ErrInvalidPromoQuota, got %v", err)
		}

		negative := -5
		_, err = domain.NewPriceOverride("po-1", "prod-1", "loc-1", 100000, start, end, "Promo", &negative)
		if err != domain.ErrInvalidPromoQuota {
			t.Fatalf("expected ErrInvalidPromoQuota, got %v", err)
		}
	})
}

func TestPriceOverride_ActiveAndOverlap(t *testing.T) {
	now := time.Now().UTC()
	start := now.Add(-2 * time.Hour) // Dimulai 2 jam lalu
	end := now.Add(2 * time.Hour)    // Berakhir 2 jam lagi

	po, err := domain.NewPriceOverride("po-1", "prod-1", "loc-1", 75000, start, end, "Flash Sale Siang", nil)
	if err != nil {
		t.Fatalf("failed to create PriceOverride: %v", err)
	}

	// 1. Cek apakah aktif sekarang
	if !po.IsActiveAt(now) {
		t.Fatal("expected promo to be active right now")
	}

	// 2. Cek apakah aktif di masa lalu (3 jam lalu) -> harus false
	past := now.Add(-3 * time.Hour)
	if po.IsActiveAt(past) {
		t.Fatal("expected promo NOT to be active 3 hours ago")
	}

	// 3. Cek apakah aktif di masa depan (3 jam lagi) -> harus false
	future := now.Add(3 * time.Hour)
	if po.IsActiveAt(future) {
		t.Fatal("expected promo NOT to be active 3 hours in the future")
	}

	// 4. Uji tumpang tindih (Overlap):
	// Kasus A: Rentang baru bertabrakan di tengah [now-1h, now+1h]
	if !po.OverlapsWith(now.Add(-1*time.Hour), now.Add(1*time.Hour)) {
		t.Fatal("expected overlap for middle-intersecting range")
	}

	// Kasus B: Rentang baru bertabrakan di awal [now-3h, now-1h]
	if !po.OverlapsWith(now.Add(-3*time.Hour), now.Add(-1*time.Hour)) {
		t.Fatal("expected overlap for start-intersecting range")
	}

	// Kasus C: Rentang baru bertabrakan di akhir [now+1h, now+3h]
	if !po.OverlapsWith(now.Add(1*time.Hour), now.Add(3*time.Hour)) {
		t.Fatal("expected overlap for end-intersecting range")
	}

	// Kasus D: Rentang baru sama sekali di luar (sudah lewat) [now-5h, now-3h]
	if po.OverlapsWith(now.Add(-5*time.Hour), now.Add(-3*time.Hour)) {
		t.Fatal("expected NO overlap for completely past range")
	}

	// Kasus E: Rentang baru sama sekali di luar (belum mulai) [now+3h, now+5h]
	if po.OverlapsWith(now.Add(3*time.Hour), now.Add(5*time.Hour)) {
		t.Fatal("expected NO overlap for completely future range")
	}

	// 5. Uji Deaktivasi manual
	po.Deactivate()
	if po.IsActive {
		t.Fatal("expected IsActive to be false after Deactivate()")
	}
	if po.IsActiveAt(now) {
		t.Fatal("expected promo NOT to be active now after Deactivate()")
	}
}

func TestPriceOverride_Quota(t *testing.T) {
	now := time.Now().UTC()
	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)
	quota := 3

	po, err := domain.NewPriceOverride("po-1", "prod-1", "loc-1", 50000, start, end, "Flash Sale 3 Unit", &quota)
	if err != nil {
		t.Fatalf("failed to create PriceOverride with quota: %v", err)
	}

	// 1. Sisa kuota awal harus 3
	rem := po.RemainingQuota()
	if rem == nil || *rem != 3 {
		t.Fatalf("expected remaining quota 3, got %v", rem)
	}
	if !po.IsActiveAt(now) {
		t.Fatal("expected promo to be active when quota is available")
	}

	// 2. Klaim 2 unit
	if err := po.ClaimQuota(2); err != nil {
		t.Fatalf("failed to claim quota: %v", err)
	}
	if po.ClaimedQuantity != 2 {
		t.Fatalf("expected claimed 2, got %d", po.ClaimedQuantity)
	}
	rem = po.RemainingQuota()
	if rem == nil || *rem != 1 {
		t.Fatalf("expected remaining quota 1, got %v", rem)
	}
	if !po.IsActiveAt(now) {
		t.Fatal("expected promo still active when 1 quota left")
	}

	// 3. Klaim 2 unit lagi (melebihi kuota sisa 1) -> harus error
	if err := po.ClaimQuota(2); err != domain.ErrPromoQuotaExhausted {
		t.Fatalf("expected ErrPromoQuotaExhausted, got %v", err)
	}

	// 4. Klaim 1 unit terakhir -> kuota pas habis
	if err := po.ClaimQuota(1); err != nil {
		t.Fatalf("failed to claim last quota: %v", err)
	}
	rem = po.RemainingQuota()
	if rem == nil || *rem != 0 {
		t.Fatalf("expected remaining quota 0, got %v", rem)
	}

	// 5. Setelah kuota habis, IsActiveAt harus false (otomatis non-aktif)
	if po.IsActiveAt(now) {
		t.Fatal("expected promo NOT to be active after quota exhausted")
	}
}
