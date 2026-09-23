package domain

import (
	"errors"
	"strings"
	"time"
)

// Category merepresentasikan entitas kategori produk di modul Inventory.
// Kategori dapat memiliki parent_id untuk mendukung struktur hierarki (pohon sub-kategori).
// Field ImageURL bersifat opsional: jika diisi, etalase dapat menampilkan icon/gambar visual kategori.
type Category struct {
	ID        string
	Name      string
	ParentID  *string // nil jika merupakan kategori tingkat atas (root)
	ImageURL  *string // nil atau kosong jika tidak memiliki gambar/icon
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewCategory adalah constructor domain untuk membuat entitas Category baru.
// Constructor memastikan invariant/aturan validasi kategori terpenuhi sebelum objek dibuat.
func NewCategory(id, name string, parentID, imageURL *string) (*Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("nama kategori tidak boleh kosong")
	}

	// Normalisasi pointer string opsional: jika pointer ada tapi string kosong, jadikan nil
	var cleanParentID *string
	if parentID != nil && strings.TrimSpace(*parentID) != "" {
		trimmed := strings.TrimSpace(*parentID)
		cleanParentID = &trimmed
	}

	var cleanImageURL *string
	if imageURL != nil && strings.TrimSpace(*imageURL) != "" {
		trimmed := strings.TrimSpace(*imageURL)
		cleanImageURL = &trimmed
	}

	now := time.Now()
	return &Category{
		ID:        id,
		Name:      name,
		ParentID:  cleanParentID,
		ImageURL:  cleanImageURL,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// UpdateName memperbarui nama kategori dengan validasi.
func (c *Category) UpdateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("nama kategori tidak boleh kosong")
	}
	c.Name = name
	c.UpdatedAt = time.Now()
	return nil
}

// SetImageURL mengatur atau menghapus gambar kategori.
func (c *Category) SetImageURL(imageURL *string) {
	if imageURL != nil && strings.TrimSpace(*imageURL) != "" {
		trimmed := strings.TrimSpace(*imageURL)
		c.ImageURL = &trimmed
	} else {
		c.ImageURL = nil
	}
	c.UpdatedAt = time.Now()
}

// SetParent mengatur parent category (sub-kategori).
func (c *Category) SetParent(parentID *string) error {
	if parentID != nil && *parentID == c.ID {
		return errors.New("kategori tidak boleh menjadi parent dari dirinya sendiri")
	}
	if parentID != nil && strings.TrimSpace(*parentID) != "" {
		trimmed := strings.TrimSpace(*parentID)
		c.ParentID = &trimmed
	} else {
		c.ParentID = nil
	}
	c.UpdatedAt = time.Now()
	return nil
}
