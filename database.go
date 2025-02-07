package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func CreateTable(db *sql.DB) error {
	query := `
	create table if not EXISTS transactions (
	id BIGINT Primary key,
	parent_id BIGINT,
	amount DOUBLE PRECISION not null,
	type TEXT not null
	);

	CREATE INDEX IF NOT EXISTS transaction_type ON transactions (type);
	CREATE INDEX IF NOT EXISTS transaction_parent_id ON transactions (parent_id) where parent_id is not null;
`

	_, err := db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func CheckTableExists(db *sql.DB, tableName string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS (
                SELECT 1 FROM pg_tables 
                WHERE schemaname = 'public' 
                AND tablename = $1
              );`

	err := db.QueryRow(query, tableName).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func Connect() *sql.DB {
	fmt.Println("Connecting to database...")
	// Connect to database
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to database!")
	fmt.Println("Checking if tables exists...")
	exists, error := CheckTableExists(db, "transactions")

	if error != nil {
		log.Fatal(error)
	}

	if !exists {
		fmt.Println("Tables not found. Creating tables...")
		err := CreateTable(db)

		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Tables created!")
	} else {
		fmt.Println("Tables found!")
	}

	return db
}

// Queries

func QueryTransactionById(db *sql.DB, itemId int64) (*Transaction, error) {
	query := `SELECT * FROM transactions where id = $1`

	// call db
	rows, err := db.Query(query, itemId)
	if err != nil {
		log.Println("Error executing query:", err)
		return nil, err
	}
	defer rows.Close()
	var transaction Transaction

	// scan the result
	if rows.Next() {
		err = rows.Scan(&transaction.Id, &transaction.ParentId, &transaction.Amount, &transaction.Type)
		if err != nil {
			log.Println("Scan error:", err)
			return nil, err
		}
		return &transaction, nil
	}
	return nil, nil
}

func QueryTransactionsByType(db *sql.DB, transactionType string) (*[]Transaction, error) {
	query := `SELECT * FROM transactions where type = $1`

	// call db
	rows, err := db.Query(query, transactionType)
	if err != nil {
		log.Println("Error executing query:", err)
		return nil, err
	}
	defer rows.Close()
	var transactions []Transaction

	for rows.Next() {
		var transaction Transaction

		err = rows.Scan(&transaction.Id, &transaction.ParentId, &transaction.Amount, &transaction.Type)
		if err != nil {
			log.Println("Scan error:", err)
			return nil, err
		}
		transactions = append(transactions, transaction)
	}

	return &transactions, nil
}

func QueryTransactionsByParentId(db *sql.DB, parentId int64) (*[]Transaction, error) {
	query := `SELECT * FROM transactions where parent_id = $1`

	// call db
	rows, err := db.Query(query, parentId)
	if err != nil {
		log.Println("Error executing query:", err)
		return nil, err
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var transaction Transaction

		err = rows.Scan(&transaction.Id, &transaction.ParentId, &transaction.Amount, &transaction.Type)
		if err != nil {
			log.Println("Scan error:", err)
			return nil, err
		}
		transactions = append(transactions, transaction)
	}

	return &transactions, nil
}
