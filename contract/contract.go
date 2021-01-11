package contract

import (
	"fmt"

	"github.com/TheLazarusNetwork/Monitor/logger"
	"github.com/TheLazarusNetwork/Monitor/utility"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/spf13/viper"
)

// Deploy ...
func Deploy(auth *bind.TransactOpts, client *ethClient.Client) {
	// Deploy the Logger Smart Contract and print the contract address and transaction hash
	loggerAddress, tx, instance, err := logger.DeployLogger(auth, client)
	utility.CheckError("Unable to bind to deployed instance of contract:", err)
	_ = instance
	fmt.Println(loggerAddress.Hex())
	fmt.Println(tx.Hash().Hex())
}

// Load ...
func Load(auth *bind.TransactOpts, client *ethClient.Client) {
	// Load the Logger Contract (if already deployed)
	loggerAddress := common.HexToAddress(viper.Get("LOGGER_CONTRACT_ADDRESS").(string))
	instance, err := logger.NewLogger(loggerAddress, client)
	utility.CheckError("Unable to load instance of the deployed contract:", err)
	tx, err := instance.DataLog(auth, "Testing Logger datafeed...1")
	utility.CheckError("Unable to call the contract method:", err)
	fmt.Printf("TX Hash: %s", tx.Hash().Hex())
}
