package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/riselabqueens/intertrans/common"
	. "github.com/riselabqueens/intertrans/common"

	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/google/uuid"
)

type ExecutorQueueSingleton struct {
	InputChannel chan ExecutionUnit
	once         sync.Once
}

type InferenceQueueSingleton struct {
	InputChannel chan InferenceUnit
	once         sync.Once
}

type LimitedBuffer struct {
	buf   bytes.Buffer
	limit int
}

func (lb *LimitedBuffer) Write(p []byte) (n int, err error) {
	if lb.buf.Len()+len(p) > lb.limit {
		return 0, errors.New("buffer size limit exceeded")
	}
	return lb.buf.Write(p)
}

func (lb *LimitedBuffer) Bytes() []byte {
	return lb.buf.Bytes()
}

func (lb *LimitedBuffer) String() string {
	return lb.buf.String()
}

func (lb *LimitedBuffer) Reset() {
	lb.buf.Reset()
}

type BackpressureWatchdog struct {
	inferenceCounter int64
	executionCounter int64
	mutex            sync.Mutex
}

var globalWatchdog *BackpressureWatchdog

func InitializeBackpressureWatchdog() {

	globalWatchdog = &BackpressureWatchdog{
		inferenceCounter: 0,
		executionCounter: 0,
		mutex:            sync.Mutex{},
	}

	go func() {

		for {

			globalWatchdog.mutex.Lock()

			if globalWatchdog.inferenceCounter != 0 && globalWatchdog.executionCounter != 0 {

				if globalWatchdog.inferenceCounter > globalWatchdog.executionCounter {
					rate := (globalWatchdog.inferenceCounter - globalWatchdog.executionCounter) / globalWatchdog.inferenceCounter * 100

					if rate > 20 {
						fmt.Fprintln(os.Stderr, "Warning: Backpressure detected. In the last 30 seconds, inference was %d%% faster than execution.", rate)
					}

				} else {
					rate := (globalWatchdog.executionCounter - globalWatchdog.inferenceCounter) / globalWatchdog.executionCounter * 100

					if rate > 20 {
						fmt.Fprintln(os.Stderr, "Warning: Backpressure detected. In the last 30 seconds, execution was %d%% faster than inference.", rate)
					}
				}

				globalWatchdog.executionCounter = 0
				globalWatchdog.inferenceCounter = 0

			}

			globalWatchdog.mutex.Unlock()
			time.Sleep(30 * time.Second)
		}
	}()
}

func (watchdog *BackpressureWatchdog) CountInference() {
	globalWatchdog.mutex.Lock()
	globalWatchdog.inferenceCounter++
	globalWatchdog.mutex.Unlock()
}

func (watchdog *BackpressureWatchdog) CountExecution() {
	globalWatchdog.mutex.Lock()
	globalWatchdog.executionCounter++
	globalWatchdog.mutex.Unlock()
}

var executorInstance *ExecutorQueueSingleton
var inferenceInstance *InferenceQueueSingleton
var inferenceInstanceLock sync.Mutex
var executorInstanceLock sync.Mutex

func GetExecutorQueueInstance() *ExecutorQueueSingleton {
	executorInstanceLock.Lock()
	defer executorInstanceLock.Unlock()

	if executorInstance == nil {
		executorInstance = &ExecutorQueueSingleton{}
		executorInstance.InputChannel = make(chan ExecutionUnit, common.ConfigStore.NumExecutionWorkers)
	} else {

	}

	return executorInstance
}

func GetInferenceQueueInstance() *InferenceQueueSingleton {
	inferenceInstanceLock.Lock()
	defer inferenceInstanceLock.Unlock()

	if inferenceInstance == nil {
		inferenceInstance = &InferenceQueueSingleton{}
		inferenceInstance.InputChannel = make(chan InferenceUnit, common.ConfigStore.NumInferenceWorkers)
	} else {

	}

	return inferenceInstance
}

func imageExists(cli *client.Client, ctx context.Context, imageName string) (bool, error) {
	images, err := cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return false, err
	}

	for _, image := range images {
		for _, tag := range image.RepoTags {
			if tag == imageName {
				return true, nil
			}
		}
	}
	return false, nil
}

// Worker that executes generated code
func InferenceWorker(id int) {
	//message := fmt.Sprintf("Started Inference Worker %d", id)
	//
	channel := GetInferenceQueueInstance().InputChannel

	for {
		unit := <-channel

		ExecuteInference(unit)
	}

}

// Worker that runs inference
func ExecutorWorker(id int) {
	channel := GetExecutorQueueInstance().InputChannel

	for {
		unit := <-channel

		ExecuteCode(unit)
	}
}

