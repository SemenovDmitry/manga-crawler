// Временный сниппет: показать состояние источников и число манг в БД.
package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
	"os"
)

func main() {
	_ = godotenv.Load()
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getenv("DB_HOST", "localhost"), getenv("DB_PORT", "5432"),
		getenv("DB_USER", "mangauser"), getenv("DB_PASSWORD", "mangapass"), getenv("DB_NAME", "mangadb"))

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, name, base_url,
		full_crawled_at, next_full_crawl_at, full_crawl_interval_secs
		FROM sources ORDER BY id`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	fmt.Println("=== sources ===")
	for rows.Next() {
		var id int
		var name, baseURL string
		var crawled, next sql.NullTime
		var interval int
		if err := rows.Scan(&id, &name, &baseURL, &crawled, &next, &interval); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("id=%d %s (%s)\n  full_crawled_at=%v next_full_crawl_at=%v interval=%ds\n",
			id, name, baseURL,
			formatTime(crawled), formatTime(next), interval)
	}

	counts, err := db.Query(`SELECT s.name, count(m.id) FROM sources s LEFT JOIN manga m ON m.source_id=s.id GROUP BY s.name ORDER BY s.name`)
	if err != nil {
		log.Fatal(err)
	}
	defer counts.Close()
	fmt.Println("=== manga counts ===")
	for counts.Next() {
		var name string
		var cnt int
		if err := counts.Scan(&name, &cnt); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s: %d\n", name, cnt)
	}
}

func formatTime(t sql.NullTime) string {
	if !t.Valid {
		return "NULL"
	}
	return t.Time.Format("02.01.2006 15:04:05")
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
