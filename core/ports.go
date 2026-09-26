package core

import "context"

type Storage interface {
	CreateAccount(ctx context.Context, acc *Account) error
	GetAccount(ctx context.Context, id int64) (*Account, error)
	GetAccountBalance(ctx context.Context, id int64) (int64, error)
	AddTransaction(ctx context.Context, tx *Transaction) error
	GetTransactionByID(ctx context.Context, id int64) (*Transaction, error)
	GetTransactionByIdempotencyKey(ctx context.Context, idempotencyKey string) (*Transaction, error)
	CreateSnapshot(ctx context.Context, id int64) error
	GetSnapshot(ctx context.Context, id int64) (*TransactionSnapshot, error)
	VerifyIntegrity(ctx context.Context) error
}
