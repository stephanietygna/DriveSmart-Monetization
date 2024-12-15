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
	//configFilePath := os.Args[1]
	configFilePath := "connection-org.yaml"
	channelName := "demo"
	mspID := "INMETROMSP"
	chaincodeName := "vehicle"

	enrollID := randomString(10)
	registerEnrollUser(configFilePath, enrollID, mspID)

	//invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "CreateVehicleWallet", "ABC1234")
	//invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "CreateCar")

	configBackend := config.FromFile(configFilePath)
	sdk, err := fabsdk.New(configBackend)
	if err != nil {
		log.Error(err)
	}

	wallet, err := gateway.NewFileSystemWallet(fmt.Sprintf("wallet/%s", mspID))
	if err != nil {
		log.Error(err)
		return
	}
	if err != nil {
		log.Errorf("Failed to create wallet: %s", err)
		return
	}

	gw, err := gateway.Connect(
		gateway.WithSDK(sdk),
		gateway.WithUser(enrollID),
		gateway.WithIdentity(wallet, enrollID),
	)
	if err != nil {
		log.Errorf("Failed to create new Gateway: %s", err)
	}
	// defer gw.Close()
	nw, err := gw.GetNetwork(channelName)
	if err != nil {
		log.Errorf("Failed to get network: %s", err)
	}

	timestamps, lats, lons, vehicleSpeeds, accel_x, accel_y, accel_z, err := ReadCSV()
	if err != nil {
		log.Fatalf("Failed to read CSV: %s", err)
	}

	// criar carteira (fora do loop, deve ser executado somente 1x)
	contract := nw.GetContract(chaincodeName)
	resp, err := contract.SubmitTransaction("CreateVehicleWallet", "ABC1234")
	if err != nil {
		log.Errorf("Failed submit transaction: %s", err)
		return
	}
	log.Println(string(resp))

	// resp, err = contract.SubmitTransaction("GiveCredits", "ABC1234", "330")
	// if err != nil {
	// 	log.Errorf("Failed submit transaction: %s", err)
	// 	return
	// }
	// log.Println(string(resp))

	for i := 0; i < 2754; i++ {

		fmt.Printf("Linha %v de %v\n", i, len(timestamps)-1)

		unixTimestamp, err := ConvertTimestampToUnix(timestamps[i])
		if err != nil {
			log.Fatalf("Erro ao converter o timestamp: %s", err)
		}

		cleanedLatitude, err := SanitizeFloatString(lats[i])
		if err != nil {
			log.Fatalf("Erro ao limpar a latitude: %s", err)
		}

		cleanedLongitude, err := SanitizeFloatString(lons[i])
		if err != nil {
			log.Fatalf("Erro ao limpar a longitude: %s", err)
		}

		fmt.Println("Timestamps:", timestamps[i])
		fmt.Println("Unix Time:", unixTimestamp)
		fmt.Println("Latitudes:", lats[i])
		fmt.Println("Longitudes:", lons[i])
		fmt.Println("Cleaned Latitude:", cleanedLatitude)
		fmt.Println("Cleaned Longitude:", cleanedLongitude)
		fmt.Println("Velocidades do veículo:", vehicleSpeeds[i])
		fmt.Println("Aceleração X:", accel_x[i])
		fmt.Println("Aceleração Y:", accel_y[i])
		fmt.Println("Aceleração Z:", accel_z[i])

		var flag string
		if i != 0 && i%10 == 0 {
			flag = "true"
		} else {
			flag = "false"
		}
		fmt.Println(flag)

		// contract = nw.GetContract(chaincodeName)
		// resp, err = contract.EvaluateTransaction("QueryVehicleData", "ABC1234")
		// if err != nil {
		// 	log.Error("Failed submit transaction: %s", err)
		// 	return
		// }
		// log.Info(string(resp))

		contract = nw.GetContract(chaincodeName)
		resp, err := contract.SubmitTransaction("StoreVehicleData", "ABC1234", unixTimestamp, cleanedLatitude, cleanedLongitude, vehicleSpeeds[i], accel_x[i], accel_y[i], accel_z[i], flag)
		if err != nil {
			log.Errorf("Failed submit transaction: %s", err)
			return
		}
		log.Info(resp)

		// contract := nw.GetContract(chaincodeName)
		// resp, err = contract.SubmitTransaction("StoreSimpleVehicleData", "ABC1234", unixTimestamp, cleanedLatitude, cleanedLongitude, "0", vehicleSpeeds[i], accel_x[i], accel_y[i], accel_z[i], flag)
		// if err != nil {
		// 	log.Errorf("Failed submit transaction: %s", err)
		// 	return
		// }
		// log.Info(resp)

		contract = nw.GetContract(chaincodeName)
		resp, err = contract.SubmitTransaction("AnalyzeDriverBehavior", "ABC1234")
		if err != nil {
			log.Errorf("Failed submit transaction: %s", err)
			return
		}
		log.Info(resp)

		// contract = nw.GetContract(chaincodeName)
		resp, err = contract.EvaluateTransaction("QueryVehicleWallet", "ABC1234")
		if err != nil {
			log.Errorf("Failed submit transaction: %s", err)
			return
		}
		log.Info(string(resp))

		// resp, err = contract.EvaluateTransaction("QueryVehicleData", "ABC1234")
		// if err != nil {
		// 	log.Errorf("Failed submit transaction: %s", err)
		// 	return
		// }
		// log.Info(string(resp))
	}

}

func registerEnrollUser(configFilePath, enrollID, mspID string) {
	log.Info("Registering User : ", enrollID)
	sdk, err := fabsdk.New(config.FromFile(configFilePath))
	if err != nil {
		log.Errorf("Failed to create SDK: %s", err)
		return
	}
	ctx := sdk.Context()
	caClient, err := mspclient.New(
		ctx,
		mspclient.WithCAInstance("inmetro-ca.default"),
		mspclient.WithOrg(mspID),
	)

	if err != nil {
		log.Errorf("Failed to create msp client: %s\n", err)
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
	if err != nil {
		log.Error(err)
	}

	signingIdentity, err := caClient.GetSigningIdentity(enrollID)
	if err != nil {
		log.Error(err)
	}

	key, err := signingIdentity.PrivateKey().Bytes()
	if err != nil {
		log.Error(err)
	}

	identity := gateway.NewX509Identity(mspID, string(signingIdentity.EnrollmentCertificate()), string(key))

	err = wallet.Put(enrollID, identity)
	if err != nil {
		log.Error(err)
	}

}

func queryCCgw(configFilePath, channelName, userName, mspID, chaincodeName, fcn string) {

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

	resp, err := contract.EvaluateTransaction(fcn)

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

func ReadCSV() ([]string, []string, []string, []string, []string, []string, []string, error) {
	// Abrir o arquivo CSV
	file, err := os.Open("data/obd_clean.csv")
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

	// Ler todas as linhas
	for {
		record, err := reader.Read()
		if err == csv.ErrFieldCount {
			continue
		}
		if err != nil {
			break
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

// ConvertTimestampToUnix converts a timestamp string to a Unix time string
func ConvertTimestampToUnix(timestamp string) (string, error) {
	layout := "2006-01-02 15:04:05.000"
	t, err := time.Parse(layout, timestamp)
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
