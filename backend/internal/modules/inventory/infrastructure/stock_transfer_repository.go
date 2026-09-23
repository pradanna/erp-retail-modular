package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/pkg/uid"
)

type mysqlStockTransferRepository struct {
	db *sql.DB
}

// NewStockTransferRepository membuat implementasi baru domain.StockTransferRepository berbasis MySQL.
func NewStockTransferRepository(db *sql.DB) domain.StockTransferRepository {
	return &mysqlStockTransferRepository{db: db}
}

// Save menyimpan dokumen transfer baru beserta item-item dan nomor serinya secara atomik dalam transaksi database.
func (r *mysqlStockTransferRepository) Save(ctx context.Context, trf *domain.StockTransfer) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi save transfer: %w", err)
	}
	defer tx.Rollback()

	// 1. Insert header inv_stock_transfers
	headerQuery := `
		INSERT INTO inv_stock_transfers (
			id, transfer_number, from_location_id, to_location_id, status,
			notes, rejection_reason, requested_by, approved_by, received_by,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.ExecContext(ctx, headerQuery,
		trf.ID, trf.TransferNumber, trf.FromLocationID, trf.ToLocationID, string(trf.Status),
		trf.Notes, trf.RejectionReason, trf.RequestedBy, trf.ApprovedBy, trf.ReceivedBy,
		trf.CreatedAt, trf.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert header transfer: %w", err)
	}

	// 2. Insert items inv_stock_transfer_items
	itemStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO inv_stock_transfer_items (
			id, transfer_id, product_id, quantity, received_quantity, created_at
		) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("gagal prepare statement transfer items: %w", err)
	}
	defer itemStmt.Close()

	serialStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO inv_stock_transfer_item_serials (
			id, transfer_item_id, serial_unit_id, created_at
		) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("gagal prepare statement transfer item serials: %w", err)
	}
	defer serialStmt.Close()

	for _, item := range trf.Items {
		_, err := itemStmt.ExecContext(ctx,
			item.ID, trf.ID, item.ProductID, item.Quantity, item.ReceivedQuantity, item.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("gagal insert transfer item: %w", err)
		}

		// Insert nomor seri terkait (jika ada)
		for _, serialID := range item.SerialUnitIDs {
			linkID := uid.New()
			_, err := serialStmt.ExecContext(ctx, linkID, item.ID, serialID, item.CreatedAt)
			if err != nil {
				return fmt.Errorf("gagal insert transfer serial link: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit transaksi save transfer: %w", err)
	}

	return nil
}

// Update memperbarui status, penyetuju, penerima, atau kuantitas diterima dokumen transfer.
func (r *mysqlStockTransferRepository) Update(ctx context.Context, trf *domain.StockTransfer) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi update transfer: %w", err)
	}
	defer tx.Rollback()

	headerQuery := `
		UPDATE inv_stock_transfers SET
			status = ?, rejection_reason = ?, approved_by = ?, received_by = ?, updated_at = ?
		WHERE id = ?`

	res, err := tx.ExecContext(ctx, headerQuery,
		string(trf.Status), trf.RejectionReason, trf.ApprovedBy, trf.ReceivedBy, trf.UpdatedAt, trf.ID,
	)
	if err != nil {
		return fmt.Errorf("gagal update header transfer: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa rows affected transfer: %w", err)
	}
	if rows == 0 {
		return domain.ErrTransferNotFound
	}

	// Update detail items (received_quantity)
	itemStmt, err := tx.PrepareContext(ctx, `
		UPDATE inv_stock_transfer_items SET received_quantity = ? WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("gagal prepare statement update items: %w", err)
	}
	defer itemStmt.Close()

	for _, item := range trf.Items {
		_, err := itemStmt.ExecContext(ctx, item.ReceivedQuantity, item.ID)
		if err != nil {
			return fmt.Errorf("gagal update item received quantity: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit update transfer: %w", err)
	}

	return nil
}

// FindByID mencari transfer berdasarkan UUID dokumen lengkap dengan item-itemnya.
func (r *mysqlStockTransferRepository) FindByID(ctx context.Context, id string) (*domain.StockTransfer, error) {
	query := `
		SELECT id, transfer_number, from_location_id, to_location_id, status,
		       notes, rejection_reason, requested_by, approved_by, received_by,
		       created_at, updated_at
		FROM inv_stock_transfers
		WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	trf, err := r.scanHeader(row)
	if err != nil {
		return nil, err
	}

	items, err := r.loadItems(ctx, trf.ID)
	if err != nil {
		return nil, err
	}
	trf.Items = items

	return trf, nil
}

// FindByNumber mencari dokumen transfer berdasarkan nomor surat jalan.
func (r *mysqlStockTransferRepository) FindByNumber(ctx context.Context, number string) (*domain.StockTransfer, error) {
	query := `
		SELECT id, transfer_number, from_location_id, to_location_id, status,
		       notes, rejection_reason, requested_by, approved_by, received_by,
		       created_at, updated_at
		FROM inv_stock_transfers
		WHERE transfer_number = ?`

	row := r.db.QueryRowContext(ctx, query, number)
	trf, err := r.scanHeader(row)
	if err != nil {
		return nil, err
	}

	items, err := r.loadItems(ctx, trf.ID)
	if err != nil {
		return nil, err
	}
	trf.Items = items

	return trf, nil
}

// List mengambil daftar riwayat dokumen transfer berdasarkan kriteria filter.
func (r *mysqlStockTransferRepository) List(ctx context.Context, filter domain.StockTransferFilter) ([]*domain.StockTransfer, error) {
	query := `
		SELECT id, transfer_number, from_location_id, to_location_id, status,
		       notes, rejection_reason, requested_by, approved_by, received_by,
		       created_at, updated_at
		FROM inv_stock_transfers
		WHERE 1=1`
	var args []any

	if filter.FromLocationID != nil && *filter.FromLocationID != "" {
		query += ` AND from_location_id = ?`
		args = append(args, *filter.FromLocationID)
	}
	if filter.ToLocationID != nil && *filter.ToLocationID != "" {
		query += ` AND to_location_id = ?`
		args = append(args, *filter.ToLocationID)
	}
	if filter.Status != nil && *filter.Status != "" {
		query += ` AND status = ?`
		args = append(args, string(*filter.Status))
	}

	query += ` ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal query list transfer: %w", err)
	}
	defer rows.Close()

	var list []*domain.StockTransfer
	for rows.Next() {
		trf, err := r.scanHeader(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, trf)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi list transfer: %w", err)
	}

	// Muat items untuk setiap transfer
	for _, trf := range list {
		items, err := r.loadItems(ctx, trf.ID)
		if err != nil {
			return nil, err
		}
		trf.Items = items
	}

	return list, nil
}

