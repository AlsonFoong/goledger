package test

import (
	"math"
	"testing"
	"time"

	c "ledger/core"
)

func TestAccount(t *testing.T) {
	acc := c.Account{ID: 14, Type: c.AccountTypeUnknown, Name: "Gopher Trading"}
	if acc.Type.IsValid() {
		t.Error("account type is not valid")
	}
}

func TestLedgerEntry(t *testing.T) {
	entry := c.LedgerEntry{AccountID: 14, EntryDirection: c.DirectionUnknown, Amount: 50}
	if entry.EntryDirection.IsValid() {
		t.Error("ledger entry direction is not valid")
	}
}

func TestTransaction(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		tx      c.Transaction
		wantErr bool
	}{
		{
			name: "empty idempotency key returns error",
			tx: c.Transaction{ID: 1, IdempotencyKey: "", Timestamp: now, LedgerEntries: []c.LedgerEntry{
				{AccountID: 14, EntryDirection: c.DirectionCredit, Amount: 100},
				{AccountID: 14, EntryDirection: c.DirectionDebit, Amount: 100},
			}},
			wantErr: true,
		},
		{
			name: "blank idempotency key returns error",
			tx: c.Transaction{ID: 1, IdempotencyKey: " ", Timestamp: now, LedgerEntries: []c.LedgerEntry{
				{AccountID: 14, EntryDirection: c.DirectionCredit, Amount: 100},
				{AccountID: 14, EntryDirection: c.DirectionDebit, Amount: 100},
			}},
			wantErr: true,
		},
		{
			name: "uninitialized time returns error",
			tx: c.Transaction{ID: 1, IdempotencyKey: " ", Timestamp: time.Time{}, LedgerEntries: []c.LedgerEntry{
				{AccountID: 14, EntryDirection: c.DirectionCredit, Amount: 100},
				{AccountID: 14, EntryDirection: c.DirectionDebit, Amount: 100},
			}},
			wantErr: true,
		},
		{
			name:    "zero entries returns error",
			tx:      c.Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []c.LedgerEntry{}},
			wantErr: true,
		},
		{
			name: "insufficient entries returns error",
			tx: c.Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []c.LedgerEntry{
				{AccountID: 14, EntryDirection: c.DirectionDebit, Amount: 100},
			}},
			wantErr: true,
		},
		{
			name: "invalid entry direction returns error",
			tx: c.Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []c.LedgerEntry{
				{AccountID: 14, EntryDirection: c.DirectionUnknown, Amount: 100},
				{AccountID: 14, EntryDirection: c.DirectionDebit, Amount: 50},
			}},
			wantErr: true,
		},
		{
			name: "imbalanced debits returns error",
			tx: c.Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []c.LedgerEntry{
				{AccountID: 14, EntryDirection: c.DirectionDebit, Amount: 100},
				{AccountID: 14, EntryDirection: c.DirectionDebit, Amount: 50},
			}},
			wantErr: true,
		},
		{
			name: "imbalanced debits and credits returns error",
			tx: c.Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []c.LedgerEntry{
				{AccountID: 14, EntryDirection: c.DirectionDebit, Amount: 50},
				{AccountID: 14, EntryDirection: c.DirectionCredit, Amount: 100},
			}},
			wantErr: true,
		},
		{
			name: "integer overflow during entry summation returns error",
			tx: c.Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []c.LedgerEntry{
				{AccountID: 14, EntryDirection: c.DirectionCredit, Amount: math.MaxInt64},
				{AccountID: 14, EntryDirection: c.DirectionCredit, Amount: 1},
				{AccountID: 14, EntryDirection: c.DirectionDebit, Amount: math.MaxInt64},
				{AccountID: 14, EntryDirection: c.DirectionDebit, Amount: 1},
			}},
			wantErr: true,
		},
		{
			name: "zero amount entries returns error",
			tx: c.Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []c.LedgerEntry{
				{AccountID: 14, EntryDirection: c.DirectionCredit, Amount: 0},
				{AccountID: 14, EntryDirection: c.DirectionDebit, Amount: 0},
			}},
			wantErr: true,
		},
		{
			name: "negative amount entries returns error",
			tx: c.Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []c.LedgerEntry{
				{AccountID: 14, EntryDirection: c.DirectionCredit, Amount: -1},
				{AccountID: 14, EntryDirection: c.DirectionDebit, Amount: -1},
			}},
			wantErr: true,
		},
		{
			name: "zero account ID succeeds",
			tx: c.Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []c.LedgerEntry{
				{AccountID: 0, EntryDirection: c.DirectionCredit, Amount: 1},
				{AccountID: 0, EntryDirection: c.DirectionDebit, Amount: 1},
			}},
			wantErr: false, // Allow this behavior
		},
		{
			name: "negative account ID succeeds",
			tx: c.Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []c.LedgerEntry{
				{AccountID: -1, EntryDirection: c.DirectionCredit, Amount: 1},
				{AccountID: -1, EntryDirection: c.DirectionDebit, Amount: 1},
			}},
			wantErr: false, // Allow this behavior
		},
		{
			name: "valid base-case transaction succeeds",
			tx: c.Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []c.LedgerEntry{
				{AccountID: 14, EntryDirection: c.DirectionCredit, Amount: 100},
				{AccountID: 14, EntryDirection: c.DirectionDebit, Amount: 100},
			}},
			wantErr: false,
		},
		{
			name: "valid split transaction succeeds",
			tx: c.Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []c.LedgerEntry{
				{AccountID: 14, EntryDirection: c.DirectionCredit, Amount: 100},
				{AccountID: 14, EntryDirection: c.DirectionDebit, Amount: 50},
				{AccountID: 14, EntryDirection: c.DirectionDebit, Amount: 50},
			}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tx.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Transaction.Validate() error = %v, want = %v", err, tt.wantErr)
			}
		})
	}
}

func TestAddSignedIntSafe(t *testing.T) {
	tests := []struct {
		name    string
		a       int64
		b       int64
		want    int64
		wantErr bool
	}{
		{
			name:    "overflow returns error",
			a:       math.MaxInt64,
			b:       1,
			want:    0,
			wantErr: true,
		},
		{
			name:    "underflow returns error",
			a:       math.MinInt64,
			b:       -1,
			want:    0,
			wantErr: true,
		},
		{
			name:    "valid addition near MinInt64 succeeds",
			a:       math.MinInt64,
			b:       10,
			want:    math.MinInt64 + 10,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.AddSignedIntSafe(tt.a, tt.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("addSignedIntSafe() failed: %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("addSignedIntSafe() = %v, want %v", got, tt.want)
			}
		})
	}
}
