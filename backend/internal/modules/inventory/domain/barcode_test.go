package domain_test

import (
	"errors"
	"testing"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

func TestNewProductBarcode(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		productID   string
		barcode     string
		isPrimary   bool
		expectedErr error
	}{
		{
			name:        "Valid EAN-13 barcode",
			id:          "bar-01",
			productID:   "prod-01",
			barcode:     "8806091234567",
			isPrimary:   true,
			expectedErr: nil,
		},
		{
			name:        "Valid alphanumeric barcode with dash",
			id:          "bar-02",
			productID:   "prod-01",
			barcode:     "ABC-12345.X",
			isPrimary:   false,
			expectedErr: nil,
		},
		{
			name:        "Barcode with surrounding whitespace should be trimmed",
			id:          "bar-03",
			productID:   "prod-01",
			barcode:     "  8806091234567  ",
			isPrimary:   false,
			expectedErr: nil,
		},
		{
			name:        "Empty ID",
			id:          "",
			productID:   "prod-01",
			barcode:     "8806091234567",
			isPrimary:   false,
			expectedErr: domain.ErrInvalidBarcodeID,
		},
		{
			name:        "Empty ProductID",
			id:          "bar-01",
			productID:   "",
			barcode:     "8806091234567",
			isPrimary:   false,
			expectedErr: domain.ErrInvalidProductID,
		},
		{
			name:        "Barcode too short (< 3 chars)",
			id:          "bar-01",
			productID:   "prod-01",
			barcode:     "12",
			isPrimary:   false,
			expectedErr: domain.ErrInvalidBarcodeCode,
		},
		{
			name:        "Barcode with spaces inside or invalid characters",
			id:          "bar-01",
			productID:   "prod-01",
			barcode:     "8806 0912",
			isPrimary:   false,
			expectedErr: domain.ErrBarcodeHasInvalidChar,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := domain.NewProductBarcode(tt.id, tt.productID, tt.barcode, tt.isPrimary)
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if b.IsPrimary != tt.isPrimary {
				t.Errorf("expected isPrimary %v, got %v", tt.isPrimary, b.IsPrimary)
			}
		})
	}
}

func TestProductBarcode_SetPrimary(t *testing.T) {
	b, err := domain.NewProductBarcode("bar-01", "prod-01", "8806091234567", false)
	if err != nil {
		t.Fatalf("failed to create barcode: %v", err)
	}

	if b.IsPrimary {
		t.Errorf("expected false, got true")
	}

	b.SetPrimary(true)
	if !b.IsPrimary {
		t.Errorf("expected true, got false")
	}
}
