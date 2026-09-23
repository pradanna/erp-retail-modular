package domain_test

import (
	"testing"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

func TestNewSerialUnit_Validation(t *testing.T) {
	tests := []struct {
		name         string
		id           string
		productID    string
		locationID   string
		serialNumber string
		expectedErr  error
	}{
		{
			name:         "ID kosong harus error",
			id:           "",
			productID:    "prod-1",
			locationID:   "loc-1",
			serialNumber: "SN-123456",
			expectedErr:  domain.ErrInvalidSerialID,
		},
		{
			name:         "ProductID kosong harus error",
			id:           "sn-1",
			productID:    "",
			locationID:   "loc-1",
			serialNumber: "SN-123456",
			expectedErr:  domain.ErrInvalidProductID,
		},
		{
			name:         "LocationID kosong harus error",
			id:           "sn-1",
			productID:    "prod-1",
			locationID:   "",
			serialNumber: "SN-123456",
			expectedErr:  domain.ErrInvalidLocationID,
		},
		{
			name:         "Serial number terlalu pendek (< 3 char)",
			id:           "sn-1",
			productID:    "prod-1",
			locationID:   "loc-1",
			serialNumber: "12",
			expectedErr:  domain.ErrInvalidSerialNumber,
		},
		{
			name:         "Serial number mengandung karakter terlarang (@)",
			id:           "sn-1",
			productID:    "prod-1",
			locationID:   "loc-1",
			serialNumber: "SN-123@#45",
			expectedErr:  domain.ErrSerialHasInvalidChar,
		},
		{
			name:         "Serial number valid dan format bersih",
			id:           "sn-1",
			productID:    "prod-1",
			locationID:   "loc-1",
			serialNumber: "  SN-LG-2026-X99.1_01  ",
			expectedErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unit, err := domain.NewSerialUnit(tt.id, tt.productID, tt.locationID, tt.serialNumber)
			if tt.expectedErr != nil {
				if err != tt.expectedErr {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if unit.SerialNumber != "SN-LG-2026-X99.1_01" {
					t.Fatalf("expected trimmed serial number, got %s", unit.SerialNumber)
				}
				if unit.Status != domain.SerialStatusAvailable {
					t.Fatalf("expected initial status 'tersedia', got %s", unit.Status)
				}
			}
		})
	}
}

func TestSerialUnit_StateMachine(t *testing.T) {
	unit, err := domain.NewSerialUnit("sn-1", "prod-1", "loc-1", "SN-SAMSUNG-S24-001")
	if err != nil {
		t.Fatalf("failed to create serial unit: %v", err)
	}

	// 1. Inisialisasi: status awal wajib 'tersedia'
	if unit.Status != domain.SerialStatusAvailable {
		t.Fatalf("expected status 'tersedia', got %s", unit.Status)
	}

	// 2. Unit belum terjual, coba retur -> harus ditolak
	if err := unit.MarkAsReturned(); err != domain.ErrSerialNotSold {
		t.Fatalf("expected ErrSerialNotSold when returning unsold unit, got %v", err)
	}

	// 3. Mutasi lokasi saat tersedia -> harus berhasil
	if err := unit.TransferLocation("loc-2"); err != nil {
		t.Fatalf("unexpected error on transfer: %v", err)
	}
	if unit.LocationID != "loc-2" {
		t.Fatalf("expected location 'loc-2', got %s", unit.LocationID)
	}

	// 4. Jual unit (MarkAsSold) -> status berubah jadi 'terjual'
	if err := unit.MarkAsSold(); err != nil {
		t.Fatalf("unexpected error on MarkAsSold: %v", err)
	}
	if unit.Status != domain.SerialStatusSold {
		t.Fatalf("expected status 'terjual', got %s", unit.Status)
	}

	// 5. Coba jual lagi unit yang sudah terjual -> harus ditolak
	if err := unit.MarkAsSold(); err != domain.ErrSerialAlreadySold {
		t.Fatalf("expected ErrSerialAlreadySold, got %v", err)
	}

	// 6. Coba mutasi cabang unit yang sudah terjual -> harus ditolak
	if err := unit.TransferLocation("loc-3"); err != domain.ErrSerialNotAvailable {
		t.Fatalf("expected ErrSerialNotAvailable when transferring sold unit, got %v", err)
	}

	// 7. Pelanggan meretur unit yang rusak (MarkAsReturned) -> status berubah jadi 'retur'
	if err := unit.MarkAsReturned(); err != nil {
		t.Fatalf("unexpected error on MarkAsReturned: %v", err)
	}
	if unit.Status != domain.SerialStatusReturned {
		t.Fatalf("expected status 'retur', got %s", unit.Status)
	}

	// 8. Coba retur lagi unit yang sudah diretur -> harus ditolak
	if err := unit.MarkAsReturned(); err != domain.ErrSerialNotSold {
		t.Fatalf("expected ErrSerialNotSold, got %v", err)
	}
}
