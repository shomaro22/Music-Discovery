package repository

import (
	"context"

	"gorm.io/gorm"
)

type ITransaction interface{ transactionMarker() }

type ITransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(tx ITransaction) error) error
}

type gormTransaction struct{ db *gorm.DB }

func (*gormTransaction) transactionMarker() {}

func transactionDB(tx ITransaction) (*gorm.DB, bool) {
	if tx == nil {
		return nil, false
	}

	value, ok := tx.(*gormTransaction)
	if !ok || value == nil || value.db == nil {
		return nil, false
	}

	return value.db, true
}
