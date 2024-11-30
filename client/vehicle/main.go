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
	registerEnrollUser(configFilePath, enrollID, mspID)

	timestamps, lats, lons, vehicleSpeeds, accel_x, accel_y, accel_z, err := ReadCSV()

	if err != nil {
		log.Fatalf("Erro ao ler o CSV: %s", err)
	}

	invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "CreateVehicleWallet", []string{"ABC1234"})

	// le de 0 até o tamanho do slice timestamps
	linhas := 5351
	for i := 0; i <= linhas; i++ {

		fmt.Println(strings.Repeat("-", 30))
		// fmt.Printf("Posição: %v/%v \n", i, len(timestamps)-1)
		fmt.Printf("Posição: %v/%v \n", i, linhas)
		fmt.Println(strings.Repeat("-", 30))

		// start := time.Now().Unix()
		// spent := start - int64(before)
		// fmt.Println("tempo gasto em segundos: ", spent)

		// Example usage of ConvertTimestampToUnix
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

		//[erro ao colocar id q nao existe?]

		// armazenar dados veiculares
		invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "StoreVehicleData", []string{"ABC1234", unixTimestamp, cleanedLatitude, cleanedLongitude, vehicleSpeeds[i], accel_x[i], accel_y[i], accel_z[i]})

		// criar carteira do veiculo (executar apenas 1 vez a cada inicialização da rede)
		// invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "CreateVehicleWallet", []string{"ABC1234"})

		// consultar dados veiculares
		queryCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "QueryVehicleData", []string{"ABC1234"})

		/* FUNÇÕES DE ANÁLISE (NECESSITAM DA CARTEIRA CRIADA) */

		invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "DetectSharpTurn", []string{"ABC1234"})
		invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "DetectAnomalousAcceleration", []string{"ABC1234"})
		// o zigzag deve ser executado a cada 10 linhas/segundos
		if i != 0 && i%10 == 0 {
			invokeCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "DetectZigZag", []string{"ABC1234"})
		}

		// consultar carteira do veiculo
		queryCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "QueryVehicleWallet", []string{"ABC1234"})

		// testando richquery (GetQueryResult). retorna apenas o último estado, então não serve para resgatar historico
		// queryCCgw(configFilePath, channelName, enrollID, mspID, chaincodeName, "TestRichQuery", []string{"1732553439"})

		// before = int(start)
	}

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
