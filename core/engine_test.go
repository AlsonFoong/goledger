package core

import (
	"testing"
	"time"
)

func TestEngine(t *testing.T) {
	tests := []struct {
		name      string
		operation func() error
		wantErr   bool
	}{
		{
			name: "creating account with duplicate ID returns error",
			operation: func() error {
				engine := NewEngine()
				createTestAccount(engine, t)
				return engine.CreateAccount(&Account{
					ID:   0,
					Type: AccountTypeAsset,
					Name: "",
				})
			},
			wantErr: true,
		},
		{
			name: "creating valid account succeeds",
			operation: func() error {
				engine := NewEngine()
				return engine.CreateAccount(&Account{
					ID:   0,
					Type: AccountTypeAsset,
					Name: "",
				})
			},
			wantErr: false,
		},
		{
			name: "creating account with invalid type returns error",
			operation: func() error {
				engine := NewEngine()
				return engine.CreateAccount(&Account{
					ID:   0,
					Type: AccountTypeUnknown,
					Name: "",
				})
			},
			wantErr: true,
		},
		{
			name: "finding non-existent account returns error",
			operation: func() error {
				engine := NewEngine()
				_, err := engine.GetAccount(-1)
				return err
			},
			wantErr: true,
		},
		{
			name: "finding existing account succeeds",
			operation: func() error {
				engine := NewEngine()
				engine.CreateAccount(&Account{
					ID:   2,
					Type: AccountTypeAsset,
					Name: "Buildings",
				})
				_, err := engine.GetAccount(2)
				return err
			},
			wantErr: false,
		},
		{
			name: "getting balance of non-existent account returns error",
			operation: func() error {
				engine := NewEngine()
				_, err := engine.GetAccountBalance(0)
				return err
			},
			wantErr: true,
		},
		{
			name: "getting balance of existing account (with no snapshots) succeeds",
			operation: func() error {
				engine := NewEngine()
				createTestAccount(engine, t)
				_, err := engine.GetAccountBalance(0)
				return err
			},
			wantErr: false,
		},
		{
			name: "getting balance of existing account with snapshots succeeds",
			operation: func() error {
				engine := NewEngine()
				createTestAccount(engine, t)
				createTestTransaction(engine, t)
				createTestSnapshot(engine, t)
				_, err := engine.GetAccountBalance(0)
				return err
			},
			wantErr: false,
		},
		{
			name: "creating snapshot of non-existent account returns error",
			operation: func() error {
				engine := NewEngine()
				err := engine.CreateSnapshot(0)
				return err
			},
			wantErr: true,
		},
		{
			name: "creating snapshot of existing account without transactions succeeds",
			operation: func() error {
				engine := NewEngine()
				createTestAccount(engine, t)
				err := engine.CreateSnapshot(0)
				return err
			},
			wantErr: false,
		},
		{
			name: "creating snapshot of existing account with transactions succeeds",
			operation: func() error {
				engine := NewEngine()
				createTestAccount(engine, t)
				createTestTransaction(engine, t)
				err := engine.CreateSnapshot(0)
				return err
			},
			wantErr: false,
		},
		{
			name: "creating snapshot of existing account with no new transactions succeeds",
			operation: func() error {
				engine := NewEngine()
				createTestAccount(engine, t)
				createTestTransaction(engine, t)
				createTestSnapshot(engine, t)
				err := engine.CreateSnapshot(0)
				return err
			},
			wantErr: false,
		},
		{
			name: "getting snapshot of non-existent account returns error",
			operation: func() error {
				engine := NewEngine()
				_, err := engine.GetSnapshot(0)
				return err
			},
			wantErr: true,
		},
		{
			name: "getting snapshot of existing account (without snapshots) returns error",
			operation: func() error {
				engine := NewEngine()
				createTestAccount(engine, t)
				_, err := engine.GetSnapshot(0)
				return err
			},
			wantErr: true,
		},
		{
			name: "getting snapshot of existing account with snapshots succeeds",
			operation: func() error {
				engine := NewEngine()
				createTestAccount(engine, t)
				createTestTransaction(engine, t)
				createTestSnapshot(engine, t)
				_, err := engine.GetSnapshot(0)
				return err
			},
			wantErr: false,
		},
		{
			name: "creating transaction with duplicate ID returns error",
			operation: func() error {
				engine := NewEngine()

				createTestAccount(engine, t)
				createTestTransaction(engine, t)
				return engine.AddTransaction(&Transaction{
					ID:             0,
					IdempotencyKey: "test2",
					Timestamp:      time.Now(),
					LedgerEntries: []LedgerEntry{
						{
							AccountID:      0,
							EntryDirection: DirectionDebit,
							Amount:         500000,
						},
						{
							AccountID:      0,
							EntryDirection: DirectionCredit,
							Amount:         500000,
						},
					},
				})
			},
			wantErr: true,
		},
		{
			name: "creating transaction with duplicate idempotency key returns error",
			operation: func() error {
				engine := NewEngine()

				createTestAccount(engine, t)
				createTestTransaction(engine, t)
				return engine.AddTransaction(&Transaction{
					ID:             2,
					IdempotencyKey: "test",
					Timestamp:      time.Now(),
					LedgerEntries: []LedgerEntry{
						{
							AccountID:      0,
							EntryDirection: DirectionDebit,
							Amount:         500000,
						},
						{
							AccountID:      0,
							EntryDirection: DirectionCredit,
							Amount:         500000,
						},
					},
				})
			},
			wantErr: true,
		},
		{
			name: "creating transaction with uninitialized time returns error",
			operation: func() error {
				engine := NewEngine()

				createTestAccount(engine, t)
				return engine.AddTransaction(&Transaction{
					ID:             1,
					IdempotencyKey: "test",
					Timestamp:      time.Time{},
					LedgerEntries: []LedgerEntry{
						{
							AccountID:      0,
							EntryDirection: DirectionDebit,
							Amount:         500000,
						},
						{
							AccountID:      0,
							EntryDirection: DirectionCredit,
							Amount:         500000,
						},
					},
				})
			},
			wantErr: true,
		},
		{
			name: "creating transaction with no ledger entries returns error",
			operation: func() error {
				engine := NewEngine()
				return engine.AddTransaction(&Transaction{
					ID:             1,
					IdempotencyKey: "test",
					Timestamp:      time.Now(),
					LedgerEntries:  []LedgerEntry{},
				})
			},
			wantErr: true,
		},
		{
			name: "creating transaction with one ledger entry returns error",
			operation: func() error {
				engine := NewEngine()

				createTestAccount(engine, t)
				return engine.AddTransaction(&Transaction{
					ID:             1,
					IdempotencyKey: "test",
					Timestamp:      time.Now(),
					LedgerEntries: []LedgerEntry{
						{
							AccountID:      0,
							EntryDirection: DirectionDebit,
							Amount:         500000,
						},
					},
				})
			},
			wantErr: true,
		},
		{
			name: "creating transaction referencing invalid account ID returns error",
			operation: func() error {
				engine := NewEngine()
				return engine.AddTransaction(&Transaction{
					ID:             1,
					IdempotencyKey: "test1",
					Timestamp:      time.Now(),
					LedgerEntries: []LedgerEntry{
						{
							AccountID:      1,
							EntryDirection: DirectionDebit,
							Amount:         500000,
						},
						{
							AccountID:      1,
							EntryDirection: DirectionCredit,
							Amount:         500000,
						},
					},
				})
			},
			wantErr: true,
		},
		{
			name: "creating transaction with unbalanced debits and credits returns error",
			operation: func() error {
				engine := NewEngine()
				return engine.AddTransaction(&Transaction{
					ID:             1,
					IdempotencyKey: "test",
					Timestamp:      time.Now(),
					LedgerEntries: []LedgerEntry{
						{
							AccountID:      1,
							EntryDirection: DirectionDebit,
							Amount:         500000,
						},
						{
							AccountID:      1,
							EntryDirection: DirectionCredit,
							Amount:         400000,
						},
					},
				})
			},
			wantErr: true,
		},
		{
			name: "creating valid transaction succeeds",
			operation: func() error {
				engine := NewEngine()

				engine.CreateAccount(&Account{
					ID:   0,
					Type: AccountTypeEquity,
					Name: "Owner's Equity",
				})

				engine.CreateAccount(&Account{
					ID:   1,
					Type: AccountTypeAsset,
					Name: "Buildings",
				})

				return engine.AddTransaction(&Transaction{
					ID:             1,
					IdempotencyKey: "test",
					Timestamp:      time.Now(),
					LedgerEntries: []LedgerEntry{
						{
							AccountID:      0,
							EntryDirection: DirectionCredit,
							Amount:         400000,
						},
						{
							AccountID:      0,
							EntryDirection: DirectionCredit,
							Amount:         100000,
						},
						{
							AccountID:      1,
							EntryDirection: DirectionDebit,
							Amount:         500000,
						},
					},
				})
			},
			wantErr: false,
		},
		{
			name: "creating valid transaction and checking ledger integrity succeeds",
			operation: func() error {
				engine := NewEngine()

				var err error
				err = engine.CreateAccount(&Account{
					ID:   0,
					Type: AccountTypeEquity,
					Name: "Owner's Equity",
				})

				if err != nil {
					t.Fatalf("creating test account failed: %v", err)
				}

				err = engine.CreateAccount(&Account{
					ID:   1,
					Type: AccountTypeAsset,
					Name: "Buildings",
				})

				if err != nil {
					t.Fatalf("creating test account failed: %v", err)
				}

				err = engine.AddTransaction(&Transaction{
					ID:             1,
					IdempotencyKey: "test",
					Timestamp:      time.Now(),
					LedgerEntries: []LedgerEntry{
						{
							AccountID:      0,
							EntryDirection: DirectionCredit,
							Amount:         400000,
						},
						{
							AccountID:      0,
							EntryDirection: DirectionCredit,
							Amount:         100000,
						},
						{
							AccountID:      1,
							EntryDirection: DirectionDebit,
							Amount:         500000,
						},
					},
				})

				if err != nil {
					t.Fatalf("creating test transaction failed: %v", err)
				}

				return engine.VerifyIntegrity()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.operation()
			if (got != nil) != tt.wantErr {
				t.Errorf("operation error = %v, want %v", got, tt.wantErr)
			}
		})
	}
}

func createTestAccount(e *Engine, t *testing.T) {
	t.Helper()
	err := e.CreateAccount(&Account{
		ID:   0,
		Type: AccountTypeAsset,
		Name: "TestAccount",
	})

	if err != nil {
		t.Fatalf("createTestAccount failed: %v", err)
	}
}

func createTestTransaction(e *Engine, t *testing.T) {
	t.Helper()
	err := e.AddTransaction(&Transaction{
		ID:             0,
		IdempotencyKey: "test",
		Timestamp:      time.Now(),
		LedgerEntries: []LedgerEntry{
			{
				AccountID:      0,
				EntryDirection: DirectionDebit,
				Amount:         500000,
			},
			{
				AccountID:      0,
				EntryDirection: DirectionCredit,
				Amount:         500000,
			},
		},
	})

	if err != nil {
		t.Fatalf("createTestTransaction failed: %v", err)
	}
}

func createTestSnapshot(e *Engine, t *testing.T) {
	t.Helper()
	err := e.CreateSnapshot(0)

	if err != nil {
		t.Fatalf("creating initial snapshot failed: %v", err)
	}
}
