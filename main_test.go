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
		{
			Name:    "duplicated-name",
			Address: "0x36Fb6cd260A63719BB7EfC865e1aEaa60922a6d9",
		},
		{
			Name:    "duplicated-name",
			Address: "0xF6Af0fD6aA7c78EA7038D04F901493f375234f24",
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

func TestOpenContracts(t *testing.T) {
	// Reset the global variable before test
	allContracts = nil
	
	expectedResult := []*ContractWatching{
		{
			Name:     "PCV",
			Address:  "0x4dDD70f4C603b6089c07875Be02fEdFD626b80Af",
			ABI:      "[{\"inputs\":[],\"name\":\"debtToPay\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
			Function: "debtToPay",
		},
	}

	err := OpenContracts("test/data/contracts.txt")
	if err != nil {
		t.Errorf("OpenContracts returned error: %v", err)
	}

	if len(allContracts) != len(expectedResult) {
		t.Errorf("Expected %d contracts, got %d", len(expectedResult), len(allContracts))
	}

	if len(allContracts) > 0 {
		if allContracts[0].Name != expectedResult[0].Name {
			t.Errorf("Expected name %s, got %s", expectedResult[0].Name, allContracts[0].Name)
		}
		if allContracts[0].Address != expectedResult[0].Address {
			t.Errorf("Expected address %s, got %s", expectedResult[0].Address, allContracts[0].Address)
		}
		if allContracts[0].Function != expectedResult[0].Function {
			t.Errorf("Expected function %s, got %s", expectedResult[0].Function, allContracts[0].Function)
		}
	}
}
