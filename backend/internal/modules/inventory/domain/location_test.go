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
			loc, err := domain.NewLocation(tt.id, tt.code, tt.locName, tt.locType, tt.address, nil, nil)
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

func TestLocation_CoordinatesValidation(t *testing.T) {
	validLat := -6.1914
	validLng := 106.9126
	invalidLat := 95.0
	invalidLng := 190.0

	// Valid coordinates
	loc, err := domain.NewLocation(
		"018f0a00-0000-7000-0000-000000000001",
		"CAB-JKT",
		"Cabang Jakarta",
		domain.LocationTypePhysical,
		"Jakarta",
		&validLat,
		&validLng,
	)
	if err != nil {
		t.Fatalf("expected valid location with coordinates, got: %v", err)
	}
	if loc.Latitude == nil || *loc.Latitude != validLat {
		t.Errorf("expected latitude %f, got %v", validLat, loc.Latitude)
	}

	// Invalid latitude (> 90)
	_, err = domain.NewLocation(
		"018f0a00-0000-7000-0000-000000000002",
		"CAB-ERR",
		"Cabang Error",
		domain.LocationTypePhysical,
		"Error",
		&invalidLat,
		&validLng,
	)
	if err == nil {
		t.Error("expected error for latitude > 90, got nil")
	}

	// Invalid longitude (> 180)
	_, err = domain.NewLocation(
		"018f0a00-0000-7000-0000-000000000003",
		"CAB-ERR2",
		"Cabang Error 2",
		domain.LocationTypePhysical,
		"Error",
		&validLat,
		&invalidLng,
	)
	if err == nil {
		t.Error("expected error for longitude > 180, got nil")
	}

	// Incomplete pair (only lat provided)
	_, err = domain.NewLocation(
		"018f0a00-0000-7000-0000-000000000004",
		"CAB-ERR3",
		"Cabang Error 3",
		domain.LocationTypePhysical,
		"Error",
		&validLat,
		nil,
	)
	if err == nil {
		t.Error("expected error for incomplete coordinates pair, got nil")
	}
}

func TestLocation_UpdateAndStatus(t *testing.T) {
	loc, err := domain.NewLocation(
		"018f0a00-0000-7000-0000-000000000001",
		"CAB-BDG",
		"Cabang Bandung Lama",
		domain.LocationTypePhysical,
		"Alamat Lama",
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create location: %v", err)
	}

	newLat := -6.8905
	newLng := 107.6104

	// Update details with coordinates
	err = loc.UpdateDetails("CAB-BDG-01", "Cabang Bandung Baru", domain.LocationTypeOnline, "Alamat Baru", &newLat, &newLng)
	if err != nil {
		t.Fatalf("UpdateDetails failed: %v", err)
	}
	if loc.Code != "CAB-BDG-01" || loc.Name != "Cabang Bandung Baru" || loc.Type != domain.LocationTypeOnline || loc.Address != "Alamat Baru" {
		t.Errorf("unexpected updated details: %s, %s, %s, %s", loc.Code, loc.Name, loc.Type, loc.Address)
	}
	if loc.Latitude == nil || *loc.Latitude != newLat {
		t.Errorf("expected latitude %f, got %v", newLat, loc.Latitude)
	}
	if loc.Longitude == nil || *loc.Longitude != newLng {
		t.Errorf("expected longitude %f, got %v", newLng, loc.Longitude)
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
