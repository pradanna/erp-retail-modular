package uid

import "github.com/google/uuid"

// New menghasilkan UUIDv7 baru yang siap dipakai sebagai primary key entity.
//
// Mengapa UUIDv7 (bukan auto-increment atau UUIDv4)?
// - AUTO-INCREMENT (1, 2, 3...): mudah ditebak, bisa enumerate resource lewat URL
// - UUIDv4 (random): tidak bisa di-sort, fragmentasi B-Tree index di database
// - UUIDv7 (timestamp + random): terurut alami (sortable by time), ramah index B-Tree,
//   dan aman dari tebakan karena mengandung bagian random
//
// Contoh UUIDv7: 0190c8f0-1234-7abc-9def-0123456789ab
//                ^^^^^^^^^^^^ bagian timestamp unix-ms
func New() string {
	id, err := uuid.NewV7()
	if err != nil {
		// Fallback ke UUIDv4 jika sistem tidak bisa membaca clock monotonic.
		// Dalam praktik, ini tidak akan pernah terjadi di server normal.
		return uuid.New().String()
	}
	return id.String()
}
