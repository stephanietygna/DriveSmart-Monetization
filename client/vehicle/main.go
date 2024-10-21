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
	//configFilePath := os.Args[1]
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
	// enrollID := "admin"
	registerEnrollUser(configFilePath, enrollID, mspID)

	timeSlice, speedSlice, directionSlice := ReadCSV()
	time := strings.Join(timeSlice, " ")
	speed := strings.Join(speedSlice, " ")
	direction := strings.Join(directionSlice, " ")

	_ = time
	_ = speed
	_ = direction


	// invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "StoreVehicleData", []string{
	// 	"ABC1234",
	// 	time,
	// 	direction,
	// 	speed,
	// })

	// queryCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "QueryVehicleData", []string{"ABC1234"})

	// invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "CreateVehicleWallet", []string{"ABC1234"})
	// queryCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "QueryVehicleWallet", []string{"ABC1234"})

	// invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "DetectAnomalousAcceleration", []string{"ABC1234"})
	// queryCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "QueryVehicleWallet", []string{"ABC1234"})

	// invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "DetectZigZag", []string{"ABC1234"})
	// queryCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "QueryVehicleWallet", []string{"ABC1234"})

	invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "DetectSharpTurn", []string{"ABC1234"})
	queryCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "QueryVehicleWallet", []string{"ABC1234"})

}

func registerEnrollUser(configFilePath, enrollID, mspID string) {
	log.Info("Registering User : ", enrollID)
	sdk, err := fabsdk.New(config.FromFile(configFilePath))
	ctx := sdk.Context()
	caClient, err := mspclient.New(
		ctx,
		mspclient.WithCAInstance("inmetro-ca.default"),
		mspclient.WithOrg(mspID),
	)

	if err != nil {
		log.Error("Failed to create msp client: %s\n", err)
	}

	if caClient != nil {
		log.Info("ca client created")
	}
	enrollmentSecret, err := caClient.Register(&mspclient.RegistrationRequest{
		Name:           enrollID,
		Type:           "client",
		MaxEnrollments: -1,
		Affiliation:    "",
		// CAName:         "INMETROMSP",
		Attributes: nil,
		Secret:     enrollID,
	})
	if err != nil {
		//fmt.Println("VERIFICAÇÃO")
		log.Error(err)
	}
	err = caClient.Enroll(
		enrollID,
		mspclient.WithSecret(enrollmentSecret),
		mspclient.WithProfile("tls"),
	)
	if err != nil {
		log.Error(errors.WithMessage(err, "failed to register identity"))
	}

	wallet, err := gateway.NewFileSystemWallet(fmt.Sprintf("wallet/%s", mspID))

	signingIdentity, err := caClient.GetSigningIdentity(enrollID)
	key, err := signingIdentity.PrivateKey().Bytes()
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
	}

	wallet, err := gateway.NewFileSystemWallet(fmt.Sprintf("wallet/%s", mspID))

	gw, err := gateway.Connect(
		gateway.WithSDK(sdk),
		gateway.WithUser(userName),
		gateway.WithIdentity(wallet, userName),
	)
	if err != nil {
		log.Error("Failed to create new Gateway: %s", err)
	}
	defer gw.Close()
	nw, err := gw.GetNetwork(channelName)
	if err != nil {
		log.Error("Failed to get network: %s", err)
	}

	contract := nw.GetContract(chaincodeName)

	resp, err := contract.SubmitTransaction(fcn, params...)

	if err != nil {
		log.Error("Failed submit transaction: %s", err)
	}
	log.Info(resp)

}

func queryCCgw(configFilePath, channelName, userName, mspID, chaincodeName, fcn string, args []string) {

	configBackend := config.FromFile(configFilePath)
	sdk, err := fabsdk.New(configBackend)
	if err != nil {
		log.Error(err)
	}

	wallet, err := gateway.NewFileSystemWallet(fmt.Sprintf("wallet/%s", mspID))

	gw, err := gateway.Connect(
		gateway.WithSDK(sdk),
		gateway.WithUser(userName),
		gateway.WithIdentity(wallet, userName),
	)

	if err != nil {
		log.Error("Failed to create new Gateway: %s", err)
	}
	defer gw.Close()
	nw, err := gw.GetNetwork(channelName)
	if err != nil {
		log.Error("Failed to get network: %s", err)
	}

	contract := nw.GetContract(chaincodeName)

	resp, err := contract.EvaluateTransaction(fcn, args...)

	if err != nil {
		log.Error("Failed submit transaction: %s", err)
	}
	log.Info(string(resp))

}

func randomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:length]
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
		timeSlice = append(timeSlice, record[0])
		speedSlice = append(speedSlice, record[1])
		directionSlice = append(directionSlice, record[2])
	}

	// Imprimir os slices
	// fmt.Println("Time Slice:", timeSlice)
	// fmt.Println("Speed Slice:", speedSlice)
	// fmt.Println("Direction Slice:", directionSlice)

	return timeSlice, speedSlice, directionSlice
}
