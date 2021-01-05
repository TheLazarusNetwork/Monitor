package main

import (
	"context"
	"math"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/TheLazarusNetwork/Monitor/logger"
	"github.com/TheLazarusNetwork/Monitor/utility"
	"github.com/TheLazarusNetwork/Monitor/wallet"

	"github.com/nxadm/tail"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stderr)
	log.SetLevel(log.DebugLevel)

	viper.AddConfigPath(".")    // Look for config in the working directory
	viper.SetConfigFile(".env") //Load .env file
	viper.SetConfigName(".env") //Load .env file
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	utility.CheckError("Error in reading config file:", err)
	log.Infof("Reading Config File: %s", viper.ConfigFileUsed())
}

func main() {

	log.Infof("Lazarus Network Monitor Version: %s", utility.Version)

	err := viper.ReadInConfig()
	utility.CheckError("Error while reading config file:", err)

	mnemonic := viper.Get("MNEMONIC").(string)
	privateKey, publicKey, path, err := wallet.HDWallet(mnemonic)
	utility.CheckError("Error in computing Hierarchical Deterministic Wallet:", err)

	privateKeyBytes := crypto.FromECDSA(privateKey)
	privateKeyHex := hexutil.Encode(privateKeyBytes) // hexutil.Encode(privateKeyBytes)[2:] for without 0x
	publicKeyBytes := crypto.FromECDSAPub(publicKey)
	publicKeyHex := hexutil.Encode(publicKeyBytes[1:])
	walletAddress := crypto.PubkeyToAddress(*publicKey).Hex()

	infuraEndPoint := viper.Get("INFURA_ENDPOINT").(string)
	client, err := ethclient.Dial(infuraEndPoint)
	utility.CheckError("Error in connecting to Infura EndPoint:", err)

	// Display mnemonic and keys
	log.Infof("Mnemonic: %s", mnemonic)
	log.Infof("ETH Private Key: %s", privateKeyHex)
	log.Infof("ETH Public Key: %s", publicKeyHex)
	log.Infof("ETH Wallet Address: %s", walletAddress)
	log.Infof("Path: %s", *path)

	nonce, err := client.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(*publicKey))
	utility.CheckError("Error in fetching nonce:", err)
	log.Infof("Nonce: %d", nonce)

	gasPrice, err := client.SuggestGasPrice(context.Background())
	utility.CheckError("Error in fetching Gas Price:", err)
	log.Infof("Gas Price: %d", gasPrice)

	balance, err := client.BalanceAt(context.Background(), common.HexToAddress(walletAddress), nil)
	utility.CheckError("Error in fetching account balance:", err)
	ethbalance := new(big.Float)
	ethbalance.SetString(balance.String())
	ethValue := new(big.Float).Quo(ethbalance, big.NewFloat(math.Pow10(18)))
	log.Infof("ETH Balance: %f", ethValue)

	auth := bind.NewKeyedTransactor(privateKey)
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)     // in wei
	auth.GasLimit = uint64(300000) // in units
	auth.GasPrice = gasPrice

	// Deploy the Logger Smart Contract and print the contract address and transaction hash
	// loggerAddress, tx, instance, err := logger.DeployLogger(auth, client)
	// utility.CheckError("Unable to bind to deployed instance of contract:", err)
	// _ = instance
	// fmt.Println(loggerAddress.Hex())
	// fmt.Println(tx.Hash().Hex())

	// Load the Logger Contract (if already deployed)
	// loggerAddress := common.HexToAddress(viper.Get("LOGGER_CONTRACT_ADDRESS").(string))
	// instance, err := logger.NewLogger(loggerAddress, client)
	// utility.CheckError("Unable to load instance of the deployed contract:", err)
	// tx, err := instance.DataLog(auth, "Testing Logger datafeed...1")
	// utility.CheckError("Unable to call the contract method:", err)
	// fmt.Printf("TX Hash: %s", tx.Hash().Hex())

	var fileName = viper.Get("LOG_FILE_PATH").(string)

	t, err := tail.TailFile(fileName, tail.Config{Follow: true})
	utility.CheckError("Error in parsing log file: ", err)

	for line := range t.Lines {
		nonce, err := client.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(*publicKey))
		utility.CheckError("Error in fetching nonce:", err)

		gasPrice, err := client.SuggestGasPrice(context.Background())
		utility.CheckError("Error in fetching Gas Price:", err)

		balance, err := client.BalanceAt(context.Background(), common.HexToAddress(walletAddress), nil)
		utility.CheckError("Error in fetching account balance:", err)
		ethbalance := new(big.Float)
		ethbalance.SetString(balance.String())
		ethValue := new(big.Float).Quo(ethbalance, big.NewFloat(math.Pow10(18)))
		log.Infof("Nonce: %d | Gas Price: %d | ETH Balance: %f", nonce, gasPrice, ethValue)

		auth := bind.NewKeyedTransactor(privateKey)
		auth.Nonce = big.NewInt(int64(nonce))
		auth.Value = big.NewInt(0)     // in wei
		auth.GasLimit = uint64(300000) // in units
		auth.GasPrice = gasPrice

		loggerAddress := common.HexToAddress(viper.Get("LOGGER_CONTRACT_ADDRESS").(string))
		instance, err := logger.NewLogger(loggerAddress, client)
		utility.CheckError("Unable to load instance of the deployed contract:", err)

		tx, err := instance.DataLog(auth, line.Text)
		utility.CheckError("Unable to write into the contract method:", err)
		log.Infof("TX Hash: %s --> DataLog: %s", tx.Hash().Hex(), line.Text)
	}
}
