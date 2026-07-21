package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"couple-mini/backend/internal/httpapi"
	"couple-mini/backend/internal/store"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(127.0.0.1:3306)/couple_mini?charset=utf8mb4&parseTime=true&loc=Local"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
	}

	repo := store.NewMySQLStore(db)
	if err := repo.EnsureSchema(ctx); err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           httpapi.NewRouter(repo),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("couple mini backend listening on http://127.0.0.1:%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
