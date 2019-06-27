package main

import (
	"math/big"
	"testing"
)

func TestTransfer(t *testing.T) {
	addr := "0xe71c8416A1359712756Bc66f6D78ab091720e9A4"
	transferTo(big.NewInt(10000000000), addr)
}
