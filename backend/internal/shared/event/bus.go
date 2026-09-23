package event

import (
	"sync"
)

// HandlerFunc adalah tipe fungsi yang dijalankan saat sebuah event diterima.
// Menggunakan interface{} (atau `any` alias-nya) agar payload bisa tipe apapun —
// handler bertanggung jawab melakukan type assertion ke tipe konkret event yang diharapkan.
type HandlerFunc func(payload any)

// Bus adalah interface kontrak event bus.
// Dengan interface, kita bisa mengganti implementasi (mis. ke Redis pub/sub)
// tanpa mengubah kode modul sama sekali — ini Dependency Inversion Principle.
type Bus interface {
	// Publish mengirim payload ke semua subscriber event bernama eventName.
	Publish(eventName string, payload any)

	// Subscribe mendaftarkan handler untuk merespons event bernama eventName.
	// Handler dipanggil secara sinkron saat Publish() dipanggil.
	Subscribe(eventName string, handler HandlerFunc)
}

// inProcessBus adalah implementasi Bus yang berjalan di dalam proses yang sama (in-process).
// Tidak ada network, tidak ada serialisasi — cocok untuk Modular Monolith.
type inProcessBus struct {
	mu       sync.RWMutex
	handlers map[string][]HandlerFunc
}

// New membuat instance event bus baru yang siap digunakan.
// Dipanggil sekali di main.go, lalu di-inject ke setiap modul.
func New() Bus {
	return &inProcessBus{
		handlers: make(map[string][]HandlerFunc),
	}
}

// Subscribe mendaftarkan handler untuk eventName tertentu.
// Menggunakan RWMutex agar thread-safe: beberapa goroutine bisa membaca (RLock)
// secara bersamaan, tapi hanya satu yang bisa menulis (Lock) pada satu waktu.
func (b *inProcessBus) Subscribe(eventName string, handler HandlerFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventName] = append(b.handlers[eventName], handler)
}

// Publish memanggil semua handler yang terdaftar untuk eventName secara berurutan.
// Menggunakan RLock karena hanya membaca map handlers, tidak mengubahnya.
func (b *inProcessBus) Publish(eventName string, payload any) {
	b.mu.RLock()
	handlers := b.handlers[eventName]
	b.mu.RUnlock()

	for _, h := range handlers {
		h(payload)
	}
}
