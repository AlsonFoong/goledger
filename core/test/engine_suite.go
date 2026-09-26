package test

import (
	"context"
	"testing"
	"time"

	c "ledger/core"
)

func RunEngineTests(t *testing.T, storageFactory func(t *testing.T) c.Storage) {
	tests := []struct {
		name      string
		operation func(e *c.Engine) error
		wantErr   bool
	}{
		{
			name: "creating account with duplicate ID returns error",
			operation: func(e *c.Engine) error {
				CreateTestAccount(e, t)
				return e.CreateAccount(context.TODO(), &c.Account{
					ID:   0,
					Type: c.AccountTypeAsset,
					Name: "",
				})
			},
			wantErr: true,
		},
		{
			name: "creating valid account succeeds",
			operation: func(e *c.Engine) error {
				return e.CreateAccount(context.TODO(), &c.Account{
					ID:   0,
					Type: c.AccountTypeAsset,
					Name: "",
				})
			},
			wantErr: false,
		},
		{
			name: "creating account with invalid type returns error",
			operation: func(e *c.Engine) error {
				return e.CreateAccount(context.TODO(), &c.Account{
					ID:   0,
					Type: c.AccountTypeUnknown,
					Name: "",
				})
			},
			wantErr: true,
		},
		{
			name: "finding non-existent account returns error",
			operation: func(e *c.Engine) error {
				_, err := e.GetAccount(context.TODO(), -1)
				return err
			},
			wantErr: true,
		},
		{
			name: "finding existing account succeeds",
			operation: func(e *c.Engine) error {
				e.CreateAccount(context.TODO(), &c.Account{
					ID:   2,
					Type: c.AccountTypeAsset,
					Name: "Buildings",
				})
				_, err := e.GetAccount(context.TODO(), 2)
				return err
			},
			wantErr: false,
		},
		{
			name: "getting balance of non-existent account returns error",
			operation: func(e *c.Engine) error {
				_, err := e.GetAccountBalance(context.TODO(), 0)
				return err
			},
			wantErr: true,
		},
		{
			name: "getting balance of existing account (with no snapshots) succeeds",
			operation: func(e *c.Engine) error {
				CreateTestAccount(e, t)
				_, err := e.GetAccountBalance(context.TODO(), 0)
				return err
			},
			wantErr: false,
		},
		{
			name: "getting balance of existing account with snapshots succeeds",
			operation: func(e *c.Engine) error {
				CreateTestAccount(e, t)
				CreateTestTransaction(e, t)
				CreateTestSnapshot(e, t)
				_, err := e.GetAccountBalance(context.TODO(), 0)
				return err
			},
			wantErr: false,
		},
		{
			name: "creating snapshot of non-existent account returns error",
			operation: func(e *c.Engine) error {
				err := e.CreateSnapshot(context.TODO(), 0)
				return err
			},
			wantErr: true,
		},
		{
			name: "creating snapshot of existing account without transactions succeeds",
			operation: func(e *c.Engine) error {
				CreateTestAccount(e, t)
				err := e.CreateSnapshot(context.TODO(), 0)
				return err
			},
			wantErr: false,
		},
		{
			name: "creating snapshot of existing account with transactions succeeds",
			operation: func(e *c.Engine) error {
				CreateTestAccount(e, t)
				CreateTestTransaction(e, t)
				err := e.CreateSnapshot(context.TODO(), 0)
				return err
			},
			wantErr: false,
		},
		{
			name: "creating snapshot of existing account with no new transactions succeeds",
			operation: func(e *c.Engine) error {
				CreateTestAccount(e, t)
				CreateTestTransaction(e, t)
				CreateTestSnapshot(e, t)
				err := e.CreateSnapshot(context.TODO(), 0)
				return err
			},
			wantErr: false,
		},
		{
			name: "getting snapshot of non-existent account returns error",
			operation: func(e *c.Engine) error {
				_, err := e.GetSnapshot(context.TODO(), 0)
				return err
			},
			wantErr: true,
		},
		{
			name: "getting snapshot of existing account (without snapshots) returns error",
			operation: func(e *c.Engine) error {
				CreateTestAccount(e, t)
				_, err := e.GetSnapshot(context.TODO(), 0)
				return err
			},
			wantErr: true,
		},
		{
			name: "getting snapshot of existing account with snapshots succeeds",
			operation: func(e *c.Engine) error {
				CreateTestAccount(e, t)
				CreateTestTransaction(e, t)
				CreateTestSnapshot(e, t)
				_, err := e.GetSnapshot(context.TODO(), 0)
				return err
			},
			wantErr: false,
		},
		{
			name: "creating transaction with duplicate ID returns error",
			operation: func(e *c.Engine) error {

				CreateTestAccount(e, t)
				CreateTestTransaction(e, t)
				return e.AddTransaction(context.TODO(), &c.Transaction{
					ID:             0,
					IdempotencyKey: "test2",
					Timestamp:      time.Now(),
					LedgerEntries: []c.LedgerEntry{
						{
							AccountID:      0,
							EntryDirection: c.DirectionDebit,
							Amount:         500000,
						},
						{
							AccountID:      0,
							EntryDirection: c.DirectionCredit,
							Amount:         500000,
						},
					},
				})
			},
			wantErr: true,
		},
		{
			name: "creating transaction with duplicate idempotency key returns error",
			operation: func(e *c.Engine) error {

				CreateTestAccount(e, t)
				CreateTestTransaction(e, t)
				return e.AddTransaction(context.TODO(), &c.Transaction{
					ID:             2,
					IdempotencyKey: "test",
					Timestamp:      time.Now(),
					LedgerEntries: []c.LedgerEntry{
						{
							AccountID:      0,
							EntryDirection: c.DirectionDebit,
							Amount:         500000,
						},
						{
							AccountID:      0,
							EntryDirection: c.DirectionCredit,
							Amount:         500000,
						},
					},
				})
			},
			wantErr: true,
		},
		{
			name: "creating transaction with uninitialized time returns error",
			operation: func(e *c.Engine) error {

				CreateTestAccount(e, t)
				return e.AddTransaction(context.TODO(), &c.Transaction{
					ID:             1,
					IdempotencyKey: "test",
					Timestamp:      time.Time{},
					LedgerEntries: []c.LedgerEntry{
						{
							AccountID:      0,
							EntryDirection: c.DirectionDebit,
							Amount:         500000,
						},
						{
							AccountID:      0,
							EntryDirection: c.DirectionCredit,
							Amount:         500000,
						},
					},
				})
			},
			wantErr: true,
		},
		{
			name: "creating transaction with no ledger entries returns error",
			operation: func(e *c.Engine) error {
				return e.AddTransaction(context.TODO(), &c.Transaction{
					ID:             1,
					IdempotencyKey: "test",
					Timestamp:      time.Now(),
					LedgerEntries:  []c.LedgerEntry{},
				})
			},
			wantErr: true,
		},
		{
			name: "creating transaction with one ledger entry returns error",
			operation: func(e *c.Engine) error {

				CreateTestAccount(e, t)
				return e.AddTransaction(context.TODO(), &c.Transaction{
					ID:             1,
					IdempotencyKey: "test",
					Timestamp:      time.Now(),
					LedgerEntries: []c.LedgerEntry{
						{
							AccountID:      0,
							EntryDirection: c.DirectionDebit,
							Amount:         500000,
						},
					},
				})
			},
			wantErr: true,
		},
		{
			name: "creating transaction referencing invalid account ID returns error",
			operation: func(e *c.Engine) error {
				return e.AddTransaction(context.TODO(), &c.Transaction{
					ID:             1,
					IdempotencyKey: "test1",
					Timestamp:      time.Now(),
					LedgerEntries: []c.LedgerEntry{
						{
							AccountID:      1,
							EntryDirection: c.DirectionDebit,
							Amount:         500000,
						},
						{
							AccountID:      1,
							EntryDirection: c.DirectionCredit,
							Amount:         500000,
						},
					},
				})
			},
			wantErr: true,
		},
		{
			name: "creating transaction with unbalanced debits and credits returns error",
			operation: func(e *c.Engine) error {
				return e.AddTransaction(context.TODO(), &c.Transaction{
					ID:             1,
					IdempotencyKey: "test",
					Timestamp:      time.Now(),
					LedgerEntries: []c.LedgerEntry{
						{
							AccountID:      1,
							EntryDirection: c.DirectionDebit,
							Amount:         500000,
						},
						{
							AccountID:      1,
							EntryDirection: c.DirectionCredit,
							Amount:         400000,
						},
					},
				})
			},
			wantErr: true,
		},
		{
			name: "creating valid transaction succeeds",
			operation: func(e *c.Engine) error {

				e.CreateAccount(context.TODO(), &c.Account{
					ID:   0,
					Type: c.AccountTypeEquity,
					Name: "Owner's Equity",
				})

				e.CreateAccount(context.TODO(), &c.Account{
					ID:   1,
					Type: c.AccountTypeAsset,
					Name: "Buildings",
				})

				return e.AddTransaction(context.TODO(), &c.Transaction{
					ID:             1,
					IdempotencyKey: "test",
					Timestamp:      time.Now(),
					LedgerEntries: []c.LedgerEntry{
						{
							AccountID:      0,
							EntryDirection: c.DirectionCredit,
							Amount:         400000,
						},
						{
							AccountID:      0,
							EntryDirection: c.DirectionCredit,
							Amount:         100000,
						},
						{
							AccountID:      1,
							EntryDirection: c.DirectionDebit,
							Amount:         500000,
						},
					},
				})
			},
			wantErr: false,
		},
		{
			name: "creating valid transaction and checking ledger integrity succeeds",
			operation: func(e *c.Engine) error {

				var err error
				err = e.CreateAccount(context.TODO(), &c.Account{
					ID:   0,
					Type: c.AccountTypeEquity,
					Name: "Owner's Equity",
				})

				if err != nil {
					t.Fatalf("creating test account failed: %v", err)
				}

				err = e.CreateAccount(context.TODO(), &c.Account{
					ID:   1,
					Type: c.AccountTypeAsset,
					Name: "Buildings",
				})

				if err != nil {
					t.Fatalf("creating test account failed: %v", err)
				}

				err = e.AddTransaction(context.TODO(), &c.Transaction{
					ID:             1,
					IdempotencyKey: "test",
					Timestamp:      time.Now(),
					LedgerEntries: []c.LedgerEntry{
						{
							AccountID:      0,
							EntryDirection: c.DirectionCredit,
							Amount:         400000,
						},
						{
							AccountID:      0,
							EntryDirection: c.DirectionCredit,
							Amount:         100000,
						},
						{
							AccountID:      1,
							EntryDirection: c.DirectionDebit,
							Amount:         500000,
						},
					},
				})

				if err != nil {
					t.Fatalf("creating test transaction failed: %v", err)
				}

				return e.VerifyIntegrity(context.TODO())
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := storageFactory(t)
			engine, err := c.NewEngine(storage)
			if err != nil {
				t.Fatalf("creating engine failed: %v", err)
			}

			got := tt.operation(engine)
			if (got != nil) != tt.wantErr {
				t.Errorf("operation error = %v, want %v", got, tt.wantErr)
			}
		})
	}
}

func CreateTestAccount(e *c.Engine, t *testing.T) {
	t.Helper()
	err := e.CreateAccount(context.TODO(), &c.Account{
		ID:   0,
		Type: c.AccountTypeAsset,
		Name: "TestAccount",
	})

	if err != nil {
		t.Fatalf("createTestAccount failed: %v", err)
	}
}

func CreateTestTransaction(e *c.Engine, t *testing.T) {
	t.Helper()
	err := e.AddTransaction(context.TODO(), &c.Transaction{
		ID:             0,
		IdempotencyKey: "test",
		Timestamp:      time.Now(),
		LedgerEntries: []c.LedgerEntry{
			{
				AccountID:      0,
				EntryDirection: c.DirectionDebit,
				Amount:         500000,
			},
			{
				AccountID:      0,
				EntryDirection: c.DirectionCredit,
				Amount:         500000,
			},
		},
	})

	if err != nil {
		t.Fatalf("createTestTransaction failed: %v", err)
	}
}

func CreateTestSnapshot(e *c.Engine, t *testing.T) {
	t.Helper()
	err := e.CreateSnapshot(context.TODO(), 0)

	if err != nil {
		t.Fatalf("creating initial snapshot failed: %v", err)
	}
}
