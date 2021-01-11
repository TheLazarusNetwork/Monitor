package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"math"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
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
	publicKeyHex := hexutil.Encode(publicKeyBytes[1:]) // As Ethereum does not DER encode its public keys, public keys in Ethereum are only 64 bytes long
	walletAddress := crypto.PubkeyToAddress(*publicKey).Hex()

	// Display mnemonic and keys
	log.Infof("Mnemonic: %s", mnemonic)
	log.Infof("ETH Private Key: %s", privateKeyHex)
	log.Infof("ETH Public Key: %s", publicKeyHex)
	log.Infof("ETH Wallet Address: %s", walletAddress)
	log.Infof("Path: %s", *path)

	// ECIES Encryption and Decryption
	ecdsaPrivateKey, err := crypto.HexToECDSA(hexutil.Encode(privateKeyBytes)[2:])
	eciesPrivateKey := ecies.ImportECDSA(ecdsaPrivateKey)
	eciesPublicKey := eciesPrivateKey.PublicKey

	infuraEndPoint := viper.Get("INFURA_ENDPOINT").(string)
	client, err := ethclient.Dial(infuraEndPoint)
	utility.CheckError("Error in connecting to Infura EndPoint:", err)

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

		// Encrypt the log data
		encryptedLogData, err := ecies.Encrypt(rand.Reader, &eciesPublicKey, []byte(line.Text), nil, nil)
		if err != nil {
			panic(err)
		}
		log.Infof("Encrypted Log Data: %s", hex.EncodeToString(encryptedLogData))

		tx, err := instance.DataLog(auth, hex.EncodeToString(encryptedLogData))
		utility.CheckError("Unable to write into the contract method:", err)

		// Decryption
		decryptedLogData, err := eciesPrivateKey.Decrypt(encryptedLogData, nil, nil)
		if err != nil {
			panic(err)
		}
		log.Infof("TX Hash: %s --> Decrypted DataLog: %s", tx.Hash().Hex(), string(decryptedLogData))
	}
}
