package core

import (
	"errors"
	"sync"
)

var (
	ErrNoTransactionFound         = errors.New("no such transaction found")
	ErrAccountDoesNotExist        = errors.New("account ID does not reference an existing account")
	ErrNoSnapshotFound            = errors.New("account ID does not have any snapshots")
	ErrAccountTypeInvalid         = errors.New("account type is invalid")
	ErrAccountAlreadyExists       = errors.New("account ID already exists")
	ErrTransactionAlreadyExists   = errors.New("transaction ID already exists")
	ErrIdempotencyPayloadConflict = errors.New("idempotency key already exists, but transaction payload does not match")
	ErrIdempotencyKeyConflict     = errors.New("idempotency key already exists")
	ErrInvalidState               = errors.New("invalid state detected - ledger is unbalanced!")
)

type Engine struct {
	Mu              sync.Mutex
	Accounts        map[int64]*Account
	TransactionIDs  map[int64]*Transaction
	IdempotencyKeys map[string]*Transaction
	Snapshots       map[int64]*TransactionSnapshot
	TransactionLog  []*Transaction
}

func NewEngine() *Engine {
	return &Engine{
		Accounts:        make(map[int64]*Account),
		TransactionIDs:  make(map[int64]*Transaction),
		IdempotencyKeys: make(map[string]*Transaction),
		Snapshots:       make(map[int64]*TransactionSnapshot),
		TransactionLog:  []*Transaction{},
	}
}

func (e *Engine) GetAccount(id int64) (*Account, error) {
	e.Mu.Lock()
	defer e.Mu.Unlock()

	got, ok := e.Accounts[id]
	if !ok {
		return nil, ErrAccountDoesNotExist
	}
	return got, nil
}

func (e *Engine) GetAccountBalance(id int64) (int64, error) {
	e.Mu.Lock()
	_, ok := e.Accounts[id]
	e.Mu.Unlock()
	if !ok {
		return 0, ErrAccountDoesNotExist
	}

	transactionSnapshot, err := e.GetSnapshot(id)
	if err != nil && err != ErrNoSnapshotFound {
		return 0, err
	}

	e.Mu.Lock()
	defer e.Mu.Unlock()

	var snapshotAmount int64
	var lastProcessedTransactionID int64
	if transactionSnapshot == nil {
		snapshotAmount = 0
		lastProcessedTransactionID = 0
	} else {
		snapshotAmount = transactionSnapshot.TotalAmount
		lastProcessedTransactionID = transactionSnapshot.LastProcessedTransactionID
	}

	var amount int64
	for i := lastProcessedTransactionID; i < int64(len(e.TransactionLog)); i++ {

		tx := e.TransactionLog[i]
		for _, entry := range tx.LedgerEntries {
			if entry.AccountID != id {
				continue
			}

			if entry.EntryDirection == DirectionDebit {
				amount, err = addSignedIntSafe(amount, entry.Amount)
			} else {
				amount, err = addSignedIntSafe(amount, -entry.Amount)
			}

			if err != nil {
				return 0, err
			}
		}
	}

	value, err := addSignedIntSafe(snapshotAmount, amount)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func (e *Engine) GetSnapshot(accountId int64) (*TransactionSnapshot, error) {
	e.Mu.Lock()
	defer e.Mu.Unlock()

	got, ok := e.Snapshots[accountId]
	if !ok {
		return nil, ErrNoSnapshotFound
	}
	return got, nil
}

func (e *Engine) CreateSnapshot(accountId int64) error {
	e.Mu.Lock()
	defer e.Mu.Unlock()

	_, ok := e.Accounts[accountId]
	if !ok {
		return ErrAccountDoesNotExist
	}

	var lastProcessedTransactionID int64
	previousSnapshot, ok := e.Snapshots[accountId]
	if !ok {
		lastProcessedTransactionID = 0
	} else {
		lastProcessedTransactionID = previousSnapshot.LastProcessedTransactionID
	}

	unprocessedTransactions := e.TransactionLog[lastProcessedTransactionID:]

	var amount int64
	var err error
	for _, tx := range unprocessedTransactions {
		for _, entry := range tx.LedgerEntries {
			if entry.AccountID != accountId {
				continue
			}

			if entry.EntryDirection == DirectionDebit {
				amount, err = addSignedIntSafe(amount, entry.Amount)
			} else {
				amount, err = addSignedIntSafe(amount, -entry.Amount)
			}

			if err != nil {
				return err
			}

			lastProcessedTransactionID = tx.ID
		}
	}

	e.Snapshots[accountId] = &TransactionSnapshot{
		AccountID:                  accountId,
		TotalAmount:                amount,
		LastProcessedTransactionID: lastProcessedTransactionID,
	}
	return nil
}

func (e *Engine) CreateAccount(acc *Account) error {
	e.Mu.Lock()
	defer e.Mu.Unlock()

	if !acc.Type.IsValid() {
		return ErrAccountTypeInvalid
	}

	_, ok := e.Accounts[acc.ID]
	if ok {
		return ErrAccountAlreadyExists
	}

	e.Accounts[acc.ID] = acc
	return nil
}

func (e *Engine) GetTransactionByID(id int64) (*Transaction, error) {
	e.Mu.Lock()
	defer e.Mu.Unlock()

	got, ok := e.TransactionIDs[id]
	if !ok {
		return nil, ErrNoTransactionFound
	}
	return got, nil
}

func (e *Engine) GetTransactionByIdempotencyKey(idempotencyKey string) (*Transaction, error) {
	e.Mu.Lock()
	defer e.Mu.Unlock()

	got, ok := e.IdempotencyKeys[idempotencyKey]
	if !ok {
		return nil, ErrNoTransactionFound
	}
	return got, nil
}

func (e *Engine) AddTransaction(t *Transaction) error {
	e.Mu.Lock()
	defer e.Mu.Unlock()

	_, ok := e.TransactionIDs[t.ID]
	if ok {
		return ErrTransactionAlreadyExists
	}

	existingTx, ok := e.IdempotencyKeys[t.IdempotencyKey]
	if ok {
		if existingTx.ID != t.ID {
			return ErrIdempotencyPayloadConflict
		}
		return ErrIdempotencyKeyConflict
	}

	err := t.Validate()
	if err != nil {
		return err
	}

	for _, entry := range t.LedgerEntries {
		_, ok := e.Accounts[entry.AccountID]
		if !ok {
			return ErrAccountDoesNotExist
		}
	}

	e.IdempotencyKeys[t.IdempotencyKey] = t
	e.TransactionIDs[t.ID] = t
	e.TransactionLog = append(e.TransactionLog, t)
	return nil
}

func (e *Engine) VerifyIntegrity() error {
	var total int64
	for _, acc := range e.Accounts {
		amount, err := e.GetAccountBalance(acc.ID)
		if err != nil {
			return err
		}

		total, err = addSignedIntSafe(total, amount)
		if err != nil {
			return err
		}
	}

	if total == 0 {
		return nil
	} else {
		return ErrInvalidState
	}
}
