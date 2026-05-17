package main

import (
	"context"
	"database/sql"
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

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"

	"everest/inventoryReservation/internal/application"
	"everest/inventoryReservation/internal/domain"
	"everest/inventoryReservation/internal/infrastructure"
	pgmigrations "everest/inventoryReservation/internal/infrastructure/postgres/migrations"
	httpiface "everest/inventoryReservation/internal/interface/http"
)

// product is the seed format accepted by the --seed flag: "sku:stock,sku:stock".
type product struct {
	id    string
	stock int
}

// getEnvOr returns the value of the named env var, or fallback if unset/empty.
func getEnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
	// PORT is set by Railway (and most PaaS platforms); fall back to 8080 for
	// local and compose runs. ADDR overrides both when a full address is needed.
	defaultAddr := ":" + getEnvOr("PORT", "8080")
	if a := os.Getenv("ADDR"); a != "" {
		defaultAddr = a
	}
	addr := flag.String("addr", defaultAddr, "listen address")
	// INVENTORY_SEED allows seeding products via env var (useful for Railway /
	// Kubernetes deployments where CLI flags are inconvenient).
	defaultSeed := getEnvOr("INVENTORY_SEED", "sku-1:1")
	seed := flag.String("seed", defaultSeed, "comma-separated <productId>:<stock> entries")
	// INVENTORY_HOLD / INVENTORY_EXPIRY_INTERVAL mirror the CLI flags as env vars.
	defaultHold := application.DefaultHoldDuration
	if h := os.Getenv("INVENTORY_HOLD"); h != "" {
		if parsed, err := time.ParseDuration(h); err == nil {
			defaultHold = parsed
		}
	}
	defaultExpiry := 10 * time.Second
	if e := os.Getenv("INVENTORY_EXPIRY_INTERVAL"); e != "" {
		if parsed, err := time.ParseDuration(e); err == nil {
			defaultExpiry = parsed
		}
	}
	hold := flag.Duration("hold", defaultHold, "reservation hold duration")
	expiryInterval := flag.Duration("expiry-interval", defaultExpiry, "how often to sweep expired reservations (0 disables)")
	flag.Parse()

	products, err := parseSeed(*seed)
	if err != nil {
		log.Fatalf("invalid --seed: %v", err)
	}

	// Open PostgreSQL when DATABASE_URL is set; otherwise fall back to in-memory.
	var db *sql.DB
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		opened, openErr := sql.Open("pgx", dsn)
		if openErr != nil {
			log.Fatalf("open postgres: %v", openErr)
		}
		opened.SetMaxOpenConns(10)
		opened.SetMaxIdleConns(3)
		if err := pgmigrations.RunAll(opened); err != nil {
			log.Fatalf("run migrations: %v", err)
		}
		log.Printf("postgres: migrations applied, using PostgresRepository")
		db = opened
	}
	if db != nil {
		defer db.Close()
	}

	// Choose repository implementation based on whether a DB was opened.
	var repo infrastructure.Repository
	if db != nil {
		repo = infrastructure.NewPostgresRepository(db)
	} else {
		repo = infrastructure.NewInMemoryRepository()
	}
	for _, p := range products {
		repo.AddProduct(domain.ProductInventory{ProductID: p.id, TotalStock: p.stock})
	}
	clk := infrastructure.SystemClock{}
	svc := application.NewReservationService(repo, infrastructure.NewLockManager(), clk, *hold)
	metrics := httpiface.NewMetrics()

	var rdb *redis.Client
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		opts, parseErr := redis.ParseURL(redisURL)
		if parseErr != nil {
			log.Printf("warn: could not parse REDIS_URL: %v", parseErr)
		} else {
			rdb = redis.NewClient(opts)
		}
	}
	if rdb != nil {
		defer rdb.Close()
	}

	mux := http.NewServeMux()
	mux.Handle("/", metrics.Middleware(httpiface.NewHandler(svc).Routes()))
	mux.HandleFunc("GET /healthz", healthzHandler(db, rdb))
	mux.Handle("GET /metrics", metrics.Handler())

	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *expiryInterval > 0 {
		go runExpirySweeper(ctx, svc, metrics, *expiryInterval)
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

// healthzHandler returns a handler that pings Postgres and Redis (when configured)
// and responds 200 ok / 503 degraded accordingly.
// FIX: previously returned 200 unconditionally — load balancers would never remove
// a pod whose DB connection was dead. Now returns 503 when any dependency is down.
func healthzHandler(db *sql.DB, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := make(map[string]string)
		allOK := true

		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			pingErr := db.PingContext(ctx)
			cancel()
			if pingErr != nil {
				checks["postgres"] = "fail: " + pingErr.Error()
				allOK = false
			} else {
				checks["postgres"] = "ok"
			}
		}

		if rdb != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			pingErr := rdb.Ping(ctx).Err()
			cancel()
			if pingErr != nil {
				// FIX: Redis is optional — a failed ping should not flip allOK or
				// return 503.  Returning 503 causes Railway (and any load balancer)
				// to treat the container as unhealthy and restart it endlessly, even
				// though the app is fully functional without Redis.  Report the
				// failure in the JSON body only so operators can see it without
				// taking the service out of rotation.
				checks["redis"] = "degraded: " + pingErr.Error()
			} else {
				checks["redis"] = "ok"
			}
		}

		status := "ok"
		code := http.StatusOK
		if !allOK {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": status,
			"checks": checks,
		})
	}
}

func runExpirySweeper(ctx context.Context, svc *application.ReservationService, metrics *httpiface.Metrics, d time.Duration) {
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			n := svc.ExpireReservations(ctx, now)
			metrics.RecordExpirySweep(n)
			if n > 0 {
				log.Printf("expired %d reservation(s)", n)
			}
		}
	}
}