// GenerateTransferNumber menghasilkan nomor surat jalan baru (TRF-YYYYMM-XXXX).
func (r *mysqlStockTransferRepository) GenerateTransferNumber(ctx context.Context) (string, error) {
	prefix := time.Now().Format("200601") // contoh: 202609
	pattern := fmt.Sprintf("TRF-%s-%%", prefix)

	query := `SELECT COUNT(*) FROM inv_stock_transfers WHERE transfer_number LIKE ?`
	var count int
	if err := r.db.QueryRowContext(ctx, query, pattern).Scan(&count); err != nil {
		return "", fmt.Errorf("gagal menghitung nomor transfer urut: %w", err)
	}

	return fmt.Sprintf("TRF-%s-%04d", prefix, count+1), nil
}

// Helper untuk membaca 1 header baris transfer
func (r *mysqlStockTransferRepository) scanHeader(scanner interface{ Scan(dest ...any) error }) (*domain.StockTransfer, error) {
	var (
		id, number, fromLoc, toLoc, status string
		notes, rejReason                   sql.NullString
		requestedBy                        string
		approvedBy, receivedBy             sql.NullString
		createdAt, updatedAt               time.Time
	)

	err := scanner.Scan(
		&id, &number, &fromLoc, &toLoc, &status,
		&notes, &rejReason, &requestedBy, &approvedBy, &receivedBy,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTransferNotFound
		}
		return nil, fmt.Errorf("gagal scan transfer header: %w", err)
	}

	var appBy *string
	if approvedBy.Valid {
		appBy = &approvedBy.String
	}

	var recBy *string
	if receivedBy.Valid {
		recBy = &receivedBy.String
	}

	return &domain.StockTransfer{
		ID:              id,
		TransferNumber:  number,
		FromLocationID:  fromLoc,
		ToLocationID:    toLoc,
		Status:          domain.TransferStatus(status),
		Notes:           notes.String,
		RejectionReason: rejReason.String,
		RequestedBy:     requestedBy,
		ApprovedBy:      appBy,
		ReceivedBy:      recBy,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}, nil
}

// Helper untuk membaca seluruh item barang dalam transfer
func (r *mysqlStockTransferRepository) loadItems(ctx context.Context, transferID string) ([]*domain.StockTransferItem, error) {
	query := `
		SELECT id, transfer_id, product_id, quantity, received_quantity, created_at
		FROM inv_stock_transfer_items
		WHERE transfer_id = ?
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, transferID)
	if err != nil {
		return nil, fmt.Errorf("gagal query transfer items: %w", err)
	}
	defer rows.Close()

	var items []*domain.StockTransferItem
	for rows.Next() {
		var (
			id, trfID, prodID string
			qty, recQty       int
			createdAt         time.Time
		)
		if err := rows.Scan(&id, &trfID, &prodID, &qty, &recQty, &createdAt); err != nil {
			return nil, fmt.Errorf("gagal scan transfer item: %w", err)
		}

		item := &domain.StockTransferItem{
			ID:               id,
			TransferID:       trfID,
			ProductID:        prodID,
			Quantity:         qty,
			ReceivedQuantity: recQty,
			SerialUnitIDs:    []string{},
			CreatedAt:        createdAt,
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi transfer items: %w", err)
	}

	// Load serial unit links untuk setiap item
	serialQuery := `
		SELECT serial_unit_id FROM inv_stock_transfer_item_serials WHERE transfer_item_id = ?`
	for _, it := range items {
		sRows, err := r.db.QueryContext(ctx, serialQuery, it.ID)
		if err != nil {
			return nil, fmt.Errorf("gagal query transfer item serials: %w", err)
		}
		for sRows.Next() {
			var sID string
			if err := sRows.Scan(&sID); err != nil {
				sRows.Close()
				return nil, err
			}
			it.SerialUnitIDs = append(it.SerialUnitIDs, sID)
		}
		sRows.Close()
	}

	return items, nil
}
