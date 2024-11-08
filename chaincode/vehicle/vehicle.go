package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	// "github.com/emicklei/go-restful/v3/log"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

// VehicleData representa os dados do veículo
type VehicleData struct {
	Latitude   string `json:"latitude"`   // Mudança Brusca de Direção
	Longitude  string `json:"longitude"`  // Mudança Brusca de Direção
	Direction  string `json:"direction"`  // Mudança Brusca de Direção
	Speed      string `json:"speed"`      // Detecção de Aceleração Anômala // Mudança Brusca de Direção
	AccelX     string `json:"accelX"`     //zigue-zague
	AccelY     string `json:"accelY"`     //zigue-zague
	AccelZ     string `json:"accelZ"`     //zigue-zague // A aceleração em Z pode ser útil para detectar comportamentos relacionados a movimentos verticais // como subidas, descidas ou saltos, especialmente em terrenos irregulares.
	TimeStamps string `json:"timestamps"` //Detecção de Aceleração Anômala
}

type VehicleWallet struct { // pk: idcarro
	Credits string `json:"credits"`
}

// ConvertStringToFloatSlice converte uma string de números separados por espaço em um slice de float64
func ConvertStringToFloatSlice(data string) ([]float64, error) {
	parts := strings.Fields(data)
	var result []float64
	for _, part := range parts {
		value, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

// MODIFICADO E TESTADO, FUNCIONAL
// StoreVehicleData armazena os dados do veículo no ledger
func (s *SmartContract) StoreVehicleData(ctx contractapi.TransactionContextInterface, idcarro string, unixTimestamp string, latitudeStr string, longitudeStr string, speedStr string, accelXstr string, accelYstr string, accelZstr string) error {
	// Recuperar dados anteriores para calcular a direção
	previousDataJSON, err := ctx.GetStub().GetState(idcarro)
	if err != nil {
		return fmt.Errorf("falha ao ler os dados do veículo do ledger: %s", err)
	}

	var previousData VehicleData
	if previousDataJSON != nil {
		err = json.Unmarshal(previousDataJSON, &previousData)
		if err != nil {
			return fmt.Errorf("falha ao desserializar os dados do veículo: %s", err)
		}

		// Calcular a direção
		latitude, err := strconv.ParseFloat(latitudeStr, 64)
		if err != nil {
			return fmt.Errorf("falha ao converter latitude: %s", err)
		}
		longitude, err := strconv.ParseFloat(longitudeStr, 64)
		if err != nil {
			return fmt.Errorf("falha ao converter longitude: %s", err)
		}
		previousLatitude, err := strconv.ParseFloat(previousData.Latitude, 64)
		if err != nil {
			return fmt.Errorf("falha ao converter latitude anterior: %s", err)
		}
		previousLongitude, err := strconv.ParseFloat(previousData.Longitude, 64)
		if err != nil {
			return fmt.Errorf("falha ao converter longitude anterior: %s", err)
		}
		direction := CalculateBearing(previousLatitude, previousLongitude, latitude, longitude)
		// Criar a estrutura VehicleData
		vehicleData := VehicleData{
			Latitude:   latitudeStr,
			Longitude:  longitudeStr,
			Direction:  fmt.Sprintf("%f", direction),
			Speed:      speedStr,
			AccelX:     accelXstr,
			AccelY:     accelYstr,
			AccelZ:     accelZstr,
			TimeStamps: unixTimestamp,
		}

		// Armazenar os dados no ledger
		vehicleDataJSON, err := json.Marshal(vehicleData)
		if err != nil {
			return fmt.Errorf("falha ao serializar os dados do veículo: %s", err)
		}

		err = ctx.GetStub().PutState(idcarro, vehicleDataJSON)
		if err != nil {
			return fmt.Errorf("falha ao armazenar os dados do veículo no ledger: %s", err)
		}

	} else {
		// Se não há dados anteriores, armazenar apenas a latitude e longitude
		vehicleData := VehicleData{
			Latitude:   latitudeStr,
			Longitude:  longitudeStr,
			Direction:  "0", // inicialmente, a direção pode ser 0
			Speed:      speedStr,
			AccelX:     accelXstr,
			AccelY:     accelYstr,
			AccelZ:     accelZstr,
			TimeStamps: unixTimestamp,
		}

		// Armazenar os dados no ledger
		vehicleDataJSON, err := json.Marshal(vehicleData)
		if err != nil {
			return fmt.Errorf("falha ao serializar os dados do veículo: %s", err)
		}

		err = ctx.GetStub().PutState(idcarro, vehicleDataJSON)
		if err != nil {
			return fmt.Errorf("falha ao armazenar os dados do veículo no ledger: %s", err)
		}
	}

	return nil
}

// InitVehicleWallet inicializa uma carteira de veículo com quantidade inicial de créditos 0
func (s *SmartContract) CreateVehicleWallet(ctx contractapi.TransactionContextInterface, idcarro string) error {
	// verifique se a carteira já existe
	indexName := "WALLET"
	compositeKey, err := ctx.GetStub().CreateCompositeKey(indexName, []string{idcarro})
	if err != nil {
		return err
	}

	// // verificação de integridade referencial
	// vehicleWalletAsBytes, err := ctx.GetStub().GetState(compositeKey)
	// if err != nil {
	// 	return fmt.Errorf("failed to read from world state: %s", err)
	// }

	// if vehicleWalletAsBytes != nil {
	// 	return fmt.Errorf("carteira já existe para o veiculo %s", idcarro)
	// }

	vehicleWallet := VehicleWallet{
		Credits: "0",
	}

	vehicleWalletJSON, err := json.Marshal(vehicleWallet)
	if err != nil {
		return fmt.Errorf("falha ao serializar a carteira do veículo: %s", err)
	}

	err = ctx.GetStub().PutState(compositeKey, vehicleWalletJSON)
	if err != nil {
		return fmt.Errorf("falha ao armazenar a carteira do veículo no ledger: %s", err)
	}

	return nil
}

// QueryVehicleWallet consulta a carteira do veículo armazenada no ledger
func (s *SmartContract) QueryVehicleWallet(ctx contractapi.TransactionContextInterface, idcarro string) (*VehicleWallet, error) {
	indexName := "WALLET"
	compositeKey, err := ctx.GetStub().CreateCompositeKey(indexName, []string{idcarro})
	if err != nil {
		return nil, err
	}

	vehicleWalletAsBytes, err := ctx.GetStub().GetState(compositeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %s", err)
	}

	if vehicleWalletAsBytes == nil {
		return nil, fmt.Errorf("carteira do veículo não encontrada")
	}

	var vehicleWallet VehicleWallet
	err = json.Unmarshal(vehicleWalletAsBytes, &vehicleWallet)
	if err != nil {
		return nil, fmt.Errorf("falha ao desserializar a carteira do veículo: %s", err)
	}

	return &vehicleWallet, nil
}

// NÃO TESTEI AINDA
// DetectAnalousAcceleration verifica a anomalia e atualiza a carteira do veículo de acordo
func (s *SmartContract) DetectAnomalousAcceleration(ctx contractapi.TransactionContextInterface, idcarro string) (string, error) {
	// Recuperar os dados do veículo do ledger
	vehicleDataJSON, err := ctx.GetStub().GetState(idcarro)
	if err != nil {
		return "", fmt.Errorf("falha ao ler os dados do veículo do ledger: %s", err)
	}
	if vehicleDataJSON == nil {
		return "", fmt.Errorf("dados do veículo não encontrados")
	}

	var vehicleData VehicleData
	err = json.Unmarshal(vehicleDataJSON, &vehicleData)
	if err != nil {
		return "", fmt.Errorf("falha ao desserializar os dados do veículo: %s", err)
	}

	// Calcular aceleração anômala
	anomalous := false
	credits := 10

	// Recuperar o histórico de dados do veículo do ledger
	historyIterator, err := ctx.GetStub().GetHistoryForKey(idcarro)
	if err != nil {
		return "", fmt.Errorf("falha ao obter histórico de dados do veículo: %s", err)
	}
	defer historyIterator.Close()

	var speedSlice []float64
	var timestampSlice []int64

	// Iterar sobre o histórico e coletar dados de velocidade e timestamps
	for historyIterator.HasNext() {
		historyEntry, err := historyIterator.Next()
		if err != nil {
			return "", fmt.Errorf("falha ao iterar sobre o histórico de dados do veículo: %s", err)
		}

		var historicalData VehicleData
		err = json.Unmarshal(historyEntry.Value, &historicalData)
		if err != nil {
			return "", fmt.Errorf("falha ao desserializar dados históricos do veículo: %s", err)
		}

		speed, err := strconv.ParseFloat(historicalData.Speed, 64)
		if err != nil {
			return "", fmt.Errorf("falha ao converter velocidade histórica: %s", err)
		}
		timestamp, err := strconv.ParseInt(historicalData.TimeStamps, 10, 64)
		if err != nil {
			return "", fmt.Errorf("falha ao converter timestamp histórico: %s", err)
		}

		speedSlice = append(speedSlice, speed)
		timestampSlice = append(timestampSlice, timestamp)

		// Parar se a diferença de tempo for maior que 5 minutos (300 segundos)
		if len(timestampSlice) > 1 && (timestampSlice[0]-timestampSlice[len(timestampSlice)-1]) > 300 {
			break
		}
	}

	// calcula o delta de velocidade e tempo entre cada timestamp
	for i := 1; i < len(speedSlice); i++ {
		deltaSpeed := math.Abs(speedSlice[i] - speedSlice[i-1])
		deltaTime := timestampSlice[i] - timestampSlice[i-1]

		// Se a variação de velocidade for maior que 30 km/h em menos de 10 segundos
		if deltaSpeed > 30 && deltaTime <= 10 {
			anomalous = true
			credits = -50
			break
		}
	}

	log.Printf("Anomalia:", anomalous)

	// Obtém o vehiclewallet atual
	indexName := "WALLET"
	compositeKey, err := ctx.GetStub().CreateCompositeKey(indexName, []string{idcarro})
	if err != nil {
		return "", err
	}

	vehicleWalletAsBytes, err := ctx.GetStub().GetState(compositeKey)
	if err != nil {
		return "", fmt.Errorf("failed to read from world state: %s", err)
	}

	if vehicleWalletAsBytes == nil {
		return "", fmt.Errorf("carteira do veículo não encontrada")
	}

	// atualiza vehiclewallet com o novo valor de créditos
	var vehicleWallet VehicleWallet
	_ = json.Unmarshal(vehicleWalletAsBytes, &vehicleWallet)
	currentCredits, err := strconv.Atoi(vehicleWallet.Credits)
	if err != nil {
		return "", fmt.Errorf("falha ao converter créditos atuais: %s", err)
	}
	vehicleWallet.Credits = strconv.Itoa(currentCredits + credits)

	vehicleWalletJSON, err := json.Marshal(vehicleWallet)

	return "", ctx.GetStub().PutState(compositeKey, vehicleWalletJSON)
}

// NÃO MODIFIQUEI/TESTEI AINDA
// Função para detectar comportamento de zigue-zague
func (s *SmartContract) DetectZigZag(ctx contractapi.TransactionContextInterface, idcarro string) error {
	// Recuperar os dados do veículo do ledger
	vehicleDataJSON, err := ctx.GetStub().GetState(idcarro)
	if err != nil {
		return fmt.Errorf("falha ao ler os dados do veículo do ledger: %s", err)
	}
	if vehicleDataJSON == nil {
		return fmt.Errorf("dados do veículo não encontrados")
	}

	var vehicleData VehicleData
	err = json.Unmarshal(vehicleDataJSON, &vehicleData)
	if err != nil {
		return fmt.Errorf("falha ao desserializar os dados do veículo: %s", err)
	}

	// Inicializa as variáveis
	var prevAccelX, prevAccelY, prevAccelZ string
	var zigzagCount int

	// VERIFICAR: [acceldata não existe, creio que accel data deveria ser accelX, accelY, accelZ?]
	// Verifica se os dados de aceleração foram armazenados como uma string
	accelData := vehicleData.AccelData // Supondo que AccelData seja uma string "accelX_value accelY_value accelZ_value"

	// Se os dados de aceleração estiverem ausentes, retorna erro
	if accelData == "" {
		return fmt.Errorf("dados de aceleração não encontrados")
	}

	// VERIFICAR: [se accel data é uma string "accelX_value accelY_value accelZ_value", contendo 1 valor cada,  então não é necessário fazer o split]
	// Separa a string de aceleração em valores de aceleração X, Y e Z
	accelValues := strings.Fields(accelData) // Divide a string em partes, por exemplo: ["accelX_value", "accelY_value", "accelZ_value"]
	if len(accelValues) != 3 {
		return fmt.Errorf("formato inválido de dados de aceleração")
	}

	accelX := accelValues[0]
	accelY := accelValues[1]
	accelZ := accelValues[2]

	// VERIFICAR: [os valores de accel estão sendo armazenados individualmente, então deveria ser utilizada a função GetHistoryForkKey para recuperar os valores anteriores]
	// Compara os valores de aceleração para detectar zigue-zague (comparando as strings diretamente)
	if accelX != prevAccelX || accelY != prevAccelY || accelZ != prevAccelZ {
		zigzagCount++
	}

	// Atualiza os valores anteriores de aceleração
	prevAccelX = accelX
	prevAccelY = accelY
	prevAccelZ = accelZ

	// Se o número de zigue-zagues for maior ou igual a 3, aplica penalização
	if zigzagCount >= 3 {
		// Aplique penalização na carteira do veículo
		// VERIFICAR: [vehicleWallet utiliza chave composta. Precisa do "indexName: WALLET"]
		vehicleWallet, err := ctx.GetStub().GetState(idcarro)
		if err != nil {
			return err
		}

		// Penalização nos créditos do veículo
		// VERIFICAR: [penaltyAmount não foi definido]
		vehicleWallet.Credits -= penaltyAmount

		// Atualiza os dados da carteira no ledger
		vehicleWalletJSON, err := json.Marshal(vehicleWallet)
		if err != nil {
			return fmt.Errorf("erro ao serializar os dados da carteira: %s", err.Error())
		}

		err = ctx.GetStub().PutState(idcarro, vehicleWalletJSON)
		if err != nil {
			return fmt.Errorf("erro ao atualizar dados da carteira do veículo: %s", err.Error())
		}

		return fmt.Errorf("Zigue-zague detectado! Penalização aplicada. Créditos restantes: %f", vehicleWallet.Credits)
	}

	// Caso contrário, o veículo está dirigindo de forma aceitável
	return fmt.Errorf("Condução normal, sem zigue-zague detectado")
}

// NÃO MODIFIQUEI/TESTEI AINDA
// Função para detectar mudanças bruscas de direção
func (s *SmartContract) DetectSharpTurn(ctx contractapi.TransactionContextInterface, idcarro string) error {
	// Recuperar os dados do veículo do ledger
	vehicleDataJSON, err := ctx.GetStub().GetState(idcarro)
	if err != nil {
		return fmt.Errorf("falha ao ler os dados do veículo do ledger: %s", err)
	}
	if vehicleDataJSON == nil {
		return fmt.Errorf("dados do veículo não encontrados")
	}

	var vehicleData VehicleData
	err = json.Unmarshal(vehicleDataJSON, &vehicleData)
	if err != nil {
		return fmt.Errorf("falha ao desserializar os dados do veículo: %s", err)
	}

	// Separar os valores das strings
	speeds := strings.Fields(vehicleData.Speed)
	directions := strings.Fields(vehicleData.Direction)

	credits := 0
	flag := false

	for i := 0; i < len(directions); i++ {
		// Conversão da direção
		direction, err := strconv.ParseFloat(directions[i], 64)
		if err != nil {
			fmt.Println("Erro ao converter direção:", err)
			continue // Pular caso haja erro na conversão
		}

		// Conversão da velocidade
		speed, err := strconv.ParseFloat(speeds[i], 64)
		if err != nil {
			fmt.Println("Erro ao converter velocidade:", err)
			continue // Pular caso haja erro na conversão
		}

		// Se a direção for maior que 0.7 rad e a velocidade maior que 30 km/h
		if direction > 0.7 && speed > 30 {
			credits = -30 // Penalidade de -30 créditos
			flag = true
		} else {
			credits = 10

		}
	}

	log.Printf("Curva brusca:", flag)

	indexName := "WALLET"
	compositeKey, err := ctx.GetStub().CreateCompositeKey(indexName, []string{idcarro})
	if err != nil {
		return err
	}

	vehicleWalletAsBytes, err := ctx.GetStub().GetState(compositeKey)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %s", err)
	}

	if vehicleWalletAsBytes == nil {
		return fmt.Errorf("carteira do veículo não encontrada")
	}

	// atualiza vehiclewallet com a penalidade de créditos
	var vehicleWallet VehicleWallet
	_ = json.Unmarshal(vehicleWalletAsBytes, &vehicleWallet)

	currentCredits, err := strconv.Atoi(vehicleWallet.Credits)
	if err != nil {
		return fmt.Errorf("falha ao converter créditos atuais: %s", err)
	}
	vehicleWallet.Credits = strconv.Itoa(currentCredits + credits)
	vehicleWalletJSON, err := json.Marshal(vehicleWallet)

	if err != nil {
		return fmt.Errorf("falha ao serializar a carteira do veículo: %s", err)
	}

	return ctx.GetStub().PutState(compositeKey, vehicleWalletJSON)
}

// CalculateBearing calcula a direção entre dois pontos geográficos
func CalculateBearing(lat1, lon1, lat2, lon2 float64) float64 {
	deltaLon := lon2 - lon1

	x := math.Cos(lat2*math.Pi/180) * math.Sin(deltaLon*math.Pi/180)
	y := math.Cos(lat1*math.Pi/180)*math.Sin(lat2*math.Pi/180) -
		math.Sin(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Cos(deltaLon*math.Pi/180)

	bearing := math.Atan2(x, y) * 180 / math.Pi
	if bearing < 0 {
		bearing += 360
	}
	return bearing
}

// QueryVehicleData consulta os dados do veículo armazenados no ledger
func (s *SmartContract) QueryVehicleData(ctx contractapi.TransactionContextInterface, idcarro string) (*VehicleData, error) {
	vehicleDataJSON, err := ctx.GetStub().GetState(idcarro)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler os dados do veículo do ledger: %s", err)
	}
	if vehicleDataJSON == nil {
		return nil, fmt.Errorf("dados do veículo não encontrados")
	}

	var vehicleData VehicleData
	err = json.Unmarshal(vehicleDataJSON, &vehicleData)
	if err != nil {
		return nil, fmt.Errorf("falha ao desserializar os dados do veículo: %s", err)
	}

	return &vehicleData, nil
}

// // QueryVehicleAnomaly consulta os dados de anomalia do veículo armazenados no ledger
// func (s *SmartContract) QueryVehicleAnomaly(ctx contractapi.TransactionContextInterface, idcarro string) (*VehicleWallet, error) {
// 	anomalyResultJSON, err := ctx.GetStub().GetState(idcarro)

// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read from world state: %s", err)
// 	}

// 	if anomalyResultJSON == nil {
// 		return nil, fmt.Errorf("nenhuma anomalia foi encontrada para o veículo %s", idcarro)
// 	}

// 	var anomalyResult AnomalyResult
// 	_ = json.Unmarshal(anomalyResultJSON, &anomalyResult)

// 	return &anomalyResult, nil
// }

func main() {
	chaincode, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		fmt.Printf("Erro ao criar o chaincode: %s", err)
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Erro ao iniciar o chaincode: %s", err)
	}
}
