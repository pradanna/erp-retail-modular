package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/erp-retail/backend/pkg/uid"
)

// TransferStatus mendefinisikan status siklus hidup dokumen mutasi stok antar cabang.
type TransferStatus string

const (
	TransferStatusPendingApproval TransferStatus = "pending_approval" // Diajukan oleh cabang, menunggu persetujuan atasan
	TransferStatusApproved        TransferStatus = "approved"         // Disetujui atasan, siap dikemas dan dikirim
	TransferStatusInTransit       TransferStatus = "in_transit"       // Barang dimuat ke truk, stok asal berkurang, dalam perjalanan
	TransferStatusReceived        TransferStatus = "received"         // Barang tiba di cabang tujuan, diverifikasi, stok tujuan bertambah
	TransferStatusRejected        TransferStatus = "rejected"         // Ditolak oleh atasan
	TransferStatusCancelled       TransferStatus = "cancelled"        // Dibatalkan oleh pemohon sebelum disetujui
)

var (
	ErrInvalidTransferID       = errors.New("ID transfer tidak valid")
	ErrInvalidTransferNumber   = errors.New("nomor transfer tidak valid")
	ErrSameLocationTransfer    = errors.New("lokasi asal dan tujuan transfer tidak boleh sama")
	ErrEmptyTransferItems      = errors.New("dokumen transfer harus memiliki setidaknya 1 item barang")
	ErrInvalidTransferQuantity = errors.New("kuantitas barang yang ditransfer harus lebih dari 0")
	ErrDuplicateTransferItem   = errors.New("produk yang sama tidak boleh didaftarkan ganda dalam satu dokumen transfer")
	ErrIllegalStatusTransition = errors.New("transisi status mutasi stok tidak diizinkan")
	ErrTransferNotFound        = errors.New("dokumen mutasi stok tidak ditemukan")
)

// StockTransferItem adalah child entity yang merepresentasikan satu baris barang dalam dokumen mutasi.
type StockTransferItem struct {
	ID               string   // Primary Key (UUIDv7)
	TransferID       string   // Foreign Key ke inv_stock_transfers(id)
	ProductID        string   // Foreign Key ke inv_products(id)
	Quantity         int      // Jumlah barang yang dikirim
	ReceivedQuantity int      // Jumlah barang yang diterima di cabang tujuan
	SerialUnitIDs    []string // Daftar ID serial unit jika barang memiliki nomor seri
	CreatedAt        time.Time
}

// StockTransferItemInput adalah DTO input saat membuat permohonan mutasi stok baru.
type StockTransferItemInput struct {
	ProductID     string
	Quantity      int
	SerialUnitIDs []string
}

