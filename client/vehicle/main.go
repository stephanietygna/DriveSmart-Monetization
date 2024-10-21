package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func main() {
	configFilePath := "connection-org.yaml"
	channelName := "demo"
	mspID := "INMETROMSP"
	chaincodeName := "vehicle"

	file, err := os.OpenFile("logs/log.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(file)

	enrollID := randomString(10)
	registerEnrollUser(configFilePath, enrollID, mspID)

	timeSlice, speedSlice, directionSlice := ReadCSV()

	// Converte slices para string para invocação
	time := strings.Join(timeSlice, " ")
	speed := strings.Join(speedSlice, " ")
	direction := strings.Join(directionSlice, " ")

	invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "DetectSharpTurn", []string{"ABC1234"})
	queryCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "QueryVehicleWallet", []string{"ABC1234"})
}

func registerEnrollUser(configFilePath, enrollID, mspID string) {
	log.Info("Registering User : ", enrollID)
	sdk, err := fabsdk.New(config.FromFile(configFilePath))
	if err != nil {
		log.Error("Failed to create SDK: %s\n", err)
		return
	}
	ctx := sdk.Context()
	caClient, err := mspclient.New(ctx, mspclient.WithCAInstance("inmetro-ca.default"), mspclient.WithOrg(mspID))
	if err != nil {
		log.Error("Failed to create msp client: %s\n", err)
		return
	}

	log.Info("ca client created")
	enrollmentSecret, err := caClient.Register(&mspclient.RegistrationRequest{
		Name:           enrollID,
		Type:           "client",
		MaxEnrollments: -1,
		Affiliation:    "",
		Secret:         enrollID,
	})
	if err != nil {
		log.Error(err)
		return
	}

	err = caClient.Enroll(enrollID, mspclient.WithSecret(enrollmentSecret), mspclient.WithProfile("tls"))
	if err != nil {
		log.Error(errors.WithMessage(err, "failed to register identity"))
		return
	}

	wallet, err := gateway.NewFileSystemWallet(fmt.Sprintf("wallet/%s", mspID))
	if err != nil {
		log.Error("Failed to create wallet: %s", err)
		return
	}

	signingIdentity, err := caClient.GetSigningIdentity(enrollID)
	if err != nil {
		log.Error("Failed to get signing identity: %s", err)
		return
	}

	key, err := signingIdentity.PrivateKey().Bytes()
	if err != nil {
		log.Error("Failed to get private key: %s", err)
		return
	}
	identity := gateway.NewX509Identity(mspID, string(signingIdentity.EnrollmentCertificate()), string(key))

	err = wallet.Put(enrollID, identity)
	if err != nil {
		log.Error(err)
	}
}

func invokeCCgw(configFilePath, channelName, userName, mspID, chaincodeName, fcn string, params []string) {
	configBackend := config.FromFile(configFilePath)
	sdk, err := fabsdk.New(configBackend)
	if err != nil {
		log.Error(err)
		return
	}

	wallet, err := gateway.NewFileSystemWallet(fmt.Sprintf("wallet/%s", mspID))
	if err != nil {
		log.Error("Failed to create wallet: %s", err)
		return
	}

	gw, err := gateway.Connect(gateway.WithSDK(sdk), gateway.WithUser(userName), gateway.WithIdentity(wallet, userName))
	if err != nil {
		log.Error("Failed to create new Gateway: %s", err)
		return
	}
	defer gw.Close()

	nw, err := gw.GetNetwork(channelName)
	if err != nil {
		log.Error("Failed to get network: %s", err)
		return
	}

	contract := nw.GetContract(chaincodeName)
	resp, err := contract.SubmitTransaction(fcn, params...)
	if err != nil {
		log.Error("Failed submit transaction: %s", err)
		return
	}
	log.Info(resp)
}

func queryCCgw(configFilePath, channelName, userName, mspID, chaincodeName, fcn string, args []string) {
	configBackend := config.FromFile(configFilePath)
	sdk, err := fabsdk.New(configBackend)
	if err != nil {
		log.Error(err)
		return
	}

	wallet, err := gateway.NewFileSystemWallet(fmt.Sprintf("wallet/%s", mspID))
	if err != nil {
		log.Error("Failed to create wallet: %s", err)
		return
	}

	gw, err := gateway.Connect(gateway.WithSDK(sdk), gateway.WithUser(userName), gateway.WithIdentity(wallet, userName))
	if err != nil {
		log.Error("Failed to create new Gateway: %s", err)
		return
	}
	defer gw.Close()

	nw, err := gw.GetNetwork(channelName)
	if err != nil {
		log.Error("Failed to get network: %s", err)
		return
	}

	contract := nw.GetContract(chaincodeName)
	resp, err := contract.EvaluateTransaction(fcn, args...)
	if err != nil {
		log.Error("Failed submit transaction: %s", err)
		return
	}
	log.Info(string(resp))
}

func randomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:length]
}

// Função para converter timestamps no formato HH:MM:SS para segundos totais
func convertToSeconds(timeStr string) int {
	parsedTime, err := time.Parse("15:04:05", timeStr)
	if err != nil {
		log.Fatalf("Erro ao converter o timestamp para segundos: %s", err)
	}
	return parsedTime.Hour()*3600 + parsedTime.Minute()*60 + parsedTime.Second()
}

func ReadCSV() ([]string, []string, []string) {
	// Abrir o arquivo CSV
	file, err := os.Open("data.csv")
	if err != nil {
		log.Fatalf("Erro ao abrir o arquivo: %s", err)
	}
	defer file.Close()

	// Criar um leitor CSV
	reader := csv.NewReader(file)

	// Ler todos os registros do arquivo CSV
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Erro ao ler o arquivo CSV: %s", err)
	}

	// Separar os registros em slices de strings
	var timeSlice, speedSlice, directionSlice []string
	for i, record := range records {
		// Ignorar o cabeçalho
		if i == 0 {
			continue
		}
		if len(record) < 3 {
			continue // Ignorar linhas incompletas
		}
		// Converter o tempo para segundos
		timeInSeconds := convertToSeconds(record[0])
		timeSlice = append(timeSlice, fmt.Sprintf("%d", timeInSeconds))
		speedSlice = append(speedSlice, record[1])
		directionSlice = append(directionSlice, record[2])
	}

	return timeSlice, speedSlice, directionSlice
}
