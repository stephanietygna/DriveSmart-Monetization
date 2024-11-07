package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
)

const (
	channelName           = "mychannel"       // Nome do canal do Hyperledger Fabric.
	contractName          = "vehicleContract" // Nome do contrato.
	queryInterval         = 10 * time.Minute  // Intervalo de execução a cada 10 minutos.
	zigzagThreshold       = 5                 // Número de mudanças de faixa considerado zigue-zague.
	accelerationThreshold = 30.0              // Threshold para aceleração anômala (variação de velocidade > 30 km/h).
	steeringThreshold     = 0.7               // Threshold para mudanças bruscas de direção (rad > 0,7).
)

type VehicleData struct {
	ID            string  `json:"id"`
	Speed         float64 `json:"speed"`
	Direction     float64 `json:"direction"`
	LaneChanges   int     `json:"laneChanges"`
	Processed     bool    `json:"processed"`
}

func main() {
	// Configuração do gateway e do contrato
	wallet, err := gateway.NewFileSystemWallet("wallet")
	if err != nil {
		log.Fatalf("Falha ao criar carteira: %v", err)
	}

	ccpPath := "/path/to/connection.yaml" // Defina o caminho para seu arquivo de configuração
	gw, err := gateway.Connect(
		gateway.WithConfig(gateway.FromFile(ccpPath)),
		gateway.WithIdentity(wallet, "appUser"),
	)
	if err != nil {
		log.Fatalf("Falha ao conectar ao gateway: %v", err)
	}
	defer gw.Close()

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(contractName)

	// Configuração do ticker para execução periódica
	ticker := time.NewTicker(queryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			processPendingVehicleData(contract)
		}
	}
}

func processPendingVehicleData(contract *gateway.Contract) {
	// Buscar lista de IDs de veículos com dados pendentes
	vehicleIDsResult, err := contract.EvaluateTransaction("GetVehicleIDsWithPendingData")
	if err != nil {
		log.Printf("Erro ao buscar IDs de veículos com dados pendentes: %v", err)
		return
	}

	var vehicleIDs []string
	err = json.Unmarshal(vehicleIDsResult, &vehicleIDs)
	if err != nil {
		log.Printf("Erro ao parsear lista de IDs: %v", err)
		return
	}

	for _, vehicleID := range vehicleIDs {
		processDataForVehicle(contract, vehicleID)
	}
}

func processDataForVehicle(contract *gateway.Contract, vehicleID string) {
	// Buscar dados não processados específicos do veículo
	result, err := contract.EvaluateTransaction("GetUnprocessedDataByVehicleID", vehicleID)
	if err != nil {
		log.Printf("Erro ao buscar dados para veículo %s: %v", vehicleID, err)
		return
	}

	var vehicleDataList []VehicleData
	err = json.Unmarshal(result, &vehicleDataList)
	if err != nil {
		log.Printf("Erro ao parsear dados do veículo %s: %v", vehicleID, err)
		return
	}

	// Avaliar cada conjunto de dados não processados para o veículo
	for _, data := range vehicleDataList {
		// Avaliação das regras
		anomalousAcceleration := checkAcceleration(data)
		sharpTurns := checkSteering(data)
		zigzagBehavior := checkZigzag(data)

		// Salvar resultados de análise para cada conjunto de dados processado
		err = saveAnalysisResult(contract, data.ID, anomalousAcceleration, sharpTurns, zigzagBehavior)
		if err != nil {
			log.Printf("Erro ao salvar análise para dados do veículo %s: %v", vehicleID, err)
		}
	}

	// Marcar os dados do veículo como processados
	err = markDataAsProcessed(contract, vehicleID)
	if err != nil {
		log.Printf("Erro ao marcar dados do veículo %s como processados: %v", vehicleID, err)
	}
}

func checkAcceleration(vehicleData VehicleData) bool {
	// Verifica se a variação de velocidade excede o limite de aceleração anômala
	return vehicleData.Speed > accelerationThreshold
}

func checkSteering(vehicleData VehicleData) bool {
	// Verifica se a direção muda bruscamente
	return math.Abs(vehicleData.Direction) > steeringThreshold
}

func checkZigzag(vehicleData VehicleData) bool {
	// Verifica se o número de mudanças de faixa excede o limite de zigue-zague
	return vehicleData.LaneChanges > zigzagThreshold
}

func saveAnalysisResult(contract *gateway.Contract, vehicleID string, anomalousAcceleration, sharpTurns, zigzagBehavior bool) error {
	result := map[string]interface{}{
		"ID":                  vehicleID,
		"AnomalousAcceleration": anomalousAcceleration,
		"SharpTurns":            sharpTurns,
		"ZigzagBehavior":        zigzagBehavior,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("falha ao serializar resultado: %v", err)
	}

	_, err = contract.SubmitTransaction("SaveAnalysisResult", string(data))
	if err != nil {
		return fmt.Errorf("falha ao salvar resultado no blockchain: %v", err)
	}

	log.Printf("Resultados processados e salvos para veículo %s", vehicleID)
	return nil
}

func markDataAsProcessed(contract *gateway.Contract, vehicleID string) error {
	// Marcar todos os dados como processados para o veículo específico
	_, err := contract.SubmitTransaction("MarkDataAsProcessed", vehicleID)
	if err != nil {
		return fmt.Errorf("falha ao marcar dados do veículo %s como processados: %v", vehicleID, err)
	}
	log.Printf("Dados marcados como processados para o veículo %s", vehicleID)
	return nil
}

