package main

import (
	"log"
	"net/http"

	"github.com/Hobrus/hobrusmetrics.git/internal/handlers"
	"github.com/Hobrus/hobrusmetrics.git/internal/storage"
)

func main() {
	memStorage := storage.NewMemStorage()

	http.HandleFunc("/update/", handlers.UpdateHandler(memStorage))

	log.Println("Сервер запущен на :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Не удалось запустить сервер: %v\n", err)
	}
}
