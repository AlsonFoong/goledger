package core

import (
	"context"
	"errors"
)

var (
	ErrNilValue           = errors.New("value passed in cannot be nil")
	ErrAccountTypeInvalid = errors.New("account type is invalid")
)

type Engine struct {
	storage Storage
}

func NewEngine(s Storage) (*Engine, error) {
	if s == nil {
		return nil, ErrNilValue
	}
	return &Engine{storage: s}, nil
}

func (e *Engine) GetAccount(ctx context.Context, id int64) (*Account, error) {
	acc, err := e.storage.GetAccount(ctx, id)
	return acc, err
}

func (e *Engine) GetAccountBalance(ctx context.Context, id int64) (int64, error) {
	val, err := e.storage.GetAccountBalance(ctx, id)
	return val, err
}

func (e *Engine) GetSnapshot(ctx context.Context, accountId int64) (*TransactionSnapshot, error) {
	txSnapshot, err := e.storage.GetSnapshot(ctx, accountId)
	return txSnapshot, err
}

func (e *Engine) CreateSnapshot(ctx context.Context, accountId int64) error {
	err := e.storage.CreateSnapshot(ctx, accountId)
	return err
}

func (e *Engine) CreateAccount(ctx context.Context, acc *Account) error {
	if acc == nil {
		return ErrNilValue
	}

	if !acc.Type.IsValid() {
		return ErrAccountTypeInvalid
	}

	err := e.storage.CreateAccount(ctx, acc)
	return err
}

func (e *Engine) GetTransactionByID(ctx context.Context, id int64) (*Transaction, error) {
	tx, err := e.storage.GetTransactionByID(ctx, id)
	return tx, err
}

func (e *Engine) GetTransactionByIdempotencyKey(ctx context.Context, idempotencyKey string) (*Transaction, error) {
	tx, err := e.storage.GetTransactionByIdempotencyKey(ctx, idempotencyKey)
	return tx, err
}

func (e *Engine) AddTransaction(ctx context.Context, t *Transaction) error {
	if t == nil {
		return ErrNilValue
	}

	err := t.Validate()
	if err != nil {
		return err
	}

	err = e.storage.AddTransaction(ctx, t)
	return err
}

func (e *Engine) VerifyIntegrity(ctx context.Context) error {
	err := e.storage.VerifyIntegrity(ctx)
	return err
}
