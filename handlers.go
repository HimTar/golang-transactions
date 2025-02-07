package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"
)

// GET Handlers

func GetAllTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT * FROM transactions")

	if err != nil {
		log.Println("Error executing query:", err)
		http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
	}

	defer rows.Close()

	var transactions []Transaction

	for rows.Next() {
		var transaction Transaction

		err := rows.Scan(&transaction.Id, &transaction.ParentId, &transaction.Amount, &transaction.Type)
		if err != nil {
			http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
			return
		}

		transactions = append(transactions, transaction)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transactions)
}

func GetTransactionByIdHandler(w http.ResponseWriter, r *http.Request) {
	transactionIDStr := chi.URLParam(r, "id")
	transactionID, err := strconv.ParseInt(transactionIDStr, 10, 64)
	if err != nil {
		log.Println("Error executing query:", err)
		http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
	}

	transaction, err := QueryTransactionById(db, transactionID)
	if err != nil {
		log.Println("Error executing query:", err)
		http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
	}
	if transaction == nil {
		http.Error(w, `{"error": "No transactions found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transaction)
}

func GetTransactionByTypeHandler(w http.ResponseWriter, r *http.Request) {
	transactions, err := QueryTransactionsByType(db, chi.URLParam(r, "type"))
	if err != nil {
		log.Println("Error executing query:", err)
		http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
	}

	if *transactions == nil {
		http.Error(w, `{"error": "No transactions found"}`, http.StatusNotFound)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(transactions)
	}
}

func GetTransactionSumHandler(w http.ResponseWriter, r *http.Request) {
	// Validations
	transactionIDStr := chi.URLParam(r, "id")
	transactionID, err := strconv.ParseInt(transactionIDStr, 10, 64)
	if err != nil {
		log.Println("Error executing query:", err)
		http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
	}

	// check if transaction exists
	transaction, err := QueryTransactionById(db, transactionID)
	if err != nil {
		log.Println("Error executing query:", err)
		http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
	}
	if transaction == nil {
		http.Error(w, `{"error": "No transactions found"}`, http.StatusNotFound)
		return
	}

	// Calculate sum of transaction and its children
	ch := make(chan SumResult)
	wg := sync.WaitGroup{}

	// we start from the parent, register wait group and start the calculation
	wg.Add(1)
	go calculateSumForTransaction(*transaction, ch, &wg)

	// this go routine will wait for all the transactions to be processed and then close the channel
	go func() {
		wg.Wait()
		close((ch))
	}()

	// sum all the values from the channel
	var sum float32
	for res := range ch {
		if res.err != nil {
			http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
			return
		}

		sum += res.Sum
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]float32{"sum": sum})
}

// PUT handlers

func CreateTransactionHandler(w http.ResponseWriter, r *http.Request) {
	// Extract transaction_id from URL
	transactionIDStr := chi.URLParam(r, "id")
	transactionID, err := strconv.ParseInt(transactionIDStr, 10, 64)

	// validations
	if err != nil {
		http.Error(w, `{"error": "Invalid transaction ID"}`, http.StatusBadRequest)
		return
	}

	// Parse JSON request body
	var newTransaction CreateTransaction
	if err := json.NewDecoder(r.Body).Decode(&newTransaction); err != nil {
		http.Error(w, `{"error": "Invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	// Validate required fields
	if newTransaction.Type == "" || newTransaction.Amount == nil {
		http.Error(w, `{"error": "Amount and Type are required"}`, http.StatusBadRequest)
		return
	}

	// Validate transaction id
	ch := make(chan bool)
	counter := 1

	go func(ch chan bool) {
		if transaction, err := QueryTransactionById(db, int64(transactionID)); err != nil || transaction != nil {
			ch <- false
			return
		}
		ch <- true
	}(ch)

	if newTransaction.ParentId != nil {
		counter++
		go func(ch chan bool) {
			if transaction, err := QueryTransactionById(db, int64(*newTransaction.ParentId)); err != nil || transaction == nil {
				ch <- false
				return
			}
			ch <- true
		}(ch)
	}

	errors := false

	// detect errors
	for i := 0; i < counter; i++ {
		result := <-ch
		if !result {
			errors = true
		}
	}
	close(ch)

	// Throw errors if any
	if errors {
		http.Error(w, `{"error":"Invalid Body"}`, http.StatusBadRequest)
		return
	}

	// Insert transaction into database
	query := `INSERT INTO transactions (id, amount, type, parent_id) VALUES ($1, $2, $3, $4)`
	_, err = db.Exec(query, transactionID, newTransaction.Amount, newTransaction.Type, newTransaction.ParentId)
	if err != nil {
		http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
		return
	}

	// Send success response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Transaction inserted successfully"})

}

// Helper functions

// Recursive function to calculate sum of transaction and its children
func calculateSumForTransaction(transaction Transaction, ch chan SumResult, wg *sync.WaitGroup) {
	defer wg.Done()

	// fetch all child transactions
	childTransactions, err := QueryTransactionsByParentId(db, transaction.Id)
	if err != nil {
		log.Println("Error executing query:", err)
		ch <- SumResult{0, err}
		return
	}

	ch <- SumResult{transaction.Amount, nil}

	if len(*childTransactions) != 0 {
		for _, childTransaction := range *childTransactions {
			// call the same function for each child transaction
			// also register wait group
			wg.Add(1)
			go calculateSumForTransaction(childTransaction, ch, wg)
		}
	}

}
