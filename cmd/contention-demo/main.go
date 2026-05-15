package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"everest/inventoryReservation/internal/application"
	"everest/inventoryReservation/internal/domain"
	"everest/inventoryReservation/internal/infrastructure"
)

func main() {
	requests := flag.Int("requests", 500, "number of concurrent reservation attempts")
	stock := flag.Int("stock", 1, "available stock for the hot SKU")
	flag.Parse()

	if err := run(context.Background(), *requests, *stock, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, requests, stock int, out io.Writer) error {
	if requests <= 0 || stock < 0 || requests <= stock {
		return fmt.Errorf("requests must be > stock and stock must be >= 0")
	}

	repo := infrastructure.NewInMemoryRepository()
	repo.AddProduct(domain.ProductInventory{ProductID: "sku-hot", TotalStock: stock})
	svc := application.NewReservationService(repo, infrastructure.NewLockManager(), infrastructure.SystemClock{}, 2*time.Minute)

	var successes atomic.Int64
	var outOfStock atomic.Int64
	var otherErrors atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})

	wg.Add(requests)
	for i := 0; i < requests; i++ {
		go func(i int) {
			defer wg.Done()
			<-start
			_, err := svc.ReserveItem(ctx, "sku-hot", fmt.Sprintf("user-%03d", i))
			switch {
			case err == nil:
				successes.Add(1)
			case errors.Is(err, domain.ErrOutOfStock):
				outOfStock.Add(1)
			default:
				otherErrors.Add(1)
			}
		}(i)
	}

	close(start)
	wg.Wait()

	available, err := svc.GetAvailableStock(ctx, "sku-hot")
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "product=sku-hot stock=%d requests=%d successes=%d outOfStock=%d otherErrors=%d available=%d\n", stock, requests, successes.Load(), outOfStock.Load(), otherErrors.Load(), available)

	if int(successes.Load()) != stock || int(outOfStock.Load()) != requests-stock || otherErrors.Load() != 0 || available != 0 {
		return fmt.Errorf("contention invariant failed")
	}
	return nil
}
