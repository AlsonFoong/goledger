package adapters

import (
	"context"
	"errors"
	"fmt"
	c "ledger/core"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidConnString   = errors.New("could not parse connection string")
	ErrConnPoolInitFailure = errors.New("could not initialize connection pool")
	ErrConnPoolPingFailure = errors.New("could not reach database instance")
	ErrInvalidState        = errors.New("invalid state detected - ledger is unbalanced!")
	ErrAccountDoesNotExist = errors.New("account ID does not reference an existing account")
)

type PostgresDatabase struct {
	pool *pgxpool.Pool
}

type PostgresTx interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func New(ctx context.Context, connString string) (*PostgresDatabase, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, ErrInvalidConnString
	}

	config.MaxConns = 20
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, ErrConnPoolInitFailure
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, ErrConnPoolPingFailure
	}

	return &PostgresDatabase{pool: pool}, nil
}

func (db *PostgresDatabase) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

func (db *PostgresDatabase) CreateAccount(ctx context.Context, acc *c.Account) error {
	query := `INSERT INTO accounts (id, type, name) VALUES ($1, $2, $3);`
	_, err := db.pool.Exec(ctx, query, acc.ID, acc.Type, acc.Name)
	return db.translateError(err)
}

func (db *PostgresDatabase) GetAccount(ctx context.Context, id int64) (*c.Account, error) {
	query := `SELECT id, type, name FROM accounts WHERE id = $1;`
	row := db.pool.QueryRow(ctx, query, id)

	var acc c.Account
	err := row.Scan(&acc.ID, &acc.Type, &acc.Name)
	if err != nil {
		return nil, db.translateError(err)
	}

	return &acc, nil
}

func (db *PostgresDatabase) GetAccountBalance(ctx context.Context, id int64) (int64, error) {
	return db.getAccountBalance(ctx, db.pool, id)
}

func (db *PostgresDatabase) AddTransaction(ctx context.Context, tx *c.Transaction) error {
	pgTx, err := db.pool.Begin(ctx)
	if err != nil {
		return db.translateError(err)
	}
	defer pgTx.Rollback(ctx)

	txQuery := `INSERT INTO transactions (id, idempotency_key, timestamp) VALUES ($1, $2, $3) RETURNING sequence_number;`
	var sequenceNumber int64
	row := pgTx.QueryRow(ctx, txQuery, tx.ID, tx.IdempotencyKey, tx.Timestamp)
	err = row.Scan(&sequenceNumber)
	if err != nil {
		return db.translateError(err)
	}

	entryQuery := `INSERT INTO ledger_entries (id, transaction_id, transaction_sequence_number, account_id, entry_direction, amount) VALUES ($1, $2, $3, $4, $5, $6);`
	for i := range tx.LedgerEntries {
		entry := tx.LedgerEntries[i]
		_, err = pgTx.Exec(ctx, entryQuery, fmt.Sprintf("%v_%v", tx.ID, i), tx.ID, sequenceNumber, entry.AccountID, entry.EntryDirection, entry.Amount)
		if err != nil {
			return db.translateError(err)
		}
	}

	err = pgTx.Commit(ctx)
	return db.translateError(err)
}

func (db *PostgresDatabase) GetTransactionByID(ctx context.Context, id int64) (*c.Transaction, error) {
	txQuery := `SELECT id, idempotency_key, timestamp FROM transactions WHERE id = $1;`
	row := db.pool.QueryRow(ctx, txQuery, id)
	return db.getTransaction(ctx, row)
}

func (db *PostgresDatabase) GetTransactionByIdempotencyKey(ctx context.Context, idempotencyKey string) (*c.Transaction, error) {
	txQuery := `SELECT id, idempotency_key, timestamp FROM transactions WHERE idempotency_key = $1;`
	row := db.pool.QueryRow(ctx, txQuery, idempotencyKey)
	return db.getTransaction(ctx, row)
}

func (db *PostgresDatabase) CreateSnapshot(ctx context.Context, id int64) error {
	pgTx, err := db.pool.Begin(ctx)
	if err != nil {
		return db.translateError(err)
	}
	defer pgTx.Rollback(ctx)

	balance, err := db.getAccountBalance(ctx, pgTx, id)
	if err != nil {
		return db.translateError(err)
	}

	var latestTxSequenceNumber *int64 // Allow nil value for accounts with no transactions
	var seq int64

	latestTxQuery := `SELECT sequence_number FROM transactions ORDER BY sequence_number DESC LIMIT 1;`
	row := pgTx.QueryRow(ctx, latestTxQuery)
	err = row.Scan(&seq)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return db.translateError(err)
		}
		// Leave sequence number as nil value
	} else {
		latestTxSequenceNumber = &seq
	}

	snapQuery := `
	INSERT INTO snapshots (account_id, total_amount, last_processed_transaction_sequence_number)
	VALUES ($1, $2, $3)
	ON CONFLICT (account_id)
	DO UPDATE SET total_amount = $2, last_processed_transaction_sequence_number = $3;`

	_, err = pgTx.Exec(ctx, snapQuery, id, balance, latestTxSequenceNumber)
	if err != nil {
		return db.translateError(err)
	}

	return db.translateError(pgTx.Commit(ctx))
}

