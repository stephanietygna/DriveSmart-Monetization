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
type VehicleData struct { // pk: idcarro / placa do veiculo
	Latitude  string `json:"latitude"`  // Mudança Brusca de Direção
	Longitude string `json:"longitude"` // Mudança Brusca de Direção
	Direction string `json:"direction"` // Mudança Brusca de Direção
	Speed     string `json:"speed"`     // Detecção de Aceleração Anômala // Mudança Brusca de Direção
	AccelX    string `json:"accelX"`    //zigue-zague
	AccelY    string `json:"accelY"`    //zigue-zague
	AccelZ    string `json:"accelZ"`    //zigue-zague // A aceleração em Z pode ser útil para detectar comportamentos relacionados a movimentos verticais // como subidas, descidas ou saltos, especialmente em terrenos irregulares.
	TimeStamp string `json:"timestamp"` //Detecção de Aceleração Anômala
	Flag      string `json:"flag"`      // controle de 10 em 10 linhas
}

type VehicleWallet struct { // pk: idcarro
	Credits int `json:"credits"`
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

// funcao x (
// 	// passar o history iterator para as 3 funções
// 	// se tiver algo de errado, penaliza, se não, premeia
// 	// armazenar o saldo
// 	// as 3 funções não serão mais chaincodes
// 	// retornarão true ou false
// 	DetectZigZag()
// 	DetectSharpTurn()
// 	DetectAnomalousAcceleration()
// )

func (s *SmartContract) AnalyzeDriverBehavior(ctx contractapi.TransactionContextInterface, idcarro string) error {
	// Recuperar o histórico de dados do veículo do ledger
	// [BUG] Ele lê o próximo mesmo que não tenha nada
	historyIterator, err := ctx.GetStub().GetHistoryForKey(idcarro)
	if err != nil {
		return fmt.Errorf("falha ao obter histórico de dados do veículo: %s", err)
	}
	defer historyIterator.Close()

	// Inicializar saldo
	var saldo int
	// Analisar cada registro histórico
	speedSlice := []string{}
	// timestampSlice := []string{}
	accelXSlice := []string{}
	accelYSlice := []string{}
	accelZSlice := []string{}
	flagSlice := []string{}
	directionSlice := []string{}

	// Iterar sobre o histórico e aplicar análises
	for historyIterator.HasNext() {
		historyEntry, err := historyIterator.Next()
		if err != nil {
			return fmt.Errorf("falha ao iterar sobre o histórico de dados do veículo: %s", err)
		}

		var historicalData VehicleData
		err = json.Unmarshal(historyEntry.Value, &historicalData)
		if err != nil {
			return fmt.Errorf("falha ao desserializar dados históricos do veículo: %s", err)
		}

		// speed, err := strconv.ParseFloat(historicalData.Speed, 32)
		// log.Printf("Velocidade (MK1): %v", historicalData.Speed)
		// if err != nil {
		// 	if historicalData.Speed == "" {
		// 		break
		// 	} else {
		// 		return fmt.Errorf("falha ao converter velocidade histórica: %s", err)
		// 	}
		// }
		// timestamp, err := strconv.ParseInt(historicalData.TimeStamp, 10, 32)
		// if err != nil {
		// 	return fmt.Errorf("falha ao converter timestamp histórico: %s", err)
		// }

		flagSlice = append(flagSlice, historicalData.Flag)
		speedSlice = append(speedSlice, historicalData.Speed)
		// timestampSlice = append(timestampSlice, historicalData.TimeStamp)
		directionSlice = append(directionSlice, historicalData.Direction)

		// converter aceleração para float
		// accelX, err := strconv.ParseFloat(historicalData.AccelX, 64)
		// if err != nil {
		// 	return fmt.Errorf("falha ao converter aceleração X histórica: %s", err)
		// }
		// accelY, err := strconv.ParseFloat(historicalData.AccelY, 64)
		// if err != nil {
		// 	return fmt.Errorf("falha ao converter aceleração Y histórica: %s", err)
		// }
		// accelZ, err := strconv.ParseFloat(historicalData.AccelZ, 64)
		// if err != nil {
		// 	return fmt.Errorf("falha ao converter aceleração Z histórica: %s", err)
		// }

		// append accel history to accelSlice
		accelXSlice = append(accelXSlice, historicalData.AccelX)
		accelYSlice = append(accelYSlice, historicalData.AccelY)
		accelZSlice = append(accelZSlice, historicalData.AccelZ)
		// fmt.Println(accelXSlice, accelYSlice, accelZSlice, flagSlice, speedSlice, directionSlice)

		// interrompe a execução após 10 registros
		if len(speedSlice) == 10 {
			break
		}
	}

	// Detectar zigue-zague se o primeiro flag for true
	// serão executados a cada 10 linhas/segundos
	if len(flagSlice) > 0 && flagSlice[0] == "true" {
		credZigZag := DetectZigZag(accelXSlice, accelYSlice, accelZSlice)
		credAnomalia, err := DetectAnomalousAcceleration(speedSlice)
		if err != nil {
			return fmt.Errorf("erro ao detectar aceleração anômala: %s", err)
		}
		saldo += credZigZag
		saldo += credAnomalia
	}
	// Detectar curvas bruscas
	credCurva := DetectSharpTurn(speedSlice[0], directionSlice[0])

	// Atualizar o saldo na carteira do cliente
	walletKey, err := ctx.GetStub().CreateCompositeKey("WALLET", []string{idcarro})
	if err != nil {
		return fmt.Errorf("erro ao criar chave composta para a carteira: %s", err)
	}

	vehicleWalletAsBytes, err := ctx.GetStub().GetState(walletKey)
	if err != nil {
		return fmt.Errorf("erro ao recuperar o saldo atual da carteira: %s", err)
	}

	vehicleWallet := VehicleWallet{}
	if vehicleWalletAsBytes != nil {
		err = json.Unmarshal(vehicleWalletAsBytes, &vehicleWallet)
		if err != nil {
			return fmt.Errorf("falha ao desserializar a carteira do veículo: %s", err)
		}
	}

	saldo += credCurva

	vehicleWallet.Credits += saldo

	vehicleWalletJSON, err := json.Marshal(vehicleWallet)
	if err != nil {
		return fmt.Errorf("falha ao serializar a carteira do veículo: %s", err)
	}

	// Salvar o registro atual no ledger
	return ctx.GetStub().PutState(walletKey, vehicleWalletJSON)
}

// StoreVehicleData armazena os dados do veículo no ledger
func (s *SmartContract) StoreVehicleData(ctx contractapi.TransactionContextInterface, idcarro string, unixTimestamp string, latitudeStr string, longitudeStr string, speedStr string, accelXstr string, accelYstr string, accelZstr string, flag string) error {
	// Recuperar dados anteriores para calcular a direção
	previousDataJSON, err := ctx.GetStub().GetState(idcarro)
	if err != nil {
		return fmt.Errorf("falha ao ler os dados do veículo do ledger: %s", err)
	}

	if previousDataJSON != nil {
		var previousVehicleData VehicleData
		err = json.Unmarshal(previousDataJSON, &previousVehicleData)
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
		previousLatitude, err := strconv.ParseFloat(previousVehicleData.Latitude, 64)
		if err != nil {
			return fmt.Errorf("falha ao converter latitude anterior: %s", err)
		}
		previousLongitude, err := strconv.ParseFloat(previousVehicleData.Longitude, 64)
		if err != nil {
			return fmt.Errorf("falha ao converter longitude anterior: %s", err)
		}
		direction := CalculateBearing(previousLatitude, previousLongitude, latitude, longitude)
		// Criar a estrutura VehicleData
		vehicleData := VehicleData{
			Latitude:  latitudeStr,
			Longitude: longitudeStr,
			Direction: fmt.Sprintf("%f", direction),
			Speed:     speedStr,
			AccelX:    accelXstr,
			AccelY:    accelYstr,
			AccelZ:    accelZstr,
			TimeStamp: unixTimestamp,
			Flag:      flag,
		}

		// Armazenar os dados no ledger
		vehicleDataJSON, err := json.Marshal(vehicleData)
		if err != nil {
			return fmt.Errorf("falha ao serializar os dados do veículo: %s", err)
		}

		return ctx.GetStub().PutState(idcarro, vehicleDataJSON)
		// if err != nil {
		// 	return fmt.Errorf("falha ao armazenar os dados do veículo no ledger: %s", err)
		// }

	}
	// ELSE
	// Se não há dados anteriores, armazenar apenas a latitude e longitude
	newVehicleData := VehicleData{
		Latitude:  latitudeStr,
		Longitude: longitudeStr,
		Direction: "0", // inicialmente, a direção pode ser 0
		Speed:     speedStr,
		AccelX:    accelXstr,
		AccelY:    accelYstr,
		AccelZ:    accelZstr,
		TimeStamp: unixTimestamp,
		Flag:      flag,
	}

	// Armazenar os dados no ledger
	newVehicleDataJSON, err := json.Marshal(newVehicleData)
	if err != nil {
		return fmt.Errorf("falha ao serializar os dados do veículo: %s", err)
	}

	return ctx.GetStub().PutState(idcarro, newVehicleDataJSON)
	// if err != nil {
	// 	return fmt.Errorf("falha ao armazenar os dados do veículo no ledger: %s", err)
	// }

	// return nil
}

// StoreSimpleVehicleData armazena os dados do veículo no ledger sem verificação extra, f
func (s *SmartContract) StoreSimpleVehicleData(ctx contractapi.TransactionContextInterface, idcarro string, unixTimestamp string, latitudeStr string, longitudeStr string, speedStr, direction string, accelXstr string, accelYstr string, accelZstr string, flag string) error {
	vehicleData := VehicleData{
		Latitude:  latitudeStr,
		Longitude: longitudeStr,
		Direction: direction,
		Speed:     speedStr,
		AccelX:    accelXstr,
		AccelY:    accelYstr,
		AccelZ:    accelZstr,
		TimeStamp: unixTimestamp,
		Flag:      flag,
	}

	vehicleDataJSON, err := json.Marshal(vehicleData)
	if err != nil {
		return fmt.Errorf("falha ao serializar os dados do veículo: %s", err)
	}

	return ctx.GetStub().PutState(idcarro, vehicleDataJSON)
}

// InitVehicleWallet inicializa uma carteira de veículo com quantidade inicial de créditos 0
func (s *SmartContract) CreateVehicleWallet(ctx contractapi.TransactionContextInterface, idcarro string) error {
	// verifique se a carteira já existe
	indexName := "WALLET"
	compositeKey, err := ctx.GetStub().CreateCompositeKey(indexName, []string{idcarro})
	if err != nil {
		return err
	}

	// verificação de integridade referencial
	vehicleWalletAsBytes, err := ctx.GetStub().GetState(compositeKey)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %s", err)
	}

	if vehicleWalletAsBytes != nil {
		return fmt.Errorf("carteira já existe para o veiculo %s", idcarro)
	}

	vehicleWallet := VehicleWallet{
		Credits: 0,
	}

	vehicleWalletJSON, err := json.Marshal(vehicleWallet)
	if err != nil {
		return fmt.Errorf("falha ao serializar a carteira do veículo: %s", err)
	}

	return ctx.GetStub().PutState(compositeKey, vehicleWalletJSON)
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

	log.Printf("Créditos: %v", vehicleWallet.Credits)
	// repeat string
	// fmt.Println(strings.Repeat("=", 10))

	return &vehicleWallet, nil
}