func isJavascriptES5(src string) bool {
	return strings.Contains(src, "import")
}

func WriteCodeToFilesystem(sourceCode string, language string) (string, string, string, bool) {
	fileExtensionsMap := GetFileExtensionsMap()
	var extension string
	var exists bool

	if language == "JavaScript" && isJavascriptES5(sourceCode) {
		extension = ".mjs"
		exists = true
	} else {
		extension, exists = fileExtensionsMap[language]
	}

	if !exists {
		panic(fmt.Sprintf("File extension for %s not found\n", language))
	}

	fileUUID := uuid.New().String()
	fileName := fileUUID + extension
	tempDir := os.TempDir()
	dirPath := filepath.Join(tempDir, "POSSIBLY_DANGEROUS", language)
	filePath := filepath.Join(dirPath, fileName)

	err := os.MkdirAll(filepath.Dir(dirPath), 0755)
	if err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		return "", "", "", false
	}

	err = os.WriteFile(filePath, []byte(sourceCode), 0777)

	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return "", "", "", false
	}

	return dirPath, filePath, fileName, true

}

func getLastPart(input string, sep string) string {
	lastIndex := strings.LastIndex(input, sep)
	if lastIndex == -1 {
		return input
	}
	return input[lastIndex+len(sep):]
}

func sanitizeUnusedPackages(src string) string {
	// Define the regular expression to find the import block
	re := regexp.MustCompile(`import\s*(?:"([^"]*)"|\(([^)]*)\))`)

	// Find the import block in the source code
	matches := re.FindStringSubmatch(src)
	if len(matches) == 0 {
		return src
	}

	// Extract the import block content
	importBlock := matches[0]
	innerImports := matches[2]

	// Split import block by newlines and trim spaces
	lines := strings.Split(innerImports, "\n")
	codeWithoutImport := strings.ReplaceAll(src, importBlock, "")
	newLines := []string{}

	for i := range lines {

		//If some import is not used in the code, remove it from the import list to prevent error
		trimmed := strings.TrimSpace(lines[i])
		noQuotes := strings.ReplaceAll(trimmed, `"`, "")

		usage := getLastPart(noQuotes, "/")

		if noQuotes != "" && (strings.Contains(codeWithoutImport, usage) || strings.Contains(codeWithoutImport, noQuotes)) {
			newLines = append(newLines, trimmed)
		}

	}

	// Check if "testing" is already imported
	testingImported := strings.Contains(importBlock, `"testing"`)
	// Check if "github.com/stretchr/testify/assert" is already imported
	assertImported := strings.Contains(importBlock, `"github.com/stretchr/testify/assert"`)
	inflectImported := strings.Contains(importBlock, `"github.com/go-openapi/inflect"`)

	// Prepare the updated import block
	var newImports []string

	if !testingImported && strings.Contains(src, "testing.") {
		newImports = append(newImports, "\"testing\"")
	}
	if !assertImported && strings.Contains(src, "assert.") {
		newImports = append(newImports, "\"github.com/stretchr/testify/assert\"")
	}
	if !inflectImported && strings.Contains(src, "inflect.") {
		newImports = append(newImports, "\"github.com/go-openapi/inflect\"")
	}

	// Construct the new import block with injected packages
	newImportBlock := fmt.Sprintf("import (\n%s\n)", strings.Join(newLines, "\n\t")+"\n\t"+strings.Join(newImports, "\n\t"))

	// Replace the old import block with the new one
	return strings.Replace(src, importBlock, newImportBlock, 1)
}

// this is only for HumanEvalX
func injectTestingAndAssertPackages(src string) string {

	//Clean packages
	if !strings.Contains(src, "package") {
		src = "package common\n\n" + src
	}

	//The test case file should not be executable
	if strings.Contains(src, "package main") {
		src = strings.Replace(src, "package main", "package common", 1)
	}

	//Remove any main function
	src = RemoveGolangMain(src)
	finalResult := sanitizeUnusedPackages(src)

	return finalResult
}

