package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"couple-mini/backend/internal/httpapi"
	"couple-mini/backend/internal/store"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
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
