package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/pkg/uid"
)

type mysqlStockMovementRepository struct {
	db *sql.DB
}

// NewStockMovementRepository membuat implementasi baru domain.StockMovementRepository berbasis MySQL.
func NewStockMovementRepository(db *sql.DB) domain.StockMovementRepository {
	return &mysqlStockMovementRepository{db: db}
}

// Save menyimpan dokumen pergerakan stok beserta item dan serialnya dalam satu transaksi database.
func (r *mysqlStockMovementRepository) Save(ctx context.Context, m *domain.StockMovement) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi database: %w", err)
	}
	defer tx.Rollback()

	// 1. Insert header inv_stock_movements
	queryHeader := `
		INSERT INTO inv_stock_movements (
			id, movement_number, type, movement_date, location_id, category_reason,
			reference_number, notes, executed_by, executed_by_name, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.ExecContext(
		ctx,
		queryHeader,
		m.ID,
		m.MovementNumber,
		string(m.Type),
		m.MovementDate.Format("2006-01-02"),
		m.LocationID,
		m.CategoryReason,
		m.ReferenceNumber,
		m.Notes,
		m.ExecutedBy,
		m.ExecutedByName,
		m.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert header pergerakan stok: %w", err)
	}

	// 2. Insert items inv_stock_movement_items
	itemStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO inv_stock_movement_items (
			id, movement_id, product_id, quantity, notes, created_at
		) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("gagal prepare statement item pergerakan: %w", err)
	}
	defer itemStmt.Close()

	serialStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO inv_stock_movement_item_serials (
			id, movement_item_id, serial_unit_id, created_at
		) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("gagal prepare statement serial pergerakan: %w", err)
	}
	defer serialStmt.Close()

	for _, item := range m.Items {
		_, err = itemStmt.ExecContext(
			ctx,
			item.ID,
			m.ID,
			item.ProductID,
			item.Quantity,
			item.Notes,
			m.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("gagal insert item pergerakan stok (%s): %w", item.ProductID, err)
		}

		// Jika ada nomor seri terkait, simpan relasinya
		for _, sn := range item.SerialNumbers {
			var serialUnitID string
			// Cari apakah nomor seri sudah ada di inv_serial_units
			errLookup := tx.QueryRowContext(ctx, `SELECT id FROM inv_serial_units WHERE serial_number = ? LIMIT 1`, sn).Scan(&serialUnitID)
			if errLookup == sql.ErrNoRows {
				// Jika belum ada (misal pada barang masuk baru), buat entitas serial baru berstatus 'tersedia'
				serialUnitID = uid.New()
				_, errInsertSerial := tx.ExecContext(ctx, `
					INSERT INTO inv_serial_units (id, product_id, location_id, serial_number, status, created_at, updated_at)
					VALUES (?, ?, ?, ?, 'tersedia', ?, ?)`,
					serialUnitID, item.ProductID, m.LocationID, sn, m.CreatedAt, m.CreatedAt,
				)
				if errInsertSerial != nil {
					return fmt.Errorf("gagal mendaftarkan serial unit baru (%s): %w", sn, errInsertSerial)
				}
			} else if errLookup != nil {
				return fmt.Errorf("gagal memeriksa serial unit (%s): %w", sn, errLookup)
			}

			// Simpan relasi ke movement_item_serials
			linkID := uid.New()
			_, errLink := serialStmt.ExecContext(ctx, linkID, item.ID, serialUnitID, m.CreatedAt)
			if errLink != nil {
				return fmt.Errorf("gagal menghubungkan serial unit ke item pergerakan: %w", errLink)
			}
		}
	}

	return tx.Commit()
}

// FindByID mencari dokumen pergerakan stok lengkap dengan item dan serial number.
func (r *mysqlStockMovementRepository) FindByID(ctx context.Context, id string) (*domain.StockMovement, error) {
	queryHeader := `
		SELECT 
			m.id, m.movement_number, m.type, m.movement_date, m.location_id, COALESCE(l.name, '') as location_name,
			m.category_reason, COALESCE(m.reference_number, ''), COALESCE(m.notes, ''),
			m.executed_by, m.executed_by_name, m.created_at
		FROM inv_stock_movements m
		LEFT JOIN inv_locations l ON m.location_id = l.id
		WHERE m.id = ? LIMIT 1`

	var m domain.StockMovement
	var movType string
	err := r.db.QueryRowContext(ctx, queryHeader, id).Scan(
		&m.ID, &m.MovementNumber, &movType, &m.MovementDate, &m.LocationID, &m.LocationName,
		&m.CategoryReason, &m.ReferenceNumber, &m.Notes,
		&m.ExecutedBy, &m.ExecutedByName, &m.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal query header pergerakan stok: %w", err)
	}
	m.Type = domain.StockMovementType(movType)

	// Query items
	queryItems := `
		SELECT 
			i.id, i.movement_id, i.product_id, COALESCE(p.name, '') as product_name, COALESCE(p.sku, '') as product_sku,
			i.quantity, COALESCE(i.notes, ''), i.created_at
		FROM inv_stock_movement_items i
		LEFT JOIN inv_products p ON i.product_id = p.id
		WHERE i.movement_id = ?
		ORDER BY i.created_at ASC`

	rows, err := r.db.QueryContext(ctx, queryItems, id)
	if err != nil {
		return nil, fmt.Errorf("gagal query items pergerakan stok: %w", err)
	}
	defer rows.Close()

	itemsMap := make(map[string]*domain.StockMovementItem)
	var itemsList []*domain.StockMovementItem

	for rows.Next() {
		var it domain.StockMovementItem
		if err := rows.Scan(
			&it.ID, &it.MovementID, &it.ProductID, &it.ProductName, &it.ProductSKU,
			&it.Quantity, &it.Notes, &it.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal scan item pergerakan stok: %w", err)
		}
		itemsMap[it.ID] = &it
		itemsList = append(itemsList, &it)
	}

	// Query serials for each item
	querySerials := `
		SELECT mis.movement_item_id, su.serial_number
		FROM inv_stock_movement_item_serials mis
		JOIN inv_serial_units su ON mis.serial_unit_id = su.id
		WHERE mis.movement_item_id IN (
			SELECT id FROM inv_stock_movement_items WHERE movement_id = ?
		)`

	sRows, err := r.db.QueryContext(ctx, querySerials, id)
	if err == nil {
		defer sRows.Close()
		for sRows.Next() {
			var itemID, sn string
			if err := sRows.Scan(&itemID, &sn); err == nil {
				if it, ok := itemsMap[itemID]; ok {
					it.SerialNumbers = append(it.SerialNumbers, sn)
				}
			}
		}
	}

	m.Items = itemsList
	return &m, nil
}

// FindByNumber mencari dokumen berdasarkan nomor dokumen unik.
func (r *mysqlStockMovementRepository) FindByNumber(ctx context.Context, number string) (*domain.StockMovement, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM inv_stock_movements WHERE movement_number = ? LIMIT 1`, number).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

