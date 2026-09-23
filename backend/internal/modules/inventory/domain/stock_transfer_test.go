package domain_test

import (
	"testing"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

func TestNewStockTransfer_Validation(t *testing.T) {
	validItems := []domain.StockTransferItemInput{
		{ProductID: "prod-tv", Quantity: 2},
		{ProductID: "prod-kulkas", Quantity: 1},
	}

	tests := []struct {
		name         string
		id           string
		number       string
		fromLoc      string
		toLoc        string
		notes        string
		requestedBy  string
		items        []domain.StockTransferItemInput
		expectedErr  error
	}{
		{
			name:        "ID kosong harus error",
			id:          "",
			number:      "TRF-001",
			fromLoc:     "loc-jakarta",
			toLoc:       "loc-bandung",
			requestedBy: "user-1",
			items:       validItems,
			expectedErr: domain.ErrInvalidTransferID,
		},
		{
			name:        "Nomor transfer kosong harus error",
			id:          "trf-1",
			number:      "  ",
			fromLoc:     "loc-jakarta",
			toLoc:       "loc-bandung",
			requestedBy: "user-1",
			items:       validItems,
			expectedErr: domain.ErrInvalidTransferNumber,
		},
		{
			name:        "Cabang asal dan tujuan sama harus error",
			id:          "trf-1",
			number:      "TRF-001",
			fromLoc:     "loc-jakarta",
			toLoc:       "loc-jakarta",
			requestedBy: "user-1",
			items:       validItems,
			expectedErr: domain.ErrSameLocationTransfer,
		},
		{
			name:        "Items kosong harus error",
			id:          "trf-1",
			number:      "TRF-001",
			fromLoc:     "loc-jakarta",
			toLoc:       "loc-bandung",
			requestedBy: "user-1",
			items:       []domain.StockTransferItemInput{},
			expectedErr: domain.ErrEmptyTransferItems,
		},
		{
			name:        "Kuantitas item 0 atau negatif harus error",
			id:          "trf-1",
			number:      "TRF-001",
			fromLoc:     "loc-jakarta",
			toLoc:       "loc-bandung",
			requestedBy: "user-1",
			items: []domain.StockTransferItemInput{
				{ProductID: "prod-tv", Quantity: 0},
			},
			expectedErr: domain.ErrInvalidTransferQuantity,
		},
		{
			name:        "Produk duplikat dalam 1 transfer harus error",
			id:          "trf-1",
			number:      "TRF-001",
			fromLoc:     "loc-jakarta",
			toLoc:       "loc-bandung",
			requestedBy: "user-1",
			items: []domain.StockTransferItemInput{
				{ProductID: "prod-tv", Quantity: 1},
				{ProductID: "prod-tv", Quantity: 2},
			},
			expectedErr: domain.ErrDuplicateTransferItem,
		},
		{
			name:        "Transfer multi-item valid berhasil dibuat",
			id:          "trf-1",
			number:      "TRF-202609-0001",
			fromLoc:     "loc-jakarta",
			toLoc:       "loc-bandung",
			notes:       "Mutasi stok pembukaan cabang",
			requestedBy: "admin-gudang",
			items:       validItems,
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trf, err := domain.NewStockTransfer(tt.id, tt.number, tt.fromLoc, tt.toLoc, tt.notes, tt.requestedBy, tt.items)
			if tt.expectedErr != nil {
				if err != tt.expectedErr {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if trf.Status != domain.TransferStatusPendingApproval {
					t.Fatalf("expected initial status pending_approval, got %s", trf.Status)
				}
				if len(trf.Items) != 2 {
					t.Fatalf("expected 2 items, got %d", len(trf.Items))
				}
			}
		})
	}
}

func TestStockTransfer_StateMachineWorkflow(t *testing.T) {
	items := []domain.StockTransferItemInput{
		{ProductID: "prod-tv", Quantity: 5},
	}

	trf, err := domain.NewStockTransfer("trf-1", "TRF-001", "loc-jkt", "loc-bdg", "Pengiriman TV", "admin-jkt", items)
	if err != nil {
		t.Fatalf("failed to create StockTransfer: %v", err)
	}

	// 1. Coba transisi ilegal: pending_approval langsung ke received -> harus ditolak
	if err := trf.Receive("admin-bdg"); err != domain.ErrIllegalStatusTransition {
		t.Fatalf("expected ErrIllegalStatusTransition, got %v", err)
	}
	// 2. Coba transisi ilegal: pending_approval langsung ke in_transit -> harus ditolak
	if err := trf.Ship(); err != domain.ErrIllegalStatusTransition {
		t.Fatalf("expected ErrIllegalStatusTransition, got %v", err)
	}

	// 3. Alur Sah: Approve oleh Owner/Superadmin
	if err := trf.Approve("superadmin-1"); err != nil {
		t.Fatalf("failed to approve transfer: %v", err)
	}
	if trf.Status != domain.TransferStatusApproved {
		t.Fatalf("expected status approved, got %s", trf.Status)
	}
	if trf.ApprovedBy == nil || *trf.ApprovedBy != "superadmin-1" {
		t.Fatalf("expected approved_by superadmin-1, got %v", trf.ApprovedBy)
	}

	// 4. Alur Sah: Truk Berangkat (in_transit)
	if err := trf.Ship(); err != nil {
		t.Fatalf("failed to ship transfer: %v", err)
	}
	if trf.Status != domain.TransferStatusInTransit {
		t.Fatalf("expected status in_transit, got %s", trf.Status)
	}

	// 5. Coba transisi ilegal: in_transit tidak bisa di-approve ulang
	if err := trf.Approve("superadmin-2"); err != domain.ErrIllegalStatusTransition {
		t.Fatalf("expected ErrIllegalStatusTransition, got %v", err)
	}

	// 6. Alur Sah: Barang Tiba di Cabang Penerima (received)
	if err := trf.Receive("admin-bdg"); err != nil {
		t.Fatalf("failed to receive transfer: %v", err)
	}
	if trf.Status != domain.TransferStatusReceived {
		t.Fatalf("expected status received, got %s", trf.Status)
	}
	if trf.ReceivedBy == nil || *trf.ReceivedBy != "admin-bdg" {
		t.Fatalf("expected received_by admin-bdg, got %v", trf.ReceivedBy)
	}
	if trf.Items[0].ReceivedQuantity != 5 {
		t.Fatalf("expected full received quantity 5, got %d", trf.Items[0].ReceivedQuantity)
	}

	// 7. Barang yang sudah 'received' tidak boleh diubah statusnya lagi
	if err := trf.Reject("superadmin-1", "Alasan palsu"); err != domain.ErrIllegalStatusTransition {
		t.Fatalf("expected ErrIllegalStatusTransition after received, got %v", err)
	}
}

func TestStockTransfer_RejectAndCancel(t *testing.T) {
	items := []domain.StockTransferItemInput{
		{ProductID: "prod-tv", Quantity: 1},
	}

	// Test Reject
	t.Run("Reject dari pending_approval", func(t *testing.T) {
		trf, _ := domain.NewStockTransfer("trf-1", "TRF-001", "loc-jkt", "loc-bdg", "Catatan", "admin-jkt", items)
		if err := trf.Reject("superadmin-1", "Stok Jakarta tidak mencukupi"); err != nil {
			t.Fatalf("failed to reject: %v", err)
		}
		if trf.Status != domain.TransferStatusRejected {
			t.Fatalf("expected status rejected, got %s", trf.Status)
		}
		if trf.RejectionReason != "Stok Jakarta tidak mencukupi" {
			t.Fatalf("expected rejection reason, got %s", trf.RejectionReason)
		}
	})

	// Test Cancel
	t.Run("Cancel dari pending_approval", func(t *testing.T) {
		trf, _ := domain.NewStockTransfer("trf-2", "TRF-002", "loc-jkt", "loc-bdg", "Catatan", "admin-jkt", items)
		if err := trf.Cancel(); err != nil {
			t.Fatalf("failed to cancel: %v", err)
		}
		if trf.Status != domain.TransferStatusCancelled {
			t.Fatalf("expected status cancelled, got %s", trf.Status)
		}
	})
}
