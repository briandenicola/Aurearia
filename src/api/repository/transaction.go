package repository

import "gorm.io/gorm"

type Transaction struct {
	db *gorm.DB
}

func NewTransaction(db *gorm.DB) *Transaction {
	return &Transaction{db: db}
}

func (tx *Transaction) DB() *gorm.DB {
	return tx.db
}
