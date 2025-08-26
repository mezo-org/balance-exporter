package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	allWatching   []*Watching
	allContracts  []*ContractWatching
	port          string
	updates       string
	prefix        string
	loadSeconds   float64
	totalLoaded   int64
	eth           *ethclient.Client
	chainId       string
)

type Watching struct {
	Name    string
	Address string
	Balance string
}

type ContractWatching struct {
	Name     string
	Address  string
	ABI      string
	Function string
	Result   string
}

func (w *Watching) String() string {
	result, _ := json.Marshal(w)
	return string(result)
}

func (c *ContractWatching) String() string {
	result, _ := json.Marshal(c)
	return string(result)
}

// Connect to geth server
func ConnectionToGeth(url string) error {
	var err error
	eth, err = ethclient.Dial(url)
	return err
}

// Fetch ETH balance from Geth server
func GetEthBalance(address string) *big.Float {
	balance, err := eth.BalanceAt(context.TODO(), common.HexToAddress(address), nil)
	if err != nil {
		fmt.Printf("Error fetching ETH Balance for address: %v\n", address)
	}
	return ToEther(balance)
}

// Fetch ETH balance from Geth server
func CurrentBlock() uint64 {
	block, err := eth.BlockByNumber(context.TODO(), nil)
	if err != nil {
		fmt.Printf("Error fetching current block height: %v\n", err)
		return 0
	}
	return block.NumberU64()
}

// CONVERTS WEI TO ETH
func ToEther(o *big.Int) *big.Float {
	pul, int := big.NewFloat(0), big.NewFloat(0)
	int.SetInt(o)
	pul.Mul(big.NewFloat(0.000000000000000001), int)
	return pul
}

// HTTP response handler for /metrics
func MetricsHttp(w http.ResponseWriter, r *http.Request) {
	var allOut []string
	total := big.NewFloat(0)
	for _, v := range allWatching {
		if v.Balance == "" {
			v.Balance = "0"
		}
		bal := big.NewFloat(0)
		bal.SetString(v.Balance)
		total.Add(total, bal)
		allOut = append(allOut, fmt.Sprintf("%vaccount_balance{name=\"%v\",address=\"%v\",chain_id=\"%s\"} %v", prefix, v.Name, v.Address, chainId, v.Balance))
	}
	allOut = append(allOut, fmt.Sprintf("%vaccount_balance_total %0.18f", prefix, total))
	allOut = append(allOut, fmt.Sprintf("%vaccount_load_seconds %0.2f", prefix, loadSeconds))
	allOut = append(allOut, fmt.Sprintf("%vaccount_loaded_addresses %v", prefix, totalLoaded))
	allOut = append(allOut, fmt.Sprintf("%vaccount_total_addresses %v", prefix, len(allWatching)))
	fmt.Fprintln(w, strings.Join(allOut, "\n"))
}

// Open the addresses.txt file (name:address)
func OpenAddresses(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments.
		if strings.HasPrefix(line, "#") {
			continue
		}

		object := strings.Split(line, ":")

		// Skip invalid lines.
		if len(object) < 2 {
			continue
		}

		if common.IsHexAddress(object[1]) {
			w := &Watching{
				Name:    object[0],
				Address: object[1],
			}
			allWatching = append(allWatching, w)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return err
}

// Open the contracts.txt file (name|address|abi|function)
func OpenContracts(filename string) error {
	if filename == "" {
		return nil
	}
	
	file, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments.
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Parse line with format: name|address|abi|function
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}

		name := parts[0]
		address := parts[1]
		abiString := parts[2]
		function := parts[3]

		if common.IsHexAddress(address) {
			c := &ContractWatching{
				Name:     name,
				Address:  address,
				ABI:      abiString,
				Function: function,
			}
			allContracts = append(allContracts, c)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func main() {
	gethUrl := os.Getenv("CHAIN_RPC_URL")
	checkFrequencySeconds := getEnvCheckFrequency()
	port = os.Getenv("PORT")
	prefix = os.Getenv("PREFIX")

	addressesFilePath := os.Getenv("ADDRESSES_FILE")
	contractsFilePath := os.Getenv("CONTRACTS_FILE")

	err := OpenAddresses(addressesFilePath)
	if err != nil {
		panic(err)
	}

	err = OpenContracts(contractsFilePath)
	if err != nil {
		panic(err)
	}

	err = ConnectionToGeth(gethUrl)
	if err != nil {
		panic(err)
	}

	chainIdBigInt, err := eth.ChainID(context.TODO())
	if err != nil {
		panic(fmt.Sprintf("Error fetching chain ID: %v", err))
	}

	chainId = chainIdBigInt.String()

	// check address balances
	go func() {
		for {
			totalLoaded = 0
			t1 := time.Now()
			fmt.Printf("Checking %v wallets...\n", len(allWatching))
			for _, v := range allWatching {
				v.Balance = GetEthBalance(v.Address).String()
				totalLoaded++
			}
			t2 := time.Now()
			loadSeconds = t2.Sub(t1).Seconds()
			fmt.Printf("Finished checking %v wallets in %0.0f seconds, sleeping for %v seconds.\n", len(allWatching), loadSeconds, checkFrequencySeconds)
			time.Sleep(time.Duration(checkFrequencySeconds) * time.Second)
		}
	}()

	block := CurrentBlock()

	fmt.Printf("balance-exporter has started on port %v using Geth server: %v at block #%v\n", port, gethUrl, block)
	http.HandleFunc("/metrics", MetricsHttp)
	panic(http.ListenAndServe("0.0.0.0:"+port, nil))
}

func getEnvCheckFrequency() int {
	defaultCheckFrequencySeconds := 60
	checkFrequencySecondsString := os.Getenv("CHECK_FREQUENCY_SECONDS")

	if len(checkFrequencySecondsString) > 0 {
		checkFrequencySeconds, err := strconv.Atoi(checkFrequencySecondsString)
		if err != nil {
			return defaultCheckFrequencySeconds
		}

		return checkFrequencySeconds
	} else {
		return defaultCheckFrequencySeconds
	}
}