// List mengambil daftar riwayat dokumen pergerakan stok dengan filter dan pagination.
func (r *mysqlStockMovementRepository) List(ctx context.Context, filter domain.StockMovementFilter) ([]*domain.StockMovement, int, error) {
	var whereConditions []string
	var args []any

	if filter.LocationID != "" {
		whereConditions = append(whereConditions, "m.location_id = ?")
		args = append(args, filter.LocationID)
	}
	if filter.Type != nil {
		whereConditions = append(whereConditions, "m.type = ?")
		args = append(args, string(*filter.Type))
	}
	if filter.StartDate != nil && !filter.StartDate.IsZero() {
		whereConditions = append(whereConditions, "m.movement_date >= ?")
		args = append(args, filter.StartDate.Format("2006-01-02"))
	}
	if filter.EndDate != nil && !filter.EndDate.IsZero() {
		whereConditions = append(whereConditions, "m.movement_date <= ?")
		args = append(args, filter.EndDate.Format("2006-01-02"))
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Hitung total data
	countQuery := fmt.Sprintf(`SELECT COUNT(m.id) FROM inv_stock_movements m %s`, whereClause)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("gagal menghitung total pergerakan stok: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	query := fmt.Sprintf(`
		SELECT 
			m.id, m.movement_number, m.type, m.movement_date, m.location_id, COALESCE(l.name, '') as location_name,
			m.category_reason, COALESCE(m.reference_number, ''), COALESCE(m.notes, ''),
			m.executed_by, m.executed_by_name, m.created_at
		FROM inv_stock_movements m
		LEFT JOIN inv_locations l ON m.location_id = l.id
		%s
		ORDER BY m.movement_date DESC, m.created_at DESC
		LIMIT ? OFFSET ?`, whereClause)

	queryArgs := append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal query daftar pergerakan stok: %w", err)
	}
	defer rows.Close()

	var list []*domain.StockMovement
	for rows.Next() {
		var m domain.StockMovement
		var movType string
		if err := rows.Scan(
			&m.ID, &m.MovementNumber, &movType, &m.MovementDate, &m.LocationID, &m.LocationName,
			&m.CategoryReason, &m.ReferenceNumber, &m.Notes,
			&m.ExecutedBy, &m.ExecutedByName, &m.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("gagal scan pergerakan stok: %w", err)
		}
		m.Type = domain.StockMovementType(movType)
		m.Items = make([]*domain.StockMovementItem, 0)
		list = append(list, &m)
	}

	if len(list) > 0 {
		var ids []any
		placeholders := make([]string, len(list))
		movementMap := make(map[string]*domain.StockMovement, len(list))
		for i, m := range list {
			ids = append(ids, m.ID)
			placeholders[i] = "?"
			movementMap[m.ID] = m
		}

		queryItems := fmt.Sprintf(`
			SELECT 
				mi.id, mi.movement_id, mi.product_id, COALESCE(p.name, '') as product_name,
				COALESCE(p.sku, '') as product_sku, mi.quantity, COALESCE(mi.notes, ''), mi.created_at
			FROM inv_stock_movement_items mi
			LEFT JOIN inv_products p ON mi.product_id = p.id
			WHERE mi.movement_id IN (%s)
		`, strings.Join(placeholders, ","))

		iRows, err := r.db.QueryContext(ctx, queryItems, ids...)
		if err == nil {
			defer iRows.Close()
			for iRows.Next() {
				var it domain.StockMovementItem
				var movID string
				if err := iRows.Scan(
					&it.ID, &movID, &it.ProductID, &it.ProductName,
					&it.ProductSKU, &it.Quantity, &it.Notes, &it.CreatedAt,
				); err == nil {
					if targetMov, exists := movementMap[movID]; exists {
						targetMov.Items = append(targetMov.Items, &it)
					}
				}
			}
		}
	}

	return list, total, nil
}

// GenerateMovementNumber membuat nomor urut dokumen transaksi baru (IN-YYYYMM-XXXX / OUT-YYYYMM-XXXX).
func (r *mysqlStockMovementRepository) GenerateMovementNumber(ctx context.Context, movementType domain.StockMovementType) (string, error) {
	prefix := "IN"
	if movementType == domain.MovementTypeOut {
		prefix = "OUT"
	}
	now := time.Now().UTC()
	period := now.Format("200601") // YYYYMM

	pattern := fmt.Sprintf("%s-%s-%%", prefix, period)
	query := `SELECT COUNT(id) FROM inv_stock_movements WHERE movement_number LIKE ?`

	var count int
	if err := r.db.QueryRowContext(ctx, query, pattern).Scan(&count); err != nil {
		return "", fmt.Errorf("gagal generate movement number: %w", err)
	}

	newNumber := fmt.Sprintf("%s-%s-%04d", prefix, period, count+1)
	return newNumber, nil
}

// GetStockCardReport mengumpulkan seluruh mutasi persediaan untuk satu produk di satu cabang.
func (r *mysqlStockMovementRepository) GetStockCardReport(
	ctx context.Context,
	productID, locationID string,
	startDate, endDate *time.Time,
) (*domain.StockCardReport, error) {
	// 1. Ambil info produk dan cabang
	var prodName, prodSKU, locName string
	errProd := r.db.QueryRowContext(ctx, `SELECT name, sku FROM inv_products WHERE id = ? LIMIT 1`, productID).Scan(&prodName, &prodSKU)
	if errProd != nil {
		return nil, fmt.Errorf("produk tidak ditemukan: %w", errProd)
	}

	errLoc := r.db.QueryRowContext(ctx, `SELECT name FROM inv_locations WHERE id = ? LIMIT 1`, locationID).Scan(&locName)
	if errLoc != nil {
		return nil, fmt.Errorf("lokasi tidak ditemukan: %w", errLoc)
	}

	var allEntries []*domain.StockCardEntry

	// 2. Query dari inv_stock_movements & inv_stock_movement_items
	queryMovements := `
		SELECT 
			m.movement_date, m.type, m.movement_number, COALESCE(m.reference_number, ''),
			m.category_reason, i.quantity, m.executed_by_name, COALESCE(i.notes, m.notes, '')
		FROM inv_stock_movement_items i
		JOIN inv_stock_movements m ON i.movement_id = m.id
		WHERE i.product_id = ? AND m.location_id = ?`

	mRows, err := r.db.QueryContext(ctx, queryMovements, productID, locationID)
	if err == nil {
		defer mRows.Close()
		for mRows.Next() {
			var dt time.Time
			var movType, docNum, refNum, reason, execName, notes string
			var qty int
			if err := mRows.Scan(&dt, &movType, &docNum, &refNum, &reason, &qty, &execName, &notes); err == nil {
				entry := &domain.StockCardEntry{
					Date:            dt,
					DocumentNumber:  docNum,
					ReferenceNumber: refNum,
					CategoryReason:  reason,
					ExecutedByName:  execName,
					Notes:           notes,
				}
				if movType == "in" {
					entry.MovementType = "stock_in"
					entry.InQuantity = qty
					entry.OutQuantity = 0
				} else {
					entry.MovementType = "stock_out"
					entry.InQuantity = 0
					entry.OutQuantity = qty
				}
				allEntries = append(allEntries, entry)
			}
		}
	}

	// 3. Query dari inv_stock_adjustments (Stock Opname)
	queryAdjustments := `
		SELECT adjustment_date, created_at, difference, reason, adjusted_by_name
		FROM inv_stock_adjustments
		WHERE product_id = ? AND location_id = ?`

	aRows, err := r.db.QueryContext(ctx, queryAdjustments, productID, locationID)
	if err == nil {
		defer aRows.Close()
		for aRows.Next() {
			var adjDate, createdAt time.Time
			var diff int
			var reason, execName string
			if err := aRows.Scan(&adjDate, &createdAt, &diff, &reason, &execName); err == nil {
				dateToUse := adjDate
				if dateToUse.IsZero() {
					dateToUse = createdAt
				}
				entry := &domain.StockCardEntry{
					Date:            dateToUse,
					MovementType:    "opname",
					DocumentNumber:  "OPNAME-" + dateToUse.Format("20060102"),
					ReferenceNumber: "-",
					CategoryReason:  reason,
					ExecutedByName:  execName,
					Notes:           "Penyesuaian Fisik Riil",
				}
				if diff >= 0 {
					entry.InQuantity = diff
					entry.OutQuantity = 0
				} else {
					entry.InQuantity = 0
					entry.OutQuantity = -diff
				}
				allEntries = append(allEntries, entry)
			}
		}
	}

	// 4. Query dari inv_stock_transfers (Transfer Masuk & Keluar)
	queryTransfers := `
		SELECT 
			t.created_at, t.transfer_number, t.from_location_id, t.to_location_id,
			t.status, ti.quantity, COALESCE(t.notes, '')
		FROM inv_stock_transfer_items ti
		JOIN inv_stock_transfers t ON ti.transfer_id = t.id
		WHERE ti.product_id = ? AND (t.from_location_id = ? OR t.to_location_id = ?)
		  AND t.status IN ('in_transit', 'received')`

	tRows, err := r.db.QueryContext(ctx, queryTransfers, productID, locationID, locationID)
	if err == nil {
		defer tRows.Close()
		for tRows.Next() {
			var dt time.Time
			var docNum, fromLoc, toLoc, status, notes string
			var qty int
			if err := tRows.Scan(&dt, &docNum, &fromLoc, &toLoc, &status, &qty, &notes); err == nil {
				entry := &domain.StockCardEntry{
					Date:            dt,
					DocumentNumber:  docNum,
					ReferenceNumber: "-",
					ExecutedByName:  "Sistem Transfer",
					Notes:           notes,
				}
				if fromLoc == locationID {
					entry.MovementType = "transfer_out"
					entry.CategoryReason = "Transfer Keluar ke Cabang Lain"
					entry.InQuantity = 0
					entry.OutQuantity = qty
				} else if toLoc == locationID && status == "received" {
					entry.MovementType = "transfer_in"
					entry.CategoryReason = "Penerimaan Transfer dari Cabang Lain"
					entry.InQuantity = qty
					entry.OutQuantity = 0
				} else {
					continue
				}
				allEntries = append(allEntries, entry)
			}
		}
	}

	// 5. Urutkan seluruh entri kronologis berdasarkan waktu (ascending)
	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].Date.Before(allEntries[j].Date)
	})

	// 6. Filter berdasarkan tanggal jika ada dan hitung opening balance
	runningBalance := 0
	openingBalance := 0
	totalIn := 0
	totalOut := 0

	var filteredEntries []*domain.StockCardEntry

	for _, entry := range allEntries {
		if startDate != nil && entry.Date.Before(*startDate) {
			// Mutasi sebelum startDate masuk ke opening balance
			openingBalance += (entry.InQuantity - entry.OutQuantity)
			runningBalance = openingBalance
			continue
		}
		if endDate != nil && entry.Date.After(*endDate) {
			continue
		}

		runningBalance += (entry.InQuantity - entry.OutQuantity)
		entry.Balance = runningBalance
		totalIn += entry.InQuantity
		totalOut += entry.OutQuantity
		filteredEntries = append(filteredEntries, entry)
	}

	if startDate == nil {
		openingBalance = 0
	}

	report := &domain.StockCardReport{
		ProductID:      productID,
		ProductName:    prodName,
		ProductSKU:     prodSKU,
		LocationID:     locationID,
		LocationName:   locName,
		OpeningBalance: openingBalance,
		TotalIn:        totalIn,
		TotalOut:       totalOut,
		ClosingBalance: runningBalance,
		Entries:        filteredEntries,
	}

	return report, nil
}

// GetValuationReport mengambil ringkasan nilai valuasi persediaan produk per cabang/gudang.
func (r *mysqlStockMovementRepository) GetValuationReport(ctx context.Context, locationID *string) ([]*domain.StockValuationItem, error) {
	whereClause := ""
	var args []any
	if locationID != nil && *locationID != "" {
		whereClause = "WHERE s.location_id = ?"
		args = append(args, *locationID)
	}

	query := fmt.Sprintf(`
		SELECT 
			s.product_id, p.sku, p.name, COALESCE(c.name, 'Umum') as category_name,
			s.location_id, l.name as location_name,
			s.quantity, s.min_stock, p.purchase_price
		FROM inv_stocks s
		JOIN inv_products p ON s.product_id = p.id
		JOIN inv_locations l ON s.location_id = l.id
		LEFT JOIN inv_categories c ON p.category_id = c.id
		%s
		ORDER BY l.name ASC, p.name ASC`, whereClause)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal query valuasi stok: %w", err)
	}
	defer rows.Close()

	var list []*domain.StockValuationItem
	for rows.Next() {
		var it domain.StockValuationItem
		if err := rows.Scan(
			&it.ProductID, &it.ProductSKU, &it.ProductName, &it.CategoryName,
			&it.LocationID, &it.LocationName,
			&it.Quantity, &it.MinStock, &it.BasePrice,
		); err != nil {
			return nil, fmt.Errorf("gagal scan valuasi stok: %w", err)
		}

		it.TotalValuation = float64(it.Quantity) * it.BasePrice
		if it.Quantity == 0 {
			it.Status = "habis"
		} else if it.Quantity <= it.MinStock {
			it.Status = "menipis"
		} else {
			it.Status = "aman"
		}

		list = append(list, &it)
	}

	return list, nil
}
