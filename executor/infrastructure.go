package executor

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/riselabqueens/intertrans/common"
)

var mutex sync.Mutex
var pidMap = make(map[int64]string)

func buildCommand(request *common.StartEndpointRequest) string {
	baseCommand := fmt.Sprintf("CUDA_VISIBLE_DEVICES=%s python -m vllm.entrypoints.openai.api_server --port %s --model %s --dtype auto --api-key %s",
		request.GpuId, request.Port, request.ModelName, request.ApiToken)

	if common.ConfigStore.Seed != -1 {
		baseCommand += fmt.Sprintf(" --seed %d", request.Seed)
	}

	if strings.Contains(request.ModelName, "Magicoder-S-DS-6.7B") {
		baseCommand += " --max-model-len 49024"
	}

	if request.LoraPath != "" {
		baseCommand += fmt.Sprintf(" --enable-lora --lora-modules %s-lora=%s", request.ModelName, request.LoraPath)
	}

	return baseCommand
}

func LaunchInstance(request *common.StartEndpointRequest) (*common.LaunchResponse, error) {
	command := buildCommand(request)

	fmt.Println(command)
	started := make(chan error, len(command))
	pid_chan := make(chan int64, 1)

	go func(command string) {
		cmd := exec.Command("bash", "-c", command)
		err := cmd.Start()

		started <- err

		// Retrieve the PID of the command
		pid := int64(cmd.Process.Pid)
		pid_chan <- pid

		// Store the PID in the map
		mutex.Lock()
		pidMap[pid] = command
		mutex.Unlock()

		// Wait for the command to finish and collect any error
		cmd.Wait()

	}(command)

	err := <-started

	if err != nil {
		return nil, err
	} else {
		pid := <-pid_chan
		response := &common.LaunchResponse{
			LaunchId: int64(pid),
		}
		return response, nil
	}

}

func StopInstance(request *common.StopEndpointRequest) (*common.LaunchResponse, error) {
	_, ok := pidMap[request.LaunchId]

	if ok {
		// Create a command to kill the process
		killCmd := exec.Command("kill", fmt.Sprintf("%d", request.LaunchId))
		// Run the kill command
		killCmd.Run()

		response := &common.LaunchResponse{
			LaunchId: request.LaunchId,
		}
		return response, nil
	} else {
		return nil, fmt.Errorf("PID not found")
	}

}
