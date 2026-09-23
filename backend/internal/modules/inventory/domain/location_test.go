package domain_test

import (
	"testing"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

func TestNewLocation(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		code      string
		locName   string
		locType   domain.LocationType
		address   string
		wantError bool
	}{
		{
			name:      "Valid physical location",
			id:        "018f0a00-0000-7000-0000-000000000001",
			code:      "CAB-BDG",
			locName:   "Cabang Bandung",
			locType:   domain.LocationTypePhysical,
			address:   "Jl. Asia Afrika No. 10",
			wantError: false,
		},
		{
			name:      "Valid online location",
			id:        "018f0a00-0000-7000-0000-000000000002",
			code:      "WH-ONLINE",
			locName:   "Gudang Storefront Online",
			locType:   domain.LocationTypeOnline,
			address:   "Gudang Utama",
			wantError: false,
		},
		{
			name:      "Empty code should fail",
			id:        "018f0a00-0000-7000-0000-000000000003",
			code:      "",
			locName:   "Cabang Surabaya",
			locType:   domain.LocationTypePhysical,
			address:   "Jl. Pemuda No. 1",
			wantError: true,
		},
		{
			name:      "Empty name should fail",
			id:        "018f0a00-0000-7000-0000-000000000004",
			code:      "CAB-SBY",
			locName:   "",
			locType:   domain.LocationTypePhysical,
			address:   "Jl. Pemuda No. 1",
			wantError: true,
		},
		{
			name:      "Invalid type should fail",
			id:        "018f0a00-0000-7000-0000-000000000005",
			code:      "CAB-MLG",
			locName:   "Cabang Malang",
			locType:   domain.LocationType("virtual_other"),
			address:   "Jl. Ijen",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := domain.NewLocation(tt.id, tt.code, tt.locName, tt.locType, tt.address)
			if (err != nil) != tt.wantError {
				t.Fatalf("NewLocation() error = %v, wantError = %v", err, tt.wantError)
			}
			if !tt.wantError {
				if loc.Code != tt.code {
					t.Errorf("expected code %s, got %s", tt.code, loc.Code)
				}
				if !loc.IsActive {
					t.Errorf("new location should be active by default")
				}
			}
		})
	}
}

func TestLocation_UpdateAndStatus(t *testing.T) {
	loc, err := domain.NewLocation(
		"018f0a00-0000-7000-0000-000000000001",
		"CAB-BDG",
		"Cabang Bandung Lama",
		domain.LocationTypePhysical,
		"Alamat Lama",
	)
	if err != nil {
		t.Fatalf("failed to create location: %v", err)
	}

	// Update details
	err = loc.UpdateDetails("CAB-BDG-01", "Cabang Bandung Baru", domain.LocationTypeOnline, "Alamat Baru")
	if err != nil {
		t.Fatalf("UpdateDetails failed: %v", err)
	}
	if loc.Code != "CAB-BDG-01" || loc.Name != "Cabang Bandung Baru" || loc.Type != domain.LocationTypeOnline || loc.Address != "Alamat Baru" {
		t.Errorf("unexpected updated details: %s, %s, %s, %s", loc.Code, loc.Name, loc.Type, loc.Address)
	}

	// Deactivate
	loc.Deactivate()
	if loc.IsActive {
		t.Errorf("expected IsActive to be false after Deactivate()")
	}

	// Activate
	loc.Activate()
	if !loc.IsActive {
		t.Errorf("expected IsActive to be true after Activate()")
	}
}