func injectCodenetJavaScriptReadline(str string) string {
	returnStr := ""

	if !isJavascriptES5(str) {
		//Minimum import for CodeNet fuzzy cases to receive from stdin
		if strings.Contains(str, "readline") {
			if !strings.Contains(str, "readline = require") && !strings.Contains(str, "{ readline } = require") {
				returnStr += "const readline = require('readline');\n"
			}
		}

		//An alternative
		if strings.Contains(str, "prompt") {
			if !strings.Contains(str, "prompt = require") && !strings.Contains(str, "{ prompt } = require") {
				returnStr += "const prompt = require('prompt-sync');\n"
			}
		}

		if strings.Contains(str, "fs") {
			if !strings.Contains(str, "fs = require") && !strings.Contains(str, "{ fs } = require") {
				returnStr += "const fs = require('fs');\n"
			}
		}

		returnStr += str
	} else {
		//Minimum import for CodeNet fuzzy cases to receive from stdin
		if strings.Contains(str, "readline") {
			if !strings.Contains(str, "import readline") && !strings.Contains(str, "import * as readline") && !strings.Contains(str, "import { readline }") {
				returnStr += "import readline from 'readline';\n"
			}
		}

		//An alternative
		if strings.Contains(str, "prompt") {
			if !strings.Contains(str, "import prompt") && !strings.Contains(str, "import * as prompt") && !strings.Contains(str, "import { prompt }") {
				returnStr += "import prompt from 'prompt-sync';\n"
			}
		}

		if strings.Contains(str, "fs") {
			if !strings.Contains(str, "import fs") && !strings.Contains(str, "import * as fs") && !strings.Contains(str, "import { fs }") {
				returnStr += "import fs from 'fs';\n"
			}
		}

		returnStr += str
	}

	return returnStr
}

// We need to standarize the code for some programming languages
func StandarizeCode(executionUnit ExecutionUnit) string {
	var result string

	//Java complains about public class. Also we change the class name to match the executor.
	if executionUnit.Language == "Java" {
		pattern := `public\s+class\s+\w+`
		// Replacement string
		replacement := "class A"
		// Compile the regex pattern
		re := regexp.MustCompile(pattern)
		// Perform the replacement
		result = re.ReplaceAllString(executionUnit.SourceCode, replacement)
	} else if executionUnit.Language == "Go" && executionUnit.ExecutionType == TEST {
		result = injectTestingAndAssertPackages(executionUnit.SourceCode)
	} else if executionUnit.Language == "Go" && executionUnit.ExecutionType != TEST {
		result = sanitizeUnusedPackages(executionUnit.SourceCode)
	} else if executionUnit.Language == "JavaScript" && executionUnit.ExecutionType != TEST {
		result = injectCodenetJavaScriptReadline(executionUnit.SourceCode)
	} else {
		result = executionUnit.SourceCode
	}

	return result
}

