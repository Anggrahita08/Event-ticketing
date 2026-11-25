package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"project-ticketing-golang/internal/model"
	"project-ticketing-golang/internal/repository"
)

type ExpiryWorker struct {
	workerRepo repository.WorkerRepository
}

func NewExpiryWorker(repo repository.WorkerRepository) *ExpiryWorker {
	return &ExpiryWorker{workerRepo: repo}
}

func (w *ExpiryWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("[Worker] Expiry Worker berjalan (interval: 30 detik)")

	for {
		select {
		case <-ctx.Done():
			log.Println("[Worker] Expiry Worker berhenti dengan rapi")
			return
		case <-ticker.C:
			w.processExpiredOrders(ctx)
		}
	}
}

func (w *ExpiryWorker) processExpiredOrders(ctx context.Context) {
	// Gunakan context dengan timeout untuk mencegah proses menggantung
	txCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	tx, err := w.workerRepo.BeginTx(txCtx)
	if err != nil {
		log.Printf("[Worker] Gagal begin tx: %v", err)
		return
	}
	defer tx.Rollback()

	// Ambil maksimal 100 order expired dengan lock
	orders, err := w.workerRepo.GetExpiredOrdersForUpdate(txCtx, tx, 100)
	if err != nil {
		log.Printf("[Worker] Gagal mengambil order expired: %v", err)
		return
	}

	if len(orders) == 0 {
		return // Tidak ada order expired
	}

	successCount := 0
	for _, order := range orders {
		err := w.workerRepo.ExpireOrder(txCtx, tx, order.ID, order.EventID, order.Quantity)
		if err != nil {
			log.Printf("[Worker] Gagal expire order %s: %v", order.ID, err)
			continue
		}

		// Catat Audit Log dengan actor_type = system
		beforeJSON, _ := json.Marshal(order)
		order.Status = "expired"
		afterJSON, _ := json.Marshal(order)

		audit := &model.AuditLog{
			ActorID:    nil,
			ActorType:  "system",
			EntityType: "order",
			EntityID:   order.ID,
			Action:     "expire",
			Before:     beforeJSON,
			After:      afterJSON,
		}
		w.workerRepo.InsertAuditLogTx(txCtx, tx, audit)

		successCount++
	}

	if successCount > 0 {
		if err := tx.Commit(); err != nil {
			log.Printf("[Worker] Gagal commit tx: %v", err)
		} else {
			log.Printf("[Worker] Berhasil expire %d order", successCount)
		}
	}
}
