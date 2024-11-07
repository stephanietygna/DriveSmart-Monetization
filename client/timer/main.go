package main

import (
	"fmt"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
)

// Processa dados de veículos não processados a cada 10 minutos.
func main() {
	sdk, err := fabsdk.New(config.FromFile("config.yaml"))
	if err != nil {
		fmt.Printf("Falha ao criar SDK: %v\n", err)
		return
	}
	defer sdk.Close()

	clientChannelContext := sdk.ChannelContext("mychannel", fabsdk.WithUser("User1"))
	client, err := channel.New(clientChannelContext)
	if err != nil {
		fmt.Printf("Falha ao criar cliente do canal: %v\n", err)
		return
	}

	for range time.Tick(10 * time.Minute) {
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
			request := channel.Request{
				ChaincodeID: "vehiclecc",
				Fcn:         "ProcessVehicleData",
				Args:        [][]byte{[]byte(vehicleID)},
			}
			_, err := client.Execute(request)
			if err != nil {
				fmt.Printf("Erro ao processar veículo %s: %v\n", vehicleID, err)
			} else {
				fmt.Printf("Veículo %s processado com sucesso\n", vehicleID)
			}
		}
	}
}
