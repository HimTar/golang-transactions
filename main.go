package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

var db *sql.DB

func main() {
	// Fetch postgres connection
	db = Connect()
	defer db.Close()

	server := chi.NewRouter()

	// GET Routes
	server.Get("/transactionservice/transactions/all", GetAllTransactionsHandler)
	server.Get("/transactionservice/transaction/{id}", GetTransactionByIdHandler)
	server.Get("/transactionservice/types/{type}", GetTransactionByTypeHandler)
	server.Get("/transactionservice/sum/{id}", GetTransactionSumHandler)

	// PUT Routes
	server.Put("/transactionservice/transaction/{id}", CreateTransactionHandler)

	server.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "404 Not Found")
	})

	fmt.Println("Starting server on port :8080")
	log.Fatal((http.ListenAndServe(":8080", server)))
}
