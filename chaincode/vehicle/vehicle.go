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
	Latitude   string `json:"latitude"`
	Longitude  string `json:"longitude"`
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
func (s *SmartContract) StoreVehicleData(ctx contractapi.TransactionContextInterface, idcarro string, latitudeStr string, longitudeStr string, speedStr string, timestampStr string) error {
	// Validações de entrada
	if !isValidLatitude(latitudeStr) || !isValidLongitude(longitudeStr) || !isValidSpeed(speedStr) {
		return fmt.Errorf("valores de entrada inválidos")
	}

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
		direction, err := CalculateBearing(previousData.Latitude, previousData.Longitude, latitudeStr, longitudeStr)
		if err != nil {
			return fmt.Errorf("falha ao calcular a direção: %s", err)
		}
		
		// Criar a estrutura VehicleData
		vehicleData := VehicleData{
			Latitude:   latitudeStr,
			Longitude:  longitudeStr,
			Speed:      speedStr,
			Direction:  direction, // Corrigido de directionStr para direction
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

	} else {
		// Se não há dados anteriores, armazenar apenas a latitude e longitude
		vehicleData := VehicleData{
			Latitude:   latitudeStr,
			Longitude:  longitudeStr,
			Speed:      speedStr,
			Direction:  "0", // Inicialmente, a direção pode ser 0
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
	}

	return nil
}

// InitVehicleWallet inicializa uma carteira de veículo com quantidade inicial de créditos 0
func (s *SmartContract) CreateVehicleWallet(ctx contractapi.TransactionContextInterface, idcarro string) error {
	// Verifique se a carteira já existe
	indexName := "WALLET"
	compositeKey, err := ctx.GetStub().CreateCompositeKey(indexName, []string{idcarro})
	if err != nil {
		return err
	}

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
		return nil, fmt.Errorf("falha ao ler a carteira do veículo: %s", err)
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

// DetectAnomalousAcceleration verifica a anomalia e atualiza a carteira do veículo de acordo
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

	log.Printf("Anomalia detectada: %v", anomalous)

	// Obtém o vehiclewallet atual
	indexName := "WALLET"
	compositeKey, err := ctx.GetStub().CreateCompositeKey(indexName, []string{idcarro})
	if err != nil {
		return "", err
	}

	vehicleWalletAsBytes, err := ctx.GetStub().GetState(compositeKey)
	if err != nil {
		return "", fmt.Errorf("falha ao ler a carteira do veículo: %s", err)
	}

	if vehicleWalletAsBytes == nil {
		return "", fmt.Errorf("carteira do veículo não encontrada")
	}

	// Atualiza vehiclewallet com o novo valor de créditos
	var vehicleWallet VehicleWallet
	err = json.Unmarshal(vehicleWalletAsBytes, &vehicleWallet)
	if err != nil {
		return "", fmt.Errorf("falha ao desserializar a carteira do veículo: %s", err)
	}

	currentCredits, err := strconv.Atoi(vehicleWallet.Credits)
	if err != nil {
		return "", fmt.Errorf("falha ao converter créditos atuais: %s", err)
	}
	vehicleWallet.Credits = strconv.Itoa(currentCredits + credits)

	vehicleWalletJSON, err := json.Marshal(vehicleWallet)
	if err != nil {
		return "", fmt.Errorf("falha ao serializar a carteira do veículo: %s", err)
	}

	return "", ctx.GetStub().PutState(compositeKey, vehicleWallet