func ExecuteCode(executionUnit ExecutionUnit) {

	imageExecutorMap := GetExecutorForLanguageMap()
	executorImage, imageExists := imageExecutorMap[executionUnit.Language]

	if !imageExists {
		panic(fmt.Sprintf("Executor image for %s not found\n", executionUnit.Language))
	}

	standardSourceCode := StandarizeCode(executionUnit)

	executionUnit.ExecutedCode = standardSourceCode

	if common.ConfigStore.UseExecutionCache {
		response, err := LoadExistingExecutionResults(&executionUnit)

		if !err {
			executionUnit.UsedExecutionCache = true
			executionUnit.OutputChannel <- response
			return
		}
	}

	dirPath, filePath, fileName, ok := WriteCodeToFilesystem(standardSourceCode, executionUnit.Language)

	if !ok {
		panic("Could not write to filesystem")
	}

	inContainerPath := "/code/" + fileName

	// Bind the command to the context
	var cmd *exec.Cmd

	// Create a context with a timeout of 90 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	if executionUnit.ExecutionType == RUN {
		cmd = exec.CommandContext(ctx, "singularity", "exec", "--memory", "4G", "--writable-tmpfs", "--no-privs", "--network", "none", "--cpus", "4", "--no-home", "--containall", "--bind", dirPath+":/code:ro", executorImage, "/bin/script", inContainerPath)
	} else {
		cmd = exec.CommandContext(ctx, "singularity", "exec", "--memory", "4G", "--writable-tmpfs", "--no-privs", "--network", "none", "--cpus", "4", "--no-home", "--containall", "--bind", dirPath+":/code:ro", executorImage, "/bin/script", inContainerPath, "test")
	}

	// Maximum allowed output of a program to prevent memory exhaustation
	const bufferSizeLimit = 1024 * 1024 // 1 MB
	stdoutOutput := LimitedBuffer{limit: bufferSizeLimit}
	stderrOutput := LimitedBuffer{limit: bufferSizeLimit}

	cmd.Stdout = &stdoutOutput
	cmd.Stderr = &stderrOutput

	// Get the process ID (PID)
	var pid int

	startTime := time.Now()

	// Create a pipe to connect to the command's standard input
	if executionUnit.StdinData != "" {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			fmt.Printf("Error creating stdin pipe: %v\n", err)
			panic("Error creating stdin pipe")
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			fmt.Printf("Error starting command: %v\n", err)
			return
		}

		// Get the process ID (PID)
		pid = cmd.Process.Pid

		//The program may print a prompt at startup which messes up evaluation. Clear it before continuing
		//FIXME. It may not good to sync sleep this, performance wise
		time.Sleep(time.Second * 3)
		stdoutOutput.Reset()

		// Write the input data to the stdin pipe
		_, err = stdin.Write([]byte(executionUnit.StdinData))
		if err != nil {
			fmt.Printf("Error writing to stdin: %v\n", err)
		}

		if err := stdin.Close(); err != nil {
			fmt.Printf("Error closing stdin: %v\n", err)
			return
		}
	} else {
		// Start the command
		if err := cmd.Start(); err != nil {
			fmt.Printf("Error starting command: %v\n", err)
			return
		}

		// Get the process ID (PID)
		pid = cmd.Process.Pid
	}

	// Some containers hang forever, we need to stop them
	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			stopCmd := exec.Command("kill", "-9", fmt.Sprintf("%d", pid))

			stopCmd.Start()
			stopErr := stopCmd.Wait()

			if stopErr != nil {
				fmt.Printf("Couldn't kill container with PID: %s\n", fmt.Sprintf("%d", pid))
			}

		}
	}()

	err := cmd.Wait()

	endTime := time.Since(startTime)

	var combinedOutput string

	if ctx.Err() == context.DeadlineExceeded {
		combinedOutput = "CMD_TIMEOUT_KILLED"
	} else if err != nil {
		var exitCode int
		if exitError, ok := err.(*exec.ExitError); ok {
			// The command has exited with a non-zero exit code
			exitCode = exitError.ExitCode()
		} else {
			// Some other error occurred
			exitCode = -1
		}
		combinedOutput = fmt.Sprintf("(Exit code: %d) %s", exitCode, stderrOutput.String())
		executionUnit.Success = false
	} else {
		combinedOutput = stdoutOutput.String()
		executionUnit.Success = true
	}

	if !utf8.ValidString(combinedOutput) {
		combinedOutput = "FAIL_INVALID_UTF8_STRING"
	}

	executionUnit.ExecutionOutput = combinedOutput
	executionUnit.WallTime = endTime

	err = os.Remove(filePath)

	if err != nil {
		panic("Something went wrong removing a file. This shouldn't happen.")
	}

	globalWatchdog.CountExecution()

	//This may be a transient error, so instead we ignore in that case
	SaveExecutionToCache(&executionUnit)

	executionUnit.OutputChannel <- executionUnit
}

type MyRoundTripper struct {
	r http.RoundTripper
}

func (mrt MyRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Add("Authorization", "Bearer: token")
	return mrt.r.RoundTrip(r)
}

func ExecuteInference(inferenceUnit InferenceUnit) {

	if common.ConfigStore.UseInferenceCache {
		cacheResponse, err := LoadInferenceExistingResponse(inferenceUnit.Prompt, inferenceUnit.ModelName)

		if !err {
			cacheResponse.IsCached = true
			inferenceUnit.OutputChannel <- cacheResponse
			return
		}
	}

	retryError := true
	retryCount := 0

	var finalResponse string
	var startInference time.Time

	for retryError {
		startInference = time.Now()

		apiKey := common.ConfigStore.InferenceApiToken

		if apiKey == "" {
			panic("API token is empty")
		}

		response, err := GetChatCompletion(apiKey, inferenceUnit.Prompt, inferenceUnit.ModelName)
		finalResponse = response

		fmt.Println(response)

		if err != nil {
			fmt.Println(err)
			if retryCount < 6 {
				retryError = true
				retryCount++
				// This is not efficient we should remove from queue and let other task try
				time.Sleep(10 * time.Second)
			} else {
				retryError = false
				finalResponse = "INFERENCE_ERROR_RETRIED"
				break
			}

		} else {
			retryError = false
			break
		}

	}

	globalWatchdog.CountInference()
	endTime := time.Since(startInference)

	InferenceResult := InferenceResult{
		Response: finalResponse,
		IsCached: false,
		WallTime: endTime,
		Success:  (finalResponse != "INFERENCE_ERROR_RETRIED"),
	}

	SaveInferenceResponseToCache(inferenceUnit.Prompt, inferenceUnit.ModelName, InferenceResult)
	inferenceUnit.OutputChannel <- InferenceResult
}
