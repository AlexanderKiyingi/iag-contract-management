// Command import-mr loads the Construction Department monthly-report workbook
// into the governance/monthly-report schema.
//
//	DATABASE_URL=... go run ./cmd/import-mr --file Inspire_Africa_MR-May2026.xlsx --period 2026-05
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alvor-technologies/iag-contract-management/internal/imports"
	"github.com/alvor-technologies/iag-contract-management/internal/persistence"
)

func main() {
	file := flag.String("file", "", "path to the monthly-report .xlsx workbook")
	periodFlag := flag.String("period", "", "reporting period as YYYY-MM (e.g. 2026-05)")
	flag.Parse()

	if *file == "" || *periodFlag == "" {
		fmt.Fprintln(os.Stderr, "usage: import-mr --file <workbook.xlsx> --period <YYYY-MM>")
		flag.Usage()
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pg, err := persistence.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("postgres connect: %v", err)
	}
	defer pg.Close()

	// Ensure the monthly-report schema (migration 008) is present before import.
	if err := persistence.RunMigrations(ctx, pg.Pool); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	gov := persistence.NewGovStore(pg.Pool)
	res, err := imports.ImportWorkbook(ctx, gov, *file, *periodFlag)
	if err != nil {
		log.Fatalf("import: %v", err)
	}

	log.Printf("imported period %s: contractors=%d contracts=%d reports=%d valuations=%d consultancy=%d",
		*periodFlag, res.Contractors, res.Contracts, res.Reports, res.Valuations, res.Consultancy)
}
