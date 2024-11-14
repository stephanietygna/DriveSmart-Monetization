package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"strconv"
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
	//enrollID := "inmetro-admin-default"
	registerEnrollUser(configFilePath, enrollID, mspID)

	// Chamar a função ReadCSV
	timestamps, lats, lons, vehicleSpeeds, accel_x, accel_y, accel_z, err := ReadCSV()
	if err != nil {
		log.Fatalf("Erro ao ler o CSV: %s", err)
	}

	// position := 22
	position := 28

	// Example usage of ConvertTimestampToUnix
	unixTimestamp, err := ConvertTimestampToUnix(timestamps[position])
	if err != nil {
		log.Fatalf("Erro ao converter o timestamp: %s", err)
	}

	cleanedLatitude, err := SanitizeFloatString(lats[position])
	cleanedLongitude, err := SanitizeFloatString(lons[position])

	fmt.Println("Timestamps:", timestamps[position])
	fmt.Println("Unix Time:", unixTimestamp)
	fmt.Println("Latitudes:", lats[position])
	fmt.Println("Longitudes:", lons[position])
	fmt.Println("Cleaned Latitude:", cleanedLatitude)
	fmt.Println("Cleaned Longitude:", cleanedLongitude)
	fmt.Println("Velocidades do veículo:", vehicleSpeeds[position])
	fmt.Println("Aceleração X:", accel_x[position])
	fmt.Println("Aceleração Y:", accel_y[position])
	fmt.Println("Aceleração Z:", accel_z[position])

	// invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "StoreVehicleData", []string{"ABC4444", unixTimestamp, cleanedLatitude, cleanedLongitude, vehicleSpeeds[position], accel_x[position], accel_y[position], accel_z[position]})
	invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "CreateVehicleWallet", []string{"ABC4444"})
	// queryCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "QueryVehicleData", []string{"ABC4444"})
	// invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "DetectZigZag", []string{"ABC4444"})
	invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "DetectSharpTurn", []string{"ABC4444"})
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

// ConvertTimestampToUnix converts a timestamp string to a Unix time string
func ConvertTimestampToUnix(timestamp string) (string, error) {
	// layout := "15:04:05.000"
	fullTimestamp := "1970-01-01 " + timestamp // Inclui data fixa para epoch Unix
	fullLayout := "2006-01-02 15:04:05.000"

	t, err := time.Parse(fullLayout, fullTimestamp)
	if err != nil {
		return "", fmt.Errorf("failed to parse timestamp: %w", err)
	}
	return fmt.Sprintf("%d", t.Unix()), nil
}

// SanitizeFloatString removes invalid characters from a float string
func SanitizeFloatString(input string) (string, error) {
	// Remove any commas or other invalid characters
	cleaned := strings.ReplaceAll(input, ".", "")
	cleaned = strings.ReplaceAll(cleaned, ",", ".")
	if _, err := strconv.ParseFloat(cleaned, 64); err != nil {
		return "", fmt.Errorf("invalid float string: %s", input)
	}
	return cleaned, nil
}

func randomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:length]
}

func ReadCSV() ([]string, []string, []string, []string, []string, []string, []string, error) {
	// Abrir o arquivo CSV
	file, err := os.Open("data/obd_limited.csv")
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("erro ao abrir o arquivo: %w", err)
	}
	defer file.Close()

	// Criar um leitor CSV
	reader := csv.NewReader(file)
	reader.Comma = ',' // Definir o delimitador como vírgula

	// Ler o cabeçalho
	header, err := reader.Read()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("erro ao ler o cabeçalho do arquivo CSV: %w", err)
	}

	// Inicializar slices para armazenar os dados das colunas desejadas
	var timestamps, lats, lons, vehicleSpeeds, accelX, accelY, accelZ []string

	// Mapear os índices das colunas desejadas
	columnIndices := make(map[string]int)
	for i, col := range header {
		switch col {
		case "timestamp":
			columnIndices["timestamp"] = i
		case "lat":
			columnIndices["lat"] = i
		case "lon":
			columnIndices["lon"] = i
		case "vehicle_speed":
			columnIndices["vehicle_speed"] = i
		case "accel_x":
			columnIndices["accel_x"] = i
		case "accel_y":
			columnIndices["accel_y"] = i
		case "accel_z":
			columnIndices["accel_z"] = i
		}
	}

	// Ler as primeiras 5 linhas
	for i := 0; i < 50; i++ {
		record, err := reader.Read()
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("erro ao ler o arquivo CSV: %w", err)
		}
		if idx, ok := columnIndices["timestamp"]; ok {
			timestamps = append(timestamps, record[idx])
		}
		if idx, ok := columnIndices["lat"]; ok {
			lats = append(lats, record[idx])
		}
		if idx, ok := columnIndices["lon"]; ok {
			lons = append(lons, record[idx])
		}
		if idx, ok := columnIndices["vehicle_speed"]; ok {
			vehicleSpeeds = append(vehicleSpeeds, record[idx])
		}
		if idx, ok := columnIndices["accel_x"]; ok {
			accelX = append(accelX, record[idx])
		}
		if idx, ok := columnIndices["accel_y"]; ok {
			accelY = append(accelY, record[idx])
		}
		if idx, ok := columnIndices["accel_z"]; ok {
			accelZ = append(accelZ, record[idx])
		}
	}

	return timestamps, lats, lons, vehicleSpeeds, accelX, accelY, accelZ, nil
}