func (db *PostgresDatabase) GetSnapshot(ctx context.Context, id int64) (*c.TransactionSnapshot, error) {
	return db.getSnapshot(ctx, db.pool, id)
}

func (db *PostgresDatabase) VerifyIntegrity(ctx context.Context) error {
	sumQuery := `SELECT COALESCE(SUM(CASE WHEN entry_direction = 'DEBIT' THEN amount ELSE -amount END), 0) FROM ledger_entries;`
	row := db.pool.QueryRow(ctx, sumQuery)

	var globalSum int64
	err := row.Scan(&globalSum)
	if err != nil {
		return db.translateError(err)
	}

	if globalSum != 0 {
		return ErrInvalidState
	} else {
		return nil
	}
}

func (db *PostgresDatabase) getTransaction(ctx context.Context, row pgx.Row) (*c.Transaction, error) {
	var tx c.Transaction
	err := row.Scan(&tx.ID, &tx.IdempotencyKey, &tx.Timestamp)
	if err != nil {
		return nil, db.translateError(err)
	}

	entriesQuery := `SELECT account_id, entry_direction, amount FROM ledger_entries WHERE transaction_id = $1;`
	rows, err := db.pool.Query(ctx, entriesQuery, tx.ID)
	var accID, amount int64
	var direction c.EntryDirection
	i := 0
	_, err = pgx.ForEachRow(rows, []any{&accID, &direction, &amount}, func() error {
		var entry c.LedgerEntry
		entry.AccountID = accID
		entry.EntryDirection = direction
		entry.Amount = amount
		tx.LedgerEntries = append(tx.LedgerEntries, entry)
		i += 1
		return nil
	})
	if err != nil {
		return nil, db.translateError(err)
	}

	return &tx, nil
}

func (db *PostgresDatabase) getAccountBalance(ctx context.Context, dbTx PostgresTx, id int64) (int64, error) {
	var exists bool
	existenceCheck := `SELECT EXISTS(SELECT 1 FROM accounts WHERE id = $1)`
	err := dbTx.QueryRow(ctx, existenceCheck, id).Scan(&exists)
	if err != nil {
		return 0, db.translateError(err)
	}
	if !exists {
		return 0, ErrAccountDoesNotExist
	}

	snapshot, err := db.getSnapshot(ctx, dbTx, id)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return 0, db.translateError(err)
		}

		snapshot = &c.TransactionSnapshot{
			AccountID:                  id,
			TotalAmount:                0,
			LastProcessedTransactionID: 0,
		}
	}

	balanceQuery := `
	SELECT entry_direction, amount FROM ledger_entries WHERE account_id = $1 AND transaction_sequence_number > $2;`
	rows, err := dbTx.Query(ctx, balanceQuery, id, snapshot.LastProcessedTransactionID)
	if err != nil {
		return 0, db.translateError(err)
	}

	var entryDirection c.EntryDirection
	var amount int64
	totalAmount := snapshot.TotalAmount

	_, err = pgx.ForEachRow(rows, []any{&entryDirection, &amount}, func() error {
		if entryDirection == c.DirectionDebit {
			totalAmount, err = c.AddSignedIntSafe(totalAmount, amount)
		} else {
			totalAmount, err = c.AddSignedIntSafe(totalAmount, -amount)
		}
		return err
	})

	return totalAmount, db.translateError(err)
}

func (db *PostgresDatabase) getSnapshot(ctx context.Context, dbTx PostgresTx, id int64) (*c.TransactionSnapshot, error) {
	snapQuery := `SELECT account_id, total_amount, last_processed_transaction_sequence_number FROM snapshots WHERE account_id = $1;`
	row := dbTx.QueryRow(ctx, snapQuery, id)

	var snapshot c.TransactionSnapshot
	err := row.Scan(&snapshot.AccountID, &snapshot.TotalAmount, &snapshot.LastProcessedTransactionID)
	if err != nil {
		return nil, db.translateError(err)
	}
	return &snapshot, nil
}

// TODO: Address double translations
// TODO: Implement basic translations
func (db *PostgresDatabase) translateError(err error) error {
	if err == nil {
		return nil
	}

	// if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
	// 	return fmt.Errorf("%v: %v", pgErr.Code, pgErr.Message)
	// }

	// return fmt.Errorf("generic error: %v", err)
	return err
}
