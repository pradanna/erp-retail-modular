package domain_test

import (
	"errors"
	"testing"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

func TestNewStockItem(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		productID   string
		locationID  string
		quantity    int
		minStock    int
		expectedErr error
	}{
		{
			name:        "Valid StockItem",
			id:          "stk-01",
			productID:   "prod-01",
			locationID:  "loc-01",
			quantity:    50,
			minStock:    5,
			expectedErr: nil,
		},
		{
			name:        "Empty ID",
			id:          "",
			productID:   "prod-01",
			locationID:  "loc-01",
			quantity:    10,
			minStock:    5,
			expectedErr: domain.ErrInvalidStockID,
		},
		{
			name:        "Empty ProductID",
			id:          "stk-01",
			productID:   "",
			locationID:  "loc-01",
			quantity:    10,
			minStock:    5,
			expectedErr: domain.ErrInvalidProductID,
		},
		{
			name:        "Empty LocationID",
			id:          "stk-01",
			productID:   "prod-01",
			locationID:  "",
			quantity:    10,
			minStock:    5,
			expectedErr: domain.ErrInvalidLocationID,
		},
		{
			name:        "Negative Quantity",
			id:          "stk-01",
			productID:   "prod-01",
			locationID:  "loc-01",
			quantity:    -1,
			minStock:    5,
			expectedErr: domain.ErrNegativeQuantity,
		},
		{
			name:        "Negative MinStock",
			id:          "stk-01",
			productID:   "prod-01",
			locationID:  "loc-01",
			quantity:    10,
			minStock:    -1,
			expectedErr: domain.ErrNegativeMinStock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, err := domain.NewStockItem(tt.id, tt.productID, tt.locationID, tt.quantity, tt.minStock)
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if item.Quantity != tt.quantity {
				t.Errorf("expected quantity %d, got %d", tt.quantity, item.Quantity)
			}
			if item.ReservedQuantity != 0 {
				t.Errorf("expected reserved quantity 0, got %d", item.ReservedQuantity)
			}
			if item.AvailableQuantity() != tt.quantity {
				t.Errorf("expected available quantity %d, got %d", tt.quantity, item.AvailableQuantity())
			}
		})
	}
}

func TestStockItem_AdjustQuantity(t *testing.T) {
	item, err := domain.NewStockItem("stk-01", "prod-01", "loc-01", 20, 5)
	if err != nil {
		t.Fatalf("failed to create stock item: %v", err)
	}

	// 1. Adjust ke jumlah valid
	if err := item.AdjustQuantity(35); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Quantity != 35 {
		t.Errorf("expected quantity 35, got %d", item.Quantity)
	}

	// 2. Adjust ke negatif (gagal)
	if err := item.AdjustQuantity(-5); !errors.Is(err, domain.ErrNegativeQuantity) {
		t.Errorf("expected ErrNegativeQuantity, got %v", err)
	}

	// 3. Reserve 10 item, lalu coba adjust di bawah 10 (gagal)
	if err := item.Reserve(10); err != nil {
		t.Fatalf("unexpected error on reserve: %v", err)
	}
	if err := item.AdjustQuantity(8); !errors.Is(err, domain.ErrReservedExceedsStock) {
		t.Errorf("expected ErrReservedExceedsStock, got %v", err)
	}
}

func TestStockItem_ReserveAndRelease(t *testing.T) {
	item, err := domain.NewStockItem("stk-01", "prod-01", "loc-01", 15, 3)
	if err != nil {
		t.Fatalf("failed to create stock item: %v", err)
	}

	// 1. Reserve 5 unit
	if err := item.Reserve(5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.ReservedQuantity != 5 {
		t.Errorf("expected reserved 5, got %d", item.ReservedQuantity)
	}
	if item.AvailableQuantity() != 10 {
		t.Errorf("expected available 10, got %d", item.AvailableQuantity())
	}

	// 2. Reserve melebihi Available (11 unit) -> Gagal
	if err := item.Reserve(11); !errors.Is(err, domain.ErrInsufficientStock) {
		t.Errorf("expected ErrInsufficientStock, got %v", err)
	}

	// 3. Reserve invalid qty (0 atau negatif)
	if err := item.Reserve(0); !errors.Is(err, domain.ErrInvalidReserveQty) {
		t.Errorf("expected ErrInvalidReserveQty, got %v", err)
	}

	// 4. Release 2 unit
	if err := item.ReleaseReservation(2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.ReservedQuantity != 3 {
		t.Errorf("expected reserved 3, got %d", item.ReservedQuantity)
	}
	if item.AvailableQuantity() != 12 {
		t.Errorf("expected available 12, got %d", item.AvailableQuantity())
	}

	// 5. Release melebihi reserved (4 unit padahal cuma ada 3) -> Gagal
	if err := item.ReleaseReservation(4); !errors.Is(err, domain.ErrExceedsReservedQty) {
		t.Errorf("expected ErrExceedsReservedQty, got %v", err)
	}
}

func TestStockItem_DeductReserved(t *testing.T) {
	item, err := domain.NewStockItem("stk-01", "prod-01", "loc-01", 20, 5)
	if err != nil {
		t.Fatalf("failed to create stock item: %v", err)
	}

	if err := item.Reserve(10); err != nil {
		t.Fatalf("failed to reserve: %v", err)
	}

	// Potong 6 unit yang sudah selesai dikirim/dibeli
	if err := item.DeductReserved(6); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Quantity != 14 {
		t.Errorf("expected physical quantity 14, got %d", item.Quantity)
	}
	if item.ReservedQuantity != 4 {
		t.Errorf("expected reserved quantity 4, got %d", item.ReservedQuantity)
	}
	if item.AvailableQuantity() != 10 {
		t.Errorf("expected available quantity 10, got %d", item.AvailableQuantity())
	}

	// Coba potong lebih dari sisa reservasi (5 unit padahal sisa 4) -> Gagal
	if err := item.DeductReserved(5); !errors.Is(err, domain.ErrExceedsReservedQty) {
		t.Errorf("expected ErrExceedsReservedQty, got %v", err)
	}
}

func TestStockItem_LowStock(t *testing.T) {
	item, err := domain.NewStockItem("stk-01", "prod-01", "loc-01", 10, 5)
	if err != nil {
		t.Fatalf("failed to create stock item: %v", err)
	}

	if item.IsLowStock() {
		t.Errorf("expected false, got true")
	}

	// Adjust ke tepat 5
	_ = item.AdjustQuantity(5)
	if !item.IsLowStock() {
		t.Errorf("expected true when quantity == min_stock")
	}

	// Adjust ke 2
	_ = item.AdjustQuantity(2)
	if !item.IsLowStock() {
		t.Errorf("expected true when quantity < min_stock")
	}

	// Update min stock
	if err := item.UpdateMinStock(-1); !errors.Is(err, domain.ErrNegativeMinStock) {
		t.Errorf("expected ErrNegativeMinStock, got %v", err)
	}

	if err := item.UpdateMinStock(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.IsLowStock() {
		t.Errorf("expected false when quantity (2) > min_stock (1)")
	}
}