func (s *SmartContract) TestRichQuery(ctx contractapi.TransactionContextInterface, query string) error {
	queryString := fmt.Sprintf(`{"selector":{"timestamp":"%s"}}`, query) // ABC4444
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)

	if err != nil {
		return fmt.Errorf("falha ao consultar o ledger: %s", err)
	}

	defer resultsIterator.Close()

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return fmt.Errorf("falha ao iterar sobre os resultados da consulta: %s", err)
		}

		var vehicleData VehicleData
		err = json.Unmarshal(queryResponse.Value, &vehicleData)
		if err != nil {
			return fmt.Errorf("falha ao desserializar os dados do veículo: %s", err)
		}

		// log.Println("Registro: ", &vehicleData)
	}

	return nil
}

// DetectAnalousAcceleration verifica a anomalia e atualiza a carteira do veículo de acordo
func DetectAnomalousAcceleration(speedSlice []string) (int, error) {

	// Calcular aceleração anômala
	anomalous := false
	credits := 10

	// calcula o delta de velocidade e tempo entre o tempo final - o de [5 segundos atrás]
	// deltaSpeed := math.Abs(speedSlice[i] - speedSlice[i-1])
	// deltaTime := timestampSlice[i] - timestampSlice[i-1]

	tamanho := len(speedSlice)
	valorfinal, err := strconv.ParseFloat(speedSlice[tamanho-1], 32)
	if err != nil {
		return 0, fmt.Errorf("erro ao converter valor para float64: %v", err.Error()) // or handle the error appropriately
	}
	valorinicial, err := strconv.ParseFloat(speedSlice[0], 32)
	if err != nil {
		return 0, fmt.Errorf("erro ao converter valor para float64: %v", err.Error()) // or handle the error appropriately
	}

	deltaSpeed := math.Abs(valorfinal - valorinicial)

	// Se a variação de velocidade for maior que 30 km/h em menos de 5 segundos
	if deltaSpeed > 30 {
		anomalous = true
		credits = -50
	}

	log.Print("Anomalia: ", anomalous)

	return credits, nil
}

