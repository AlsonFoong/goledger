package adapters

import (
	c "ledger/core"
	ct "ledger/core/test"
	"testing"
)

func TestEngineMemory(t *testing.T) {
	ct.RunEngineTests(t, func(t *testing.T) c.Storage {
		return New()
	})
}
