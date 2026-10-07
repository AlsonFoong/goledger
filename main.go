package main

import (
	"context"
	"fmt"
	c "ledger/core"
	"log"
	"strconv"
	"sync"
	"time"

	p "ledger/adapters/postgres"
)

func main() {
	// e, _ := c.NewEngine(m.New())
	connString := "postgresql://postgres:password123@localhost:5432/postgres?sslmode=disable"
	db, _ := p.New(context.Background(), connString)
	e, _ := c.NewEngine(db)

	e.CreateAccount(context.TODO(), &c.Account{
		ID:   1,
		Type: c.AccountTypeEquity,
		Name: "Owner's Equity",
	})

	e.CreateAccount(context.TODO(), &c.Account{
		ID:   2,
		Type: c.AccountTypeAsset,
		Name: "Buildings",
	})

	e.CreateAccount(context.TODO(), &c.Account{
		ID:   3,
		Type: c.AccountTypeAsset,
		Name: "Equipment",
	})

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			err := e.AddTransaction(context.TODO(), &c.Transaction{
				ID:             int64(i),
				IdempotencyKey: "tx" + strconv.Itoa(i),
				Timestamp:      time.Now(),
				LedgerEntries: []c.LedgerEntry{
					{
						AccountID:      1,
						EntryDirection: c.DirectionCredit,
						Amount:         1000000000,
					},
					{
						AccountID:      2,
						EntryDirection: c.DirectionDebit,
						Amount:         1000000000,
					},
					{
						AccountID:      1,
						EntryDirection: c.DirectionCredit,
						Amount:         200000000,
					},
					{
						AccountID:      3,
						EntryDirection: c.DirectionDebit,
						Amount:         200000000,
					},
				},
			})

			if err != nil {
				log.Fatalf("err occurred: %v", err)
			}
		}(int64(i))
	}

	wg.Wait()
	fmt.Println("Time elapsed (round one):", time.Since(start))

	fmt.Println(e.VerifyIntegrity(context.TODO()))
	fmt.Println("Time elapsed (first integrity check):", time.Since(start))

	e.CreateSnapshot(context.TODO(), 1)
	e.CreateSnapshot(context.TODO(), 2)
	e.CreateSnapshot(context.TODO(), 3)

	fmt.Println("Time elapsed (snapshots):", time.Since(start))

	fmt.Println(e.VerifyIntegrity(context.TODO()))
	fmt.Println("Time elapsed (second integrity check):", time.Since(start))

	for i := 1001; i < 1500; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			err := e.AddTransaction(context.TODO(), &c.Transaction{
				ID:             int64(i),
				IdempotencyKey: "txsecond" + strconv.Itoa(i),
				Timestamp:      time.Now(),
				LedgerEntries: []c.LedgerEntry{
					{
						AccountID:      1,
						EntryDirection: c.DirectionCredit,
						Amount:         1000000000,
					},
					{
						AccountID:      2,
						EntryDirection: c.DirectionDebit,
						Amount:         1000000000,
					},
					{
						AccountID:      1,
						EntryDirection: c.DirectionCredit,
						Amount:         200000000,
					},
					{
						AccountID:      3,
						EntryDirection: c.DirectionDebit,
						Amount:         200000000,
					},
				},
			})

			if err != nil {
				log.Fatalf("err occurred: %v", err)
			}
		}(int64(i))
	}

	wg.Wait()
	fmt.Println("Time elapsed (round two):", time.Since(start))

	// fmt.Println(e.GetTransactionByID(998393))
	// fmt.Println(e.GetTransactionByID(99))

	// fmt.Printf("engine.accounts: %v\n", e.Accounts)
	// fmt.Printf("engine.transactions: %v\n", e.IdempotencyKeys)

	// jsonData, _ := json.MarshalIndent(e.Accounts, "", "  ")
	// fmt.Println(string(jsonData))

	// jsonData2, _ := json.MarshalIndent(e.Transactions, "", "  ")
	// fmt.Println(string(jsonData2))

	// for i := 0; i < len(e.Transactions); i++ {
	// 	// fmt.Println(e.Transactions[int64(i)].ID)
	// 	fmt.Println(e.Transactions[int64(i)].Timestamp)
	// 	// fmt.Println("break")
	// }

	// var total int64
	// for _, acc := range e.Accounts {
	// 	amount, err := e.GetAccountBalance(acc.ID)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		return
	// 	}
	// 	total += amount
	// }

	// if total == 0 {
	// 	fmt.Println("VALID STATE MAINTAINED, SUM(DEBITS) - SUM(CREDITS) = 0")
	// } else {
	// 	fmt.Println("INVALID STATE DETECTED, TOTAL =", total)
	// }

	fmt.Println(e.VerifyIntegrity(context.TODO()))
	fmt.Println("Time elapsed (third integrity check):", time.Since(start))
}
