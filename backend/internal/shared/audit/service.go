package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/erp-retail/backend/internal/shared/event"
)

// Service menangani pencatatan dan pembacaan jejak audit aktivitas sistem.
type Service struct {
	repo Repository
	log  *slog.Logger
}

// NewService membuat instance baru audit Service.
func NewService(repo Repository, log *slog.Logger) *Service {
	return &Service{
		repo: repo,
		log:  log,
	}
}

// Record mencatat satu aksi ke dalam tabel audit_logs.
func (s *Service) Record(ctx context.Context, entry AuditLogEntry) error {
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	if entry.UserName == "" {
		entry.UserName = "System"
	}
	if entry.UserRole == "" {
		entry.UserRole = "system"
	}

	if err := s.repo.Insert(ctx, &entry); err != nil {
		s.log.Error("gagal menyimpan audit log", "action", entry.Action, "error", err)
		return err
	}
	return nil
}

// List mengambil daftar audit log dengan filter dan pagination.
func (s *Service) List(ctx context.Context, filter AuditFilter) ([]*AuditLogEntry, int, error) {
	return s.repo.List(ctx, filter)
}

// SubscribeEventBus mendaftarkan listener untuk event-event kunci agar otomatis tercatat ke audit log.
func (s *Service) SubscribeEventBus(bus event.Bus) {
	// 1. Audit Stock Opname (Penyesuaian Fisik Stok)
	bus.Subscribe(event.EventStockAdjusted, func(payload any) {
		p, ok := payload.(event.StockAdjustedPayload)
		if !ok {
			s.log.Warn("payload EventStockAdjusted tidak valid", "type", fmt.Sprintf("%T", payload))
			return
		}

		diffSign := ""
		if p.Difference > 0 {
			diffSign = fmt.Sprintf("+%d", p.Difference)
		} else {
			diffSign = fmt.Sprintf("%d", p.Difference)
		}

		summary := fmt.Sprintf(
			"Stock Opname: %s di %s diubah dari %d ke %d unit (%s). Alasan: %s",
			p.ProductName, p.LocationName, p.PreviousQty, p.NewQty, diffSign, p.Reason,
		)

		detailsBytes, _ := json.Marshal(map[string]any{
			"adjustment_id": p.AdjustmentID,
			"product_id":    p.ProductID,
			"product_name":  p.ProductName,
			"location_id":   p.LocationID,
			"location_name": p.LocationName,
			"previous_qty":  p.PreviousQty,
			"new_qty":       p.NewQty,
			"difference":    p.Difference,
			"reason":        p.Reason,
		})
		detailsStr := string(detailsBytes)

		entry := AuditLogEntry{
			UserID:     &p.AdjustedBy,
			UserName:   p.AdjustedByName,
			UserRole:   "staff",
			Action:     "inventory.stock_opname",
			Module:     "inventory",
			TargetType: strPtr("stock"),
			TargetID:   &p.ProductID,
			Summary:    summary,
			Details:    &detailsStr,
			CreatedAt:  time.Now().UTC(),
		}

		if err := s.Record(context.Background(), entry); err != nil {
			s.log.Error("gagal merekam audit log stock opname", "error", err)
		}
	})

	// 2. Audit Penerimaan Barang (Goods Received)
	bus.Subscribe(event.EventGoodsReceived, func(payload any) {
		p, ok := payload.(event.GoodsReceivedPayload)
		if !ok {
			return
		}

		summary := fmt.Sprintf("Penerimaan barang PO %s di lokasi %s", p.PurchaseOrderID, p.LocationID)
		detailsBytes, _ := json.Marshal(p)
		detailsStr := string(detailsBytes)

		entry := AuditLogEntry{
			UserID:     &p.ReceivedBy,
			UserName:   "Admin Gudang",
			UserRole:   "warehouse",
			Action:     "purchasing.goods_received",
			Module:     "purchasing",
			TargetType: strPtr("purchase_order"),
			TargetID:   &p.PurchaseOrderID,
			Summary:    summary,
			Details:    &detailsStr,
			CreatedAt:  time.Now().UTC(),
		}
		_ = s.Record(context.Background(), entry)
	})

	// 3. Audit Transaksi Penjualan Terbayar (Order Paid)
	bus.Subscribe(event.EventOrderPaid, func(payload any) {
		p, ok := payload.(event.OrderPaidPayload)
		if !ok {
			return
		}

		summary := fmt.Sprintf("Transaksi penjualan %s terbayar lunas (Rp %d)", p.OrderID, p.Total)
		detailsBytes, _ := json.Marshal(p)
		detailsStr := string(detailsBytes)

		entry := AuditLogEntry{
			UserID:     p.SalesmanID,
			UserName:   "Kasir",
			UserRole:   "cashier",
			Action:     "sales.order_paid",
			Module:     "sales",
			TargetType: strPtr("order"),
			TargetID:   &p.OrderID,
			Summary:    summary,
			Details:    &detailsStr,
			CreatedAt:  time.Now().UTC(),
		}
		_ = s.Record(context.Background(), entry)
	})
}

func strPtr(s string) *string {
	return &s
}
