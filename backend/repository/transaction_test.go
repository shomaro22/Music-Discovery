package repository

import "testing"

type invalidTransaction struct{}

func (invalidTransaction) transactionMarker() {}

func TestTransactionDBNil(t *testing.T) {
	db, ok := transactionDB(nil)

	if ok {
		t.Fatal("expected invalid transaction")
	}
	if db != nil {
		t.Fatal("expected nil db")
	}
}

func TestTransactionDBInvalidImplementation(t *testing.T) {
	db, ok := transactionDB(invalidTransaction{})

	if ok {
		t.Fatal("expected invalid transaction")
	}
	if db != nil {
		t.Fatal("expected nil db")
	}
}

func TestTransactionDBNilGormTransaction(t *testing.T) {
	var tx *gormTransaction

	db, ok := transactionDB(tx)

	if ok {
		t.Fatal("expected invalid transaction")
	}
	if db != nil {
		t.Fatal("expected nil db")
	}
}
