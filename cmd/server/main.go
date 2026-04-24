package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"everest/inventoryReservation/internal/application"
	"everest/inventoryReservation/internal/domain"
	"everest/inventoryReservation/internal/infrastructure"
	httpiface "everest/inventoryReservation/internal/interface/http"
)

// product is the seed format accepted by the --seed flag: "sku:stock,sku:stock".
type product struct {
	id    string
	stock int
}

func parseSeed(s string) ([]product, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	var out []product
	for _, part := range strings.Split(s, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(kv) != 2 || strings.TrimSpace(kv[0]) == "" {
			return nil, errors.New("seed entries must be <productId>:<stock>")
		}
		n, err := strconv.Atoi(strings.TrimSpace(kv[1]))
		if err != nil || n < 0 {
			return nil, errors.New("seed stock must be a non-negative integer")
		}
		out = append(out, product{id: strings.TrimSpace(kv[0]), stock: n})
	}
	return out, nil
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	seed := flag.String("seed", "sku-1:1", "comma-separated <productId>:<stock> entries")
	hold := flag.Duration("hold", application.DefaultHoldDuration, "reservation hold duration")
	expiryInterval := flag.Duration("expiry-interval", 10*time.Second, "how often to sweep expired reservations (0 disables)")
	flag.Parse()

	products, err := parseSeed(*seed)
	if err != nil {
		log.Fatalf("invalid --seed: %v", err)
	}

	repo := infrastructure.NewInMemoryRepository()
	for _, p := range products {
		repo.AddProduct(domain.ProductInventory{ProductID: p.id, TotalStock: p.stock})
	}
	clk := infrastructure.SystemClock{}
	svc := application.NewReservationService(repo, infrastructure.NewLockManager(), clk, *hold)

	mux := http.NewServeMux()
	mux.Handle("/", httpiface.NewHandler(svc).Routes())
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *expiryInterval > 0 {
		go runExpirySweeper(ctx, svc, *expiryInterval)
	}

	go func() {
		log.Printf("inventory-reservation listening on %s (seed=%q hold=%s)", *addr, *seed, *hold)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func runExpirySweeper(ctx context.Context, svc *application.ReservationService, d time.Duration) {
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			n := svc.ExpireReservations(ctx, now)
			if n > 0 {
				log.Printf("expired %d reservation(s)", n)
			}
		}
	}
}