// StockTransfer adalah Aggregate Root untuk pengelolaan mutasi stok antar cabang.
type StockTransfer struct {
	ID              string
	TransferNumber  string
	FromLocationID  string
	ToLocationID    string
	Status          TransferStatus
	Notes           string
	RejectionReason string
	RequestedBy     string
	ApprovedBy      *string
	ReceivedBy      *string
	Items           []*StockTransferItem
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewStockTransfer membuat Aggregate Root StockTransfer baru dengan validasi invariant ketat.
func NewStockTransfer(
	id, transferNumber, fromLocationID, toLocationID, notes, requestedBy string,
	items []StockTransferItemInput,
) (*StockTransfer, error) {
	if id == "" {
		return nil, ErrInvalidTransferID
	}
	if strings.TrimSpace(transferNumber) == "" {
		return nil, ErrInvalidTransferNumber
	}
	if fromLocationID == "" || toLocationID == "" {
		return nil, ErrInvalidLocationID
	}
	if fromLocationID == toLocationID {
		return nil, ErrSameLocationTransfer
	}
	if strings.TrimSpace(requestedBy) == "" {
		return nil, errors.New("pemohon transfer (requested_by) wajib diisi")
	}
	if len(items) == 0 {
		return nil, ErrEmptyTransferItems
	}

	now := time.Now().UTC()
	seenProducts := make(map[string]bool)
	transferItems := make([]*StockTransferItem, 0, len(items))

	for _, it := range items {
		prodID := strings.TrimSpace(it.ProductID)
		if prodID == "" {
			return nil, ErrInvalidProductID
		}
		if it.Quantity <= 0 {
			return nil, ErrInvalidTransferQuantity
		}
		if seenProducts[prodID] {
			return nil, ErrDuplicateTransferItem
		}
		seenProducts[prodID] = true

		// Jika barang menyertakan nomor seri, pastikan jumlah serial cocok dengan quantity
		if len(it.SerialUnitIDs) > 0 && len(it.SerialUnitIDs) != it.Quantity {
			return nil, errors.New("jumlah nomor seri yang dipilih harus sama dengan kuantitas yang ditransfer")
		}

		transferItems = append(transferItems, &StockTransferItem{
			ID:               uid.New(),
			TransferID:       id,
			ProductID:        prodID,
			Quantity:         it.Quantity,
			ReceivedQuantity: 0,
			SerialUnitIDs:    it.SerialUnitIDs,
			CreatedAt:        now,
		})
	}

	return &StockTransfer{
		ID:              id,
		TransferNumber:  strings.TrimSpace(transferNumber),
		FromLocationID:  fromLocationID,
		ToLocationID:    toLocationID,
		Status:          TransferStatusPendingApproval,
		Notes:           strings.TrimSpace(notes),
		RejectionReason: "",
		RequestedBy:     strings.TrimSpace(requestedBy),
		ApprovedBy:      nil,
		ReceivedBy:      nil,
		Items:           transferItems,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// Approve mengubah status menjadi 'approved' setelah diperiksa oleh atasan (Owner/Superadmin).
func (t *StockTransfer) Approve(approvedBy string) error {
	if t.Status != TransferStatusPendingApproval {
		return ErrIllegalStatusTransition
	}
	approver := strings.TrimSpace(approvedBy)
	if approver == "" {
		return errors.New("penyetuju transfer (approved_by) wajib diisi")
	}

	t.Status = TransferStatusApproved
	t.ApprovedBy = &approver
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// Reject menolak permohonan mutasi stok dan mencatat alasannya.
func (t *StockTransfer) Reject(rejectedBy, reason string) error {
	if t.Status != TransferStatusPendingApproval {
		return ErrIllegalStatusTransition
	}
	rejecter := strings.TrimSpace(rejectedBy)
	if rejecter == "" {
		return errors.New("penolak transfer (rejected_by) wajib diisi")
	}

	t.Status = TransferStatusRejected
	t.RejectionReason = strings.TrimSpace(reason)
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// Ship menandai barang telah dimuat ke ekspedisi/truk pengiriman dan dalam perjalanan ('in_transit').
func (t *StockTransfer) Ship() error {
	if t.Status != TransferStatusApproved {
		return ErrIllegalStatusTransition
	}

	t.Status = TransferStatusInTransit
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// Receive menandai barang telah tiba di cabang tujuan dan diverifikasi oleh admin penerima.
func (t *StockTransfer) Receive(receivedBy string) error {
	if t.Status != TransferStatusInTransit {
		return ErrIllegalStatusTransition
	}
	receiver := strings.TrimSpace(receivedBy)
	if receiver == "" {
		return errors.New("penerima transfer (received_by) wajib diisi")
	}

	// MVP mendukung 100% full receipt
	for _, it := range t.Items {
		it.ReceivedQuantity = it.Quantity
	}

	t.Status = TransferStatusReceived
	t.ReceivedBy = &receiver
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// Cancel membatalkan transfer oleh pemohon sebelum disetujui atasan.
func (t *StockTransfer) Cancel() error {
	if t.Status != TransferStatusPendingApproval {
		return ErrIllegalStatusTransition
	}

	t.Status = TransferStatusCancelled
	t.UpdatedAt = time.Now().UTC()
	return nil
}
