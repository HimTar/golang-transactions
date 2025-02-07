package main

type Transaction struct {
	Id       int64   `json:"id"`
	Amount   float32 `json:"amount"`
	ParentId *int64  `json:"parent_id"`
	Type     string  `json:"type"`
}

type CreateTransaction struct {
	Amount   *float32 `json:"amount"`
	ParentId *int64   `json:"parent_id"`
	Type     string   `json:"type"`
}

type SumResult struct {
	Sum float32
	err error
}
