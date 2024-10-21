package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3/log"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

// VehicleData representa os dados do veículo
type VehicleData struct {
	Speed      string `json:"speed"`
	Direction  string `json:"direction"`
	TimeStamps string `json:"timestamps"`
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

// StoreVehicleData armazena os dados do veículo no ledger
func (s *SmartContract) StoreVehicleData(ctx contractapi.TransactionContextInterface, idcarro string, speedStr string, directionStr string, timestampStr string) error {
	// Criar a estrutura VehicleData
	vehicleData := VehicleData{
		Speed:      speedStr,
		Direction:  directionStr,
		TimeStamps: timestampStr,
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

// 	alousAcceleration verifica a anomalia e atualiza a carteira do veículo de acordo
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

	// Converter strings para slices de float64
	speedSlice, err := ConvertStringToFloatSlice(vehicleData.Speed)
	if err != nil {
		return "", fmt.Errorf("falha ao converter velocidade: %s", err)
	}

	timestampSlice, err := ConvertStringToFloatSlice(vehicleData.TimeStamps)
	if err != nil {
		return "", fmt.Errorf("falha ao converter timestamps: %s", err)
	}

	if len(speedSlice) != len(timestampSlice) {
		return "", fmt.Errorf("o número de timestamps e velocidades deve ser igual")
	}

	// Calcular aceleração anômala
	anomalous := false
	credits := 10

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

	zigZagCount := 0

	// Separar as direções
	directions := strings.Fields(vehicleData.Direction)

	// Verifica se há pelo menos 2 direções para comparar
	if len(directions) < 2 {
		return fmt.Errorf("não há dados suficientes para detectar zigue-zague")
	}

	for i := 1; i < len(directions); i++ {
		// Converter a direção atual
		directionCurrent, err := strconv.ParseFloat(directions[i], 64)
		if err != nil {
			fmt.Println("Erro ao converter direção atual:", err)
			continue // Pular caso haja erro na conversão
		}

		// Converter a direção anterior
		directionPrevious, err := strconv.ParseFloat(directions[i-1], 64)
		if err != nil {
			fmt.Println("Erro ao converter direção anterior:", err)
			continue // Pular caso haja erro na conversão
		}

		// Verifica se a direção mudou de positivo para negativo ou vice-versa
		if directionCurrent*directionPrevious < 0 {
			zigZagCount++
		}
	}

	credits := 0
	infringement := false

	// Definindo 3 mudanças de faixa como zigue-zague
	if zigZagCount >= 3 {
		infringement = true
		credits = -40
	} else {
		credits = 10
		infringement = false
	}

	log.Printf("Zigue-zague:", infringement, "Créditos:", credits)

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

	// atualiza vehiclewallet com a penalidade de -40 créditos
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
			credits = -30// Penalidade de -30 créditos
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
