package adapters

import (
	"context"
	"errors"
	"sync"

	c "ledger/core"
)

var (
	ErrNoTransactionFound         = errors.New("no such transaction found")
	ErrAccountDoesNotExist        = errors.New("account ID does not reference an existing account")
	ErrNoSnapshotFound            = errors.New("account ID does not have any snapshots")
	ErrAccountAlreadyExists       = errors.New("account ID already exists")
	ErrTransactionAlreadyExists   = errors.New("transaction ID already exists")
	ErrIdempotencyPayloadConflict = errors.New("idempotency key already exists, but transaction payload does not match")
	ErrIdempotencyKeyConflict     = errors.New("idempotency key already exists")
	ErrInvalidState               = errors.New("invalid state detected - ledger is unbalanced!")
)

type MemoryStorage struct {
	Mu              sync.Mutex
	Accounts        map[int64]*c.Account
	TransactionIDs  map[int64]*c.Transaction
	IdempotencyKeys map[string]*c.Transaction
	Snapshots       map[int64]*c.TransactionSnapshot
	TransactionLog  []*c.Transaction
}

func New() *MemoryStorage {
	return &MemoryStorage{
		Accounts:        make(map[int64]*c.Account),
		TransactionIDs:  make(map[int64]*c.Transaction),
		IdempotencyKeys: make(map[string]*c.Transaction),
		Snapshots:       make(map[int64]*c.TransactionSnapshot),
		TransactionLog:  []*c.Transaction{},
	}
}

func (mem *MemoryStorage) GetAccount(ctx context.Context, id int64) (*c.Account, error) {
	mem.Mu.Lock()
	defer mem.Mu.Unlock()

	got, ok := mem.Accounts[id]
	if !ok {
		return nil, ErrAccountDoesNotExist
	}
	return got, nil
}

func (mem *MemoryStorage) GetAccountBalance(ctx context.Context, id int64) (int64, error) {
	mem.Mu.Lock()
	_, ok := mem.Accounts[id]
	mem.Mu.Unlock()
	if !ok {
		return 0, ErrAccountDoesNotExist
	}

	transactionSnapshot, err := mem.GetSnapshot(context.Background(), id)
	if err != nil && err != ErrNoSnapshotFound {
		return 0, err
	}

	mem.Mu.Lock()
	defer mem.Mu.Unlock()

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
	for i := lastProcessedTransactionID; i < int64(len(mem.TransactionLog)); i++ {

		tx := mem.TransactionLog[i]
		for _, entry := range tx.LedgerEntries {
			if entry.AccountID != id {
				continue
			}

			if entry.EntryDirection == c.DirectionDebit {
				amount, err = c.AddSignedIntSafe(amount, entry.Amount)
			} else {
				amount, err = c.AddSignedIntSafe(amount, -entry.Amount)
			}

			if err != nil {
				return 0, err
			}
		}
	}

	value, err := c.AddSignedIntSafe(snapshotAmount, amount)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func (mem *MemoryStorage) GetSnapshot(ctx context.Context, accountId int64) (*c.TransactionSnapshot, error) {
	mem.Mu.Lock()
	defer mem.Mu.Unlock()

	got, ok := mem.Snapshots[accountId]
	if !ok {
		return nil, ErrNoSnapshotFound
	}
	return got, nil
}

func (mem *MemoryStorage) CreateSnapshot(ctx context.Context, accountId int64) error {
	mem.Mu.Lock()
	defer mem.Mu.Unlock()

	_, ok := mem.Accounts[accountId]
	if !ok {
		return ErrAccountDoesNotExist
	}

	var lastProcessedTransactionID int64
	previousSnapshot, ok := mem.Snapshots[accountId]
	if !ok {
		lastProcessedTransactionID = 0
	} else {
		lastProcessedTransactionID = previousSnapshot.LastProcessedTransactionID
	}

	unprocessedTransactions := mem.TransactionLog[lastProcessedTransactionID:]

	var amount int64
	var err error
	for _, tx := range unprocessedTransactions {
		for _, entry := range tx.LedgerEntries {
			if entry.AccountID != accountId {
				continue
			}

			if entry.EntryDirection == c.DirectionDebit {
				amount, err = c.AddSignedIntSafe(amount, entry.Amount)
			} else {
				amount, err = c.AddSignedIntSafe(amount, -entry.Amount)
			}

			if err != nil {
				return err
			}

			lastProcessedTransactionID = tx.ID
		}
	}

	mem.Snapshots[accountId] = &c.TransactionSnapshot{
		AccountID:                  accountId,
		TotalAmount:                amount,
		LastProcessedTransactionID: lastProcessedTransactionID,
	}
	return nil
}

func (mem *MemoryStorage) CreateAccount(ctx context.Context, acc *c.Account) error {
	mem.Mu.Lock()
	defer mem.Mu.Unlock()

	// if !acc.Type.IsValid() {
	// 	return ErrAccountTypeInvalid
	// }

	_, ok := mem.Accounts[acc.ID]
	if ok {
		return ErrAccountAlreadyExists
	}

	mem.Accounts[acc.ID] = acc
	return nil
}

func (mem *MemoryStorage) GetTransactionByID(ctx context.Context, id int64) (*c.Transaction, error) {
	mem.Mu.Lock()
	defer mem.Mu.Unlock()

	got, ok := mem.TransactionIDs[id]
	if !ok {
		return nil, ErrNoTransactionFound
	}
	return got, nil
}

func (mem *MemoryStorage) GetTransactionByIdempotencyKey(ctx context.Context, idempotencyKey string) (*c.Transaction, error) {
	mem.Mu.Lock()
	defer mem.Mu.Unlock()

	got, ok := mem.IdempotencyKeys[idempotencyKey]
	if !ok {
		return nil, ErrNoTransactionFound
	}
	return got, nil
}

func (mem *MemoryStorage) AddTransaction(ctx context.Context, t *c.Transaction) error {
	mem.Mu.Lock()
	defer mem.Mu.Unlock()

	_, ok := mem.TransactionIDs[t.ID]
	if ok {
		return ErrTransactionAlreadyExists
	}

	existingTx, ok := mem.IdempotencyKeys[t.IdempotencyKey]
	if ok {
		if existingTx.ID != t.ID {
			return ErrIdempotencyPayloadConflict
		}
		return ErrIdempotencyKeyConflict
	}

	// err := t.Validate()
	// if err != nil {
	// 	return err
	// }

	for _, entry := range t.LedgerEntries {
		_, ok := mem.Accounts[entry.AccountID]
		if !ok {
			return ErrAccountDoesNotExist
		}
	}

	mem.IdempotencyKeys[t.IdempotencyKey] = t
	mem.TransactionIDs[t.ID] = t
	mem.TransactionLog = append(mem.TransactionLog, t)
	return nil
}

// TODO: No global locking?
func (mem *MemoryStorage) VerifyIntegrity(ctx context.Context) error {
	var total int64
	for _, acc := range mem.Accounts {
		amount, err := mem.GetAccountBalance(context.Background(), acc.ID)
		if err != nil {
			return err
		}

		total, err = c.AddSignedIntSafe(total, amount)
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
