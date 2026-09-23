package license

import "strings"

// License menyimpan informasi modul-modul yang aktif untuk instalasi ini.
// Di MVP, lisensi dibaca dari konfigurasi (env var) saat startup.
// Di masa depan, bisa diganti dengan validasi kriptografis (signed license key).
type License struct {
	enabledModules map[string]bool
}

// New membuat License dari daftar nama modul yang aktif.
// Dipanggil di main.go dengan data dari config.EnabledModules.
func New(enabledModules []string) *License {
	m := make(map[string]bool, len(enabledModules))
	for _, mod := range enabledModules {
		m[strings.ToLower(strings.TrimSpace(mod))] = true
	}
	return &License{enabledModules: m}
}

// Enabled memeriksa apakah modul dengan nama tertentu aktif di instalasi ini.
// HANYA boleh dipanggil di main.go saat proses mounting modul.
// DILARANG dipanggil di dalam handler, use case, atau domain logic.
func (l *License) Enabled(moduleName string) bool {
	return l.enabledModules[strings.ToLower(moduleName)]
}
