package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

// Função para calcular valores numéricos separados por espaço 
func calcularValores(valor string) (float64, error) {
	// Separar a string em partes, usando o espaço como delimitador
	valores := strings.Fields(valor)
	var soma float64
	for _, v := range valores {
		// Converter a string para float64
		num, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("erro ao converter valor para float: %v", err)
		}
		soma += num
	}
	return soma, nil
}

func main() {
	// Inicializa o SDK do Hyperledger Fabric
	sdk, err := fabsdk.New(config.FromFile("config.yaml"))
	if err != nil {
		fmt.Printf("Falha ao criar SDK: %v\n", err)
		return
	}
	defer sdk.Close()

	// Cria o cliente para o canal
	clientChannelContext := sdk.ChannelContext("mychannel", fabsdk.WithUser("User1"))
	client, err := channel.New(clientChannelContext)
	if err != nil {
		fmt.Printf("Falha ao criar cliente do canal: %v\n", err)
		return
	}

	for range time.Tick(10 * time.Minute) { // em 10m
		// Buscar IDs dos veículos com dados não processados
		request := channel.Request{
			ChaincodeID: "vehiclecc",
			Fcn:         "GetVehicleIDsWithPendingData",
			Args:        [][]byte{},
		}
		response, err := client.Execute(request)
		if err != nil {
			fmt.Printf("Erro ao buscar IDs de veículos: %v\n", err)
			continue
		}

		var vehicleIDs []string
		err = json.Unmarshal(response.Payload, &vehicleIDs)
		if err != nil {
			fmt.Printf("Erro ao desserializar IDs de veículos: %v\n", err)
			continue
		}

		// Processar cada veículo
		for _, vehicleID := range vehicleIDs {
			// Obter dados do veículo
			dataRequest := channel.Request{
				ChaincodeID: "vehiclecc",
				Fcn:         "GetVehicleData",
				Args:        [][]byte{[]byte(vehicleID)},
			}

			response, err := client.Execute(dataRequest)
			if err != nil {
				fmt.Printf("Erro ao obter dados para o veículo %s: %v\n", vehicleID, err)
				continue
			}

			// Supõe-se que os dados retornados são valores separados por espaço (como "10 20 30")
			valorData := string(response.Payload)

			// Calcular a soma dos valores numéricos separados por espaço
			soma, err := calcularValores(valorData)
			if err != nil {
				fmt.Printf("Erro ao calcular valores para o veículo %s: %v\n", vehicleID, err)
				continue
			}

			// Processa o veículo (com base na soma ou outros cálculos)
			request := channel.Request{
				ChaincodeID: "vehiclecc",
				Fcn:         "ProcessVehicleData",
				Args:        [][]byte{[]byte(vehicleID), []byte(fmt.Sprintf("%f", soma))}, // Enviar soma calculada como string
			}
			_, err = client.Execute(request)
			if err != nil {
				fmt.Printf("Erro ao processar veículo %s: %v\n", vehicleID, err)
			} else {
				fmt.Printf("Veículo %s processado com sucesso. Soma dos valores: %f\n", vehicleID, soma)
			}
		}
	}
}
