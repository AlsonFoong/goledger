package adapters

import (
	c "ledger/core"
	ct "ledger/core/test"
	"testing"
)

func TestMemoryEngine(t *testing.T) {
	ct.RunEngineTests(t, func(t *testing.T) c.Storage {
		return New()
	})
}

func BenchmarkAddTransaction(b *testing.B) {
	ct.RunAddTransactionBenchmark(b, func(b *testing.B) c.Storage {
		return New()
	})
}
