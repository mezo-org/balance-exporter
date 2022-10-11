package main

import (
	"reflect"
	"testing"
)

func TestOpenAddresses(t *testing.T) {
	expectedResult := []*Watching{
		{
			Name:    "etherdelta",
			Address: "0x8d12A197cB00D4747a1fe03395095ce2A5CC6819",
		},
		{
			Name:    "needs-trimming",
			Address: "0x647dC1366Da28f8A64EB831fC8E9F05C90d1EA5a",
		},
		{
			Name:    "bittrex",
			Address: "0xFBb1b73C4f0BDa4f67dcA266ce6Ef42f520fBB98",
		},
		{
			Name:    "poloniex",
			Address: "0x32Be343B94f860124dC4fEe278FDCBD38C102D88",
		},
		{
			Name:    "kraken",
			Address: "0x267be1c1d684f78cb4f6a176c4911b741e4ffdc0",
		},
	}

	OpenAddresses("test/data/addresses.txt")

	if !reflect.DeepEqual(allWatching, expectedResult) {
		t.Errorf(
			"unexpected result:\nexpected: %s\nactual:   %s",
			expectedResult,
			allWatching,
		)
	}
}
