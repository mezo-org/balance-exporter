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

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum"
)

var (
	allWatching  []*Watching
	allContracts []*ContractWatching
	port         string
	updates      string
	prefix       string
	loadSeconds  float64
	totalLoaded  int64
	eth          *ethclient.Client
	chainId      string
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

func CallContractFunction(contractAddress string, abiString string, functionName string) (string, error) {
	// Parse ABI
	contractABI, err := abi.JSON(strings.NewReader(abiString))
	if err != nil {
		return "", fmt.Errorf("failed to parse ABI: %v", err)
	}

	// Pack the function call
	data, err := contractABI.Pack(functionName)
	if err != nil {
		return "", fmt.Errorf("failed to pack function call: %v", err)
	}

	// Create call message
	contractAddr := common.HexToAddress(contractAddress)
	msg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: data,
	}

	// Check if eth client is connected
	if eth == nil {
		return "", fmt.Errorf("failed to call contract: eth client not connected")
	}

	// Call the contract
	result, err := eth.CallContract(context.TODO(), msg, nil)
	if err != nil {
		return "", fmt.Errorf("failed to call contract: %v", err)
	}

	// Unpack the result
	var output []interface{}
	err = contractABI.UnpackIntoInterface(&output, functionName, result)
	if err != nil {
		return "", fmt.Errorf("failed to unpack result: %v", err)
	}

	// Convert result to string
	if len(output) > 0 {
		if bigInt, ok := output[0].(*big.Int); ok {
			return bigInt.String(), nil
		}
		return fmt.Sprintf("%v", output[0]), nil
	}

	return "0", nil
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
	
	// Add contract function results
	for _, c := range allContracts {
		if c.Result == "" {
			c.Result = "0"
		}
		allOut = append(allOut, fmt.Sprintf("%vcontract_function_result{contract=\"%v\",function=\"%v\",address=\"%v\",chain_id=\"%s\"} %v", prefix, c.Name, c.Function, c.Address, chainId, c.Result))
	}
	allOut = append(allOut, fmt.Sprintf("%vcontract_total_contracts %v", prefix, len(allContracts)))
	
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
			fmt.Printf("Checking %v wallets and %v contracts...\n", len(allWatching), len(allContracts))
			
			// Check wallet balances
			for _, v := range allWatching {
				v.Balance = GetEthBalance(v.Address).String()
				totalLoaded++
			}
			
			// Check contract functions
			for _, c := range allContracts {
				result, err := CallContractFunction(c.Address, c.ABI, c.Function)
				if err != nil {
					fmt.Printf("Error calling contract function %s on %s: %v\n", c.Function, c.Name, err)
					c.Result = "0"
				} else {
					c.Result = result
				}
				totalLoaded++
			}
			
			t2 := time.Now()
			loadSeconds = t2.Sub(t1).Seconds()
			fmt.Printf("Finished checking %v wallets and %v contracts in %0.0f seconds, sleeping for %v seconds.\n", len(allWatching), len(allContracts), loadSeconds, checkFrequencySeconds)
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
