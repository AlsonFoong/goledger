package core

import (
	"errors"
	"math"
	"strings"
	"time"
)

var (
	ErrInvalidIdempotencyKey  = errors.New("idempotency key is invalid")
	ErrInvalidTimestamp       = errors.New("timestamp is invalid")
	ErrInsufficientEntries    = errors.New("transaction must contain at least 2 ledger entries")
	ErrInvalidEntryDirection  = errors.New("ledger entry direction is invalid")
	ErrImbalancedTransaction  = errors.New("total debits must equal total credits")
	ErrOverflow               = errors.New("signed overflow detected")
	ErrUnderflow              = errors.New("signed underflow detected")
	ErrNonPositiveEntryAmount = errors.New("entry amount must be positive")
	// Non-positive account IDs are considered valid
	// ErrNonPositiveEntryAccountID = errors.New("entry account ID must be positive")
)

// Accounts

type Account struct {
	ID   int64
	Type AccountType
	Name string
}

type AccountType uint8

const (
	AccountTypeUnknown AccountType = iota
	AccountTypeAsset
	AccountTypeLiability
	AccountTypeRevenue
	AccountTypeExpense
	AccountTypeEquity
)

func (t AccountType) String() string {
	switch t {
	case AccountTypeAsset:
		return "ASSET"
	case AccountTypeLiability:
		return "LIABILITY"
	case AccountTypeRevenue:
		return "REVENUE"
	case AccountTypeExpense:
		return "EXPENSE"
	case AccountTypeEquity:
		return "EQUITY"
	default:
		return "UNKNOWN"
	}
}

func (t AccountType) IsValid() bool {
	return t >= AccountTypeAsset && t <= AccountTypeEquity
}

// Ledger Entries

type LedgerEntry struct {
	AccountID      int64
	EntryDirection EntryDirection
	Amount         int64
}

type EntryDirection uint8

const (
	DirectionUnknown EntryDirection = iota
	DirectionDebit
	DirectionCredit
)

func (d EntryDirection) String() string {
	switch d {
	case DirectionDebit:
		return "DEBIT"
	case DirectionCredit:
		return "CREDIT"
	default:
		return "UNKNOWN"
	}
}

func (d EntryDirection) IsValid() bool {
	return d >= DirectionDebit && d <= DirectionCredit
}

func (l LedgerEntry) Validate() error {
	// Non-positive account IDs are considered valid
	// if l.AccountID <= 0 {
	// 	return ErrNonPositiveEntryAccountID
	// }

	if !l.EntryDirection.IsValid() {
		return ErrInvalidEntryDirection
	}

	if l.Amount <= 0 {
		return ErrNonPositiveEntryAmount
	}

	return nil
}

// Transactions

type Transaction struct {
	ID             int64
	IdempotencyKey string
	Timestamp      time.Time
	LedgerEntries  []LedgerEntry // Minimum of 2
}

func (t Transaction) Validate() error {
	if strings.TrimSpace(t.IdempotencyKey) == "" {
		return ErrInvalidIdempotencyKey
	}

	if t.Timestamp.IsZero() {
		return ErrInvalidTimestamp
	}

	if len(t.LedgerEntries) < 2 {
		return ErrInsufficientEntries
	}

	var totalDebits int64
	var totalCredits int64

	for _, entry := range t.LedgerEntries {
		var err error

		err = entry.Validate()

		if err != nil {
			return err
		}

		switch entry.EntryDirection {
		case DirectionDebit:
			totalDebits, err = addSignedIntSafe(totalDebits, entry.Amount)
		case DirectionCredit:
			totalCredits, err = addSignedIntSafe(totalCredits, entry.Amount)
		}

		if err != nil {
			return err
		}
	}

	if totalDebits != totalCredits {
		return ErrImbalancedTransaction
	}

	return nil
}

func addSignedIntSafe(a, b int64) (int64, error) {
	// a + b > math.MaxInt64, b > 0
	if b > 0 && a > math.MaxInt64-b {
		return 0, ErrOverflow
	}
	// a + b < math.MinInt64, b < 0
	if b < 0 && a < math.MinInt64-b {
		return 0, ErrUnderflow
	}
	return a + b, nil
}

type TransactionSnapshot struct {
	AccountID                  int64
	TotalAmount                int64
	LastProcessedTransactionID int64
}