// Função para detectar comportamento de zigue-zague

// pegar cerca de 10 segundos de linhas
// então, comparar segundo[9] com segundo [8] OU com segundo[9] com segundo[7]
// ex: comparar o sinal atual com o de 2 segundos antes

func DetectZigZag(accelXSlice []string, accelYSlice []string, accelZSlice []string) int {
	// Cria a chave composta para a carteira
	// Variáveis para comparação e contagem de zigue-zague
	var zigzagCount int
	credits := 10      // Define o valor da penalização ou recompensa
	detection := false // Define se houve zigue-zague

	// ele vai ler de tras para frente (do mais antigo até o mais recente)
	for i := len(accelXSlice) - 1; i > 0; i-- {
		// Recupera últimos valores de aceleração para detectar zigue-zague

		// parametros removidos
		// currentAccelX := accelXSlice[i]
		// nextAccelX := accelXSlice[i-1]
		// nextAccelY := accelYSlice[i-1]

		currentAccelY, err := strconv.ParseFloat(accelYSlice[i], 64)
		if err != nil {
			log.Printf("Erro ao converter aceleração Y: %v", err)
			continue
		}
		currentAccelZ := accelZSlice[i]
		nextAccelZ := accelZSlice[i-1]

		// Compara os valores para detectar zigue-zague
		if currentAccelY >= 0.0080 && currentAccelZ != nextAccelZ {
			zigzagCount++
		}
	}

	// Se o número de zigue-zagues for maior ou igual a 3, aplica penalização
	if zigzagCount >= 3 {
		// Aplique penalização na carteira do veículo
		credits = -40
		detection = true
	}

	// Caso contrário, o veículo está dirigindo de forma aceitável
	log.Printf("Zigue-zague: %v", detection)

	return credits
}

