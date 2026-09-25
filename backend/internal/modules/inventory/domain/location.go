package domain

import (
	"errors"
	"time"
)

// LocationType adalah value object yang merepresentasikan tipe sebuah Location.
// Di sistem ini, "gudang online untuk storefront" BUKAN entitas terpisah —
// ia hanyalah Location biasa dengan type = "online".
// Keputusan ini menyederhanakan logic stok karena storefront cukup
// "membaca" satu Location seperti cabang fisik lainnya.
type LocationType string

const (
	LocationTypePhysical LocationType = "physical" // Cabang/toko fisik
	LocationTypeOnline   LocationType = "online"   // Gudang virtual untuk storefront
)

// Location merepresentasikan cabang fisik atau gudang virtual dalam satu instalasi toko.
// Satu instalasi dapat memiliki banyak Location (multi-cabang).
type Location struct {
	ID        string
	Code      string // Kode unik operasional cabang, mis. "CAB-BDG", "WH-ONLINE"
	Name      string
	Type      LocationType
	Address   string
	Latitude  *float64
	Longitude *float64
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewLocation adalah constructor domain untuk membuat Location baru.
func NewLocation(id, code, name string, locType LocationType, address string, lat, lng *float64) (*Location, error) {
	if code == "" {
		return nil, errors.New("kode lokasi tidak boleh kosong")
	}
	if name == "" {
		return nil, errors.New("nama lokasi tidak boleh kosong")
	}
	if locType != LocationTypePhysical && locType != LocationTypeOnline {
		return nil, errors.New("tipe lokasi harus 'physical' atau 'online'")
	}
	if err := validateCoordinates(lat, lng); err != nil {
		return nil, err
	}

	now := time.Now()
	return &Location{
		ID:        id,
		Code:      code,
		Name:      name,
		Type:      locType,
		Address:   address,
		Latitude:  lat,
		Longitude: lng,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// UpdateDetails memperbarui informasi kode, nama, tipe, alamat, dan koordinat lokasi.
func (l *Location) UpdateDetails(code, name string, locType LocationType, address string, lat, lng *float64) error {
	if code == "" {
		return errors.New("kode lokasi tidak boleh kosong")
	}
	if name == "" {
		return errors.New("nama lokasi tidak boleh kosong")
	}
	if locType != LocationTypePhysical && locType != LocationTypeOnline {
		return errors.New("tipe lokasi harus 'physical' atau 'online'")
	}
	if err := validateCoordinates(lat, lng); err != nil {
		return err
	}

	l.Code = code
	l.Name = name
	l.Type = locType
	l.Address = address
	l.Latitude = lat
	l.Longitude = lng
	l.UpdatedAt = time.Now()
	return nil
}

func validateCoordinates(lat, lng *float64) error {
	if (lat != nil && lng == nil) || (lat == nil && lng != nil) {
		return errors.New("latitude dan longitude harus diisi bersamaan atau dikosongkan keduanya")
	}
	if lat != nil {
		if *lat < -90.0 || *lat > 90.0 {
			return errors.New("latitude harus berada di antara -90 dan +90 derajat")
		}
	}
	if lng != nil {
		if *lng < -180.0 || *lng > 180.0 {
			return errors.New("longitude harus berada di antara -180 dan +180 derajat")
		}
	}
	return nil
}

// Deactivate menonaktifkan operasional lokasi (soft delete).
func (l *Location) Deactivate() {
	l.IsActive = false
	l.UpdatedAt = time.Now()
}

// Activate mengaktifkan kembali lokasi yang nonaktif.
func (l *Location) Activate() {
	l.IsActive = true
	l.UpdatedAt = time.Now()
}

