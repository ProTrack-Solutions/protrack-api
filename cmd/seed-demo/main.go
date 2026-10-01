// cmd/seed-demo/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/ProTrack-Solutions/protrack-api/internal/database"
	"github.com/ProTrack-Solutions/protrack-api/internal/demo"
)

// Recria a empresa de demonstração (apaga os dados dela e gera de novo).
// Não mexe em nenhuma outra empresa.
func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.NewConnect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	start := time.Now()
	result, err := demo.Seed(ctx, db.Pool)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("empresa demo recriada em %s (banco %s@%s)\n", time.Since(start).Round(time.Millisecond), cfg.DBName, cfg.DBHost)
	fmt.Printf("  produtos: %d | clientes: %d | vendas: %d | itens: %d\n", result.Products, result.Customers, result.Sales, result.SaleItems)
	fmt.Printf("  parcelas: %d | pagamentos: %d | contas a pagar: %d\n", result.Receivables, result.PaymentsHistories, result.BillsPayable)
	fmt.Printf("  login: %s (company_id %s)\n", demo.UserEmail, demo.CompanyID)
}
