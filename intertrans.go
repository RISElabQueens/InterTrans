package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"

	"github.com/dgraph-io/badger/v4"
	"github.com/gosuri/uiprogress"
	"github.com/riselabqueens/intertrans/algo"
	"github.com/riselabqueens/intertrans/common"
	"github.com/riselabqueens/intertrans/executor"
	"google.golang.org/grpc"
)

type TranslationServer struct {
	common.UnimplementedTranslationServiceServer
}

type InfrastructureServer struct {
	common.UnimplementedInfrastructureServiceServer
}

const (
	maxMsgSize = 1000 * 1024 * 1024 * 2 // 2GB
)

func (m *TranslationServer) BatchTranslate(ctx context.Context, request *common.BatchTranslationRequest) (*common.BatchTranslationResponse, error) {
	return algo.InterTrans(request), nil
}

func (m *TranslationServer) BatchTranslateCAK(ctx context.Context, request *common.BatchTranslationRequest) (*common.BatchTranslationResponse, error) {
	return algo.DirectCAK(request), nil
}

// FIXME: This assumes that each intermediate edge is a single translation. This is not always the case.
func (m *TranslationServer) BatchRunVerification(ctx context.Context, request *common.BatchVerificationRequest) (*common.BatchVerificationResponse, error) {
	return algo.BatchRunVerification(request), nil
}

func (m *InfrastructureServer) LaunchInferenceEndpoint(ctx context.Context, request *common.StartEndpointRequest) (*common.LaunchResponse, error) {
	return executor.LaunchInstance(request)
}

func (m *InfrastructureServer) StopInferenceEndpoint(ctx context.Context, request *common.StopEndpointRequest) (*common.LaunchResponse, error) {
	return executor.StopInstance(request)
}

func main() {

	if len(os.Args) != 3 {
		fmt.Println("Usage: runserver <path_to_yaml_file>")
		return
	}

	command := os.Args[1]
	filePath := os.Args[2]

	if command != "runserver" {
		fmt.Println("Invalid command. Use 'runserver'.")
		return
	}

	err := common.LoadConfig(filePath)

	if err != nil {
		fmt.Println(err)
		return
	}

	num_execution_workers := common.ConfigStore.NumExecutionWorkers
	num_inference_workers := common.ConfigStore.NumInferenceWorkers

	uiprogress.Start()

	//FIXME: It works, but is commented out because the Print gets overriten by the progress bar so the messages never show up
	executor.InitializeBackpressureWatchdog()

	for i := 0; i < num_execution_workers; i++ {
		go executor.ExecutorWorker(i)
	}

	for i := 0; i < num_inference_workers; i++ {
		go executor.InferenceWorker(i)
	}

	lis, err := net.Listen("tcp", common.ConfigStore.ServerAddress+":"+common.ConfigStore.ServerPort)
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
	}
	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
	)

	translationServer := &TranslationServer{}
	infrastructureServer := &InfrastructureServer{}
	common.RegisterTranslationServiceServer(s, translationServer)
	common.RegisterInfrastructureServiceServer(s, infrastructureServer)

	if common.ConfigStore.ComputeEfficientMode {
		fmt.Println("Info: Using compute efficient mode.")
	}

	if common.ConfigStore.UseResponseCache {
		fmt.Println("Info: Using response cache database.")
	}

	if common.ConfigStore.UseInferenceCache {
		fmt.Println("Info: Using inference cache database.")
	}

	if common.ConfigStore.UseExecutionCache {
		fmt.Println("Info: Using execution cache database.")
	}

	if common.ConfigStore.UseTranscoderTestFormat {
		fmt.Println("Info: Using TransCoder Test Format for Unit Tests.")
	}

	db, err := badger.Open(badger.DefaultOptions(common.ConfigStore.DatabasePath))
	if err != nil {
		panic("Couldn't load cache database")
	}

	common.StoreDatabase(db)
	defer db.Close()

	numCPU := runtime.NumCPU()
	fmt.Printf("Info: Goroutines scheduled across %d CPUs\n", numCPU)

	fmt.Printf("🛤️🚀 InterTrans Engine Launched\n")
	fmt.Printf("-- Listening for requests at %v\n", lis.Addr())

	if err := s.Serve(lis); err != nil {
		fmt.Printf("Failed to serve: %v\n", err)
	}
}
