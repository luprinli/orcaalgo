package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var symbolMap = map[string]string{
	"^_us":   "SPX500",
	"^_usnq": "NAS100",
	"^_uk":   "UK100",
	"^_de":   "GER40",
	"^_jp":   "JPN225",
	"^_hk":   "HK50",
	"^_pl":   "PL20",
}

const PriceScale = 100000.0

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		log.Fatal("usage: seed_candles <path to daily/world/stooq stocks indices>")
	}
	dir := os.Args[1]

	host := envOrDefault("ORCA_DB_HOST", "localhost")
	port := envOrDefault("ORCA_DB_PORT", "5432")
	user := envOrDefault("ORCA_DB_USER", "orca")
	pass := envOrDefault("ORCA_DB_PASSWORD", "orca")
	dbname := envOrDefault("ORCA_DB_NAME", "orca_core")
	ssl := envOrDefault("ORCA_DB_SSLMODE", "disable")

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, dbname, ssl)
	log.Printf("connecting to %s:%s/%s as %s", host, port, dbname, user)

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	total := 0

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("read dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txt") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		f, err := os.Open(path)
		if err != nil {
			log.Printf("skip %s: %v", entry.Name(), err)
			continue
		}

		ticker := strings.TrimSuffix(entry.Name(), ".txt")
		mappedSym, ok := symbolMap[ticker]
		if !ok {
			f.Close()
			continue
		}

		var symID int32
		err = pool.QueryRow(ctx, "SELECT id FROM symbols WHERE ticker = $1 ORDER BY id LIMIT 1", mappedSym).Scan(&symID)
		if err != nil {
			log.Printf("  skip %s (no symbol %s in DB: %v)", ticker, mappedSym, err)
			f.Close()
			continue
		}

		scanner := bufio.NewScanner(f)
		scanner.Scan()

		var rows [][]interface{}
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" || !strings.Contains(line, ",") {
				continue
			}
			parts := strings.Split(line, ",")
			if len(parts) < 9 {
				continue
			}
			ymd := strings.TrimSpace(parts[2])
			if len(ymd) != 8 {
				continue
			}
			y, _ := strconv.Atoi(ymd[0:4])
			m, _ := strconv.Atoi(ymd[4:6])
			d, _ := strconv.Atoi(ymd[6:8])

			open := parsePrice(parts[4])
			high := parsePrice(parts[5])
			low := parsePrice(parts[6])
			close_ := parsePrice(parts[7])
			vol, _ := strconv.ParseFloat(strings.TrimSpace(parts[8]), 64)

			if close_ == 0 {
				continue
			}

			ts := time.Date(y, time.Month(m), d, 16, 0, 0, 0, time.UTC)
			openRaw := int64(math.Round(open * PriceScale))
			highRaw := int64(math.Round(high * PriceScale))
			lowRaw := int64(math.Round(low * PriceScale))
			closeRaw := int64(math.Round(close_ * PriceScale))
			volRaw := int64(math.Round(vol))

			rows = append(rows, []interface{}{ts, symID, "1d", openRaw, highRaw, lowRaw, closeRaw, volRaw, "stooq"})
		}
		f.Close()

		if len(rows) > 0 {
			cols := []string{"time", "symbol_id", "timeframe", "open_raw", "high_raw", "low_raw", "close_raw", "volume", "source"}
			inserted, err := pool.CopyFrom(ctx, pgx.Identifier{"candles"}, cols, pgx.CopyFromRows(rows))
			if err != nil {
				log.Printf("  %s: insert error: %v", mappedSym, err)
				continue
			}
			total += int(inserted)
			log.Printf("  %s: %d candles inserted", mappedSym, inserted)
		}
	}
	log.Printf("Done. %d total candles seeded.", total)
}

func parsePrice(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
