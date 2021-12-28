package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/vippsas/go-cosmosdb/cosmosapi"
)

const NumberOfElevators = 4
const NumberOfFloors = 6
const NumberOfElevatorDoors = NumberOfElevators * NumberOfFloors

type ElevatorDirection int16
type ElevatorStatus int16

const (
	UP   ElevatorDirection = 1
	DOWN                   = 2
	NONE                   = 3
)
const (
	MOVING       ElevatorStatus = 1
	ATFLOOR                     = 2
	DOORSOPENING                = 3
	DOORSCLOSING                = 4
)

type ElevatorState struct {
	ElevatorNumber int               `json:"elevatorNumber"`
	Direction      ElevatorDirection `json:"direction"`
	ElevatorStatus ElevatorStatus    `json:"elevatorStatus"`
	FromFloor      int               `json:"fromFloor"`
	ToFloor        int               `json:"toFloor"`
	Progress       int               `json:"progress"`
	AtFloor        int               `json:"atFloor"`
	Id             string            `json:"id"`
}

type ElevatorDoorState struct {
	ElevatorShaftNumber int               `json:"elevatorShaftNumber"`
	Floor               int               `json:"floor"`
	Open                bool              `json:"open"`
	ElevatorAtFloor     int               `json:"elevatorAtFloor"`
	ElevatorDirection   ElevatorDirection `json:"elevatorDirection"`
}

type State struct {
	ElevatorState     []ElevatorState     `json:"elevatorState"`
	ElevatorDoorState []ElevatorDoorState `json:"elevatorDoorState"`
}

type config struct {
	DbUrl  string
	DbKey  string
	DbName string
}

func elevatorStateToDoorIsOpen(elevatorState ElevatorState, currentFloor int) bool {
	if elevatorState.ElevatorStatus == DOORSOPENING && currentFloor == elevatorState.AtFloor {
		return true
	}
	return false
}

func stateHandler(w http.ResponseWriter, r *http.Request) {
	elevatorStates := getElevatorsFromDB()

	for i := range elevatorStates {
		elevatorStates[i].ElevatorNumber, _ = strconv.Atoi(elevatorStates[i].Id)
	}
	var state State
	state.ElevatorState = elevatorStates
	elevatorDoorStates := make([]ElevatorDoorState, NumberOfElevatorDoors)
	total := 0
	for elevator := range [NumberOfElevators]int{} {
		for floor := range [NumberOfFloors]int{} {
			elevatorId := elevatorStates[elevator].ElevatorNumber
			// fmt.Printf("elevatorid: %v floori: %v\n", elevatorId, floor)
			elevatorDoorStates[total] = ElevatorDoorState{
				ElevatorShaftNumber: elevatorId,
				Floor:               floor + 1,
				Open:                elevatorStateToDoorIsOpen(elevatorStates[elevator], floor+1),
				ElevatorAtFloor:     elevatorStates[elevator].AtFloor,
				ElevatorDirection:   ElevatorDirection(elevatorStates[elevator].ElevatorStatus),
			}
			total++
		}
	}
	state.ElevatorDoorState = elevatorDoorStates
	fmt.Printf("State: %+v\n", state)
	bytes, err := json.Marshal(state)
	if err != nil {
		err = errors.WithStack(err)
		fmt.Println(err)
	}
	output := string(bytes)
	fmt.Fprint(w, output)
}

func getElevatorsFromDB() []ElevatorState {
	DbUrl, _ := os.LookupEnv("DbUrl")
	DbKey, _ := os.LookupEnv("DbKey")
	DbName, _ := os.LookupEnv("DbName")
	cosmosCfg := cosmosapi.Config{
		MasterKey: DbKey,
	}
	qops := cosmosapi.DefaultQueryDocumentOptions()
	qops.EnableCrossPartition = true

	client := cosmosapi.New(DbUrl, cosmosCfg, nil, nil)
	qry := cosmosapi.Query{
		Query: "SELECT * FROM c WHERE c.id <= '4'",
	}
	// fmt.Printf("Query : %v\n", qry)
	var estates []ElevatorState
	_, err := client.QueryDocuments(context.Background(), DbName, "elevator", qry, &estates, qops)
	if err != nil {
		err = errors.WithStack(err)
		fmt.Println(err)
	}
	return estates
}

func main() {
	listenAddr := ":8080"
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		listenAddr = ":" + val
	}
	http.HandleFunc("/", stateHandler)
	//mux := http.NewServeMux()
	// mux.HandleFunc("/", stateHandler)
	log.Printf("About to listen on %s. Go to https://127.0.0.1%s/", listenAddr, listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