// Função para detectar mudanças bruscas de direção
func DetectSharpTurn(speed string, direction string) int {
	credits := 0
	flag := false

	// Conversão da direção para float
	directionFloat, err := strconv.ParseFloat(direction, 64)
	if err != nil {
		fmt.Println("Erro ao converter direção:", err)
	}

	// debug
	// log.Printf("Direção: %v", direction)
	// log.Printf("Velocidade (MK2): %v", speed)

	//converter
	speedFloat, err := strconv.ParseFloat(speed, 64)
	if err != nil {
		fmt.Println("Erro ao converter velocidade:", err)
	}

	// Se a direção for maior que 0.7 rad e a velocidade maior que 30 km/h
	if directionFloat == 0 {
		log.Printf("Direção neutra")
		credits += 10
	} else if directionFloat < 0.7 && speedFloat > 30 {
		credits -= 30 // Penalidade de -30 créditos
		flag = true
	} else {
		credits += 10
	}

	log.Printf("Curva brusca: %v.", flag)
	return credits
}

// CalculateBearing calcula a direção entre dois pontos geográficos
// CalculateBearing calcula a direção entre dois pontos geográficos
func CalculateBearing(lat1, lon1, lat2, lon2 float64) float64 {
	deltaLon := lon2 - lon1

	x := math.Cos(lat2*math.Pi/180) * math.Sin(deltaLon*math.Pi/180)
	y := math.Cos(lat1*math.Pi/180)*math.Sin(lat2*math.Pi/180) -
		math.Sin(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Cos(deltaLon*math.Pi/180)

	bearing := math.Atan2(x, y)
	if bearing < 0 {
		bearing += 2 * math.Pi
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

func (s *SmartContract) GiveCredits(ctx contractapi.TransactionContextInterface, idcarro string, credits int) error {
	indexName := "WALLET"
	compositeKey, err := ctx.GetStub().CreateCompositeKey(indexName, []string{idcarro})
	if err != nil {
		return fmt.Errorf("erro ao criar chave composta para a carteira: %s", err)
	}

	vehicleWalletAsBytes, err := ctx.GetStub().GetState(compositeKey)
	if err != nil {
		return fmt.Errorf("erro ao recuperar o saldo atual da carteira: %s", err)
	}

	if vehicleWalletAsBytes == nil {
		return fmt.Errorf("carteira do veículo não encontrada")
	}

	var vehicleWallet VehicleWallet
	err = json.Unmarshal(vehicleWalletAsBytes, &vehicleWallet)
	if err != nil {
		return fmt.Errorf("falha ao desserializar a carteira do veículo: %s", err)
	}

	vehicleWallet.Credits += credits

	vehicleWalletJSON, err := json.Marshal(vehicleWallet)
	if err != nil {
		return fmt.Errorf("falha ao serializar a carteira do veículo: %s", err)
	}

	return ctx.GetStub().PutState(compositeKey, vehicleWalletJSON)
}

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
