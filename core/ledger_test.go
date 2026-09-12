package core

import (
	"math"
	"testing"
	"time"
)

func TestAccount(t *testing.T) {
	acc := Account{14, AccountTypeUnknown, "Gopher Trading"}
	if acc.Type.IsValid() {
		t.Error("account type is not valid")
	}
}

func TestLedgerEntry(t *testing.T) {
	entry := LedgerEntry{14, DirectionUnknown, 50}
	if entry.EntryDirection.IsValid() {
		t.Error("ledger entry direction is not valid")
	}
}

func TestTransaction(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		tx      Transaction
		wantErr bool
	}{
		{
			name: "empty idempotency key returns error",
			tx: Transaction{ID: 1, IdempotencyKey: "", Timestamp: now, LedgerEntries: []LedgerEntry{
				{AccountID: 14, EntryDirection: DirectionCredit, Amount: 100},
				{AccountID: 14, EntryDirection: DirectionDebit, Amount: 100},
			}},
			wantErr: true,
		},
		{
			name: "blank idempotency key returns error",
			tx: Transaction{ID: 1, IdempotencyKey: " ", Timestamp: now, LedgerEntries: []LedgerEntry{
				{AccountID: 14, EntryDirection: DirectionCredit, Amount: 100},
				{AccountID: 14, EntryDirection: DirectionDebit, Amount: 100},
			}},
			wantErr: true,
		},
		{
			name: "uninitialized time returns error",
			tx: Transaction{ID: 1, IdempotencyKey: " ", Timestamp: time.Time{}, LedgerEntries: []LedgerEntry{
				{AccountID: 14, EntryDirection: DirectionCredit, Amount: 100},
				{AccountID: 14, EntryDirection: DirectionDebit, Amount: 100},
			}},
			wantErr: true,
		},
		{
			name:    "zero entries returns error",
			tx:      Transaction{1, "tt", now, []LedgerEntry{}},
			wantErr: true,
		},
		{
			name: "insufficient entries returns error",
			tx: Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []LedgerEntry{
				{14, DirectionDebit, 100},
			}},
			wantErr: true,
		},
		{
			name: "invalid entry direction returns error",
			tx: Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []LedgerEntry{
				{AccountID: 14, EntryDirection: DirectionUnknown, Amount: 100},
				{AccountID: 14, EntryDirection: DirectionDebit, Amount: 50},
			}},
			wantErr: true,
		},
		{
			name: "imbalanced debits returns error",
			tx: Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []LedgerEntry{
				{AccountID: 14, EntryDirection: DirectionDebit, Amount: 100},
				{AccountID: 14, EntryDirection: DirectionDebit, Amount: 50},
			}},
			wantErr: true,
		},
		{
			name: "imbalanced debits and credits returns error",
			tx: Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []LedgerEntry{
				{AccountID: 14, EntryDirection: DirectionDebit, Amount: 50},
				{AccountID: 14, EntryDirection: DirectionCredit, Amount: 100},
			}},
			wantErr: true,
		},
		{
			name: "integer overflow during entry summation returns error",
			tx: Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []LedgerEntry{
				{AccountID: 14, EntryDirection: DirectionCredit, Amount: math.MaxInt64},
				{AccountID: 14, EntryDirection: DirectionCredit, Amount: 1},
				{AccountID: 14, EntryDirection: DirectionDebit, Amount: math.MaxInt64},
				{AccountID: 14, EntryDirection: DirectionDebit, Amount: 1},
			}},
			wantErr: true,
		},
		{
			name: "zero amount entries returns error",
			tx: Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []LedgerEntry{
				{AccountID: 14, EntryDirection: DirectionCredit, Amount: 0},
				{AccountID: 14, EntryDirection: DirectionDebit, Amount: 0},
			}},
			wantErr: true,
		},
		{
			name: "negative amount entries returns error",
			tx: Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []LedgerEntry{
				{AccountID: 14, EntryDirection: DirectionCredit, Amount: -1},
				{AccountID: 14, EntryDirection: DirectionDebit, Amount: -1},
			}},
			wantErr: true,
		},
		{
			name: "zero account ID succeeds",
			tx: Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []LedgerEntry{
				{AccountID: 0, EntryDirection: DirectionCredit, Amount: 1},
				{AccountID: 0, EntryDirection: DirectionDebit, Amount: 1},
			}},
			wantErr: false, // Allow this behavior
		},
		{
			name: "negative account ID succeeds",
			tx: Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []LedgerEntry{
				{AccountID: -1, EntryDirection: DirectionCredit, Amount: 1},
				{AccountID: -1, EntryDirection: DirectionDebit, Amount: 1},
			}},
			wantErr: false, // Allow this behavior
		},
		{
			name: "valid base-case transaction succeeds",
			tx: Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []LedgerEntry{
				{AccountID: 14, EntryDirection: DirectionCredit, Amount: 100},
				{AccountID: 14, EntryDirection: DirectionDebit, Amount: 100},
			}},
			wantErr: false,
		},
		{
			name: "valid split transaction succeeds",
			tx: Transaction{ID: 1, IdempotencyKey: "tt", Timestamp: now, LedgerEntries: []LedgerEntry{
				{AccountID: 14, EntryDirection: DirectionCredit, Amount: 100},
				{AccountID: 14, EntryDirection: DirectionDebit, Amount: 50},
				{AccountID: 14, EntryDirection: DirectionDebit, Amount: 50},
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
			got, err := addSignedIntSafe(tt.a, tt.b)
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
