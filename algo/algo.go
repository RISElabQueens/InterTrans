package algo

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gosuri/uiprogress"
	"github.com/RISElabQueens/intertrans/common"
	. "github.com/RISElabQueens/intertrans/common"
	. "github.com/RISElabQueens/intertrans/executor"
	"golang.org/x/sync/semaphore"
)

type TraversalUnit struct {
	TranslationEdge         TranslationEdge
	ExecutorQueueSingleton  ExecutorQueueSingleton
	InferenceQueueSingleton InferenceQueueSingleton
	IsDone                  bool
}

func GetTotalEdgesCountIntermediates(batchRequest *BatchTranslationRequest) int {
	total := 0
	var totalEdgesForRequest int

	//TODO: This is hardcoded
	for _, request := range batchRequest.TranslationRequests {
		if ConfigStore.ExpansionDepth == 1 {
			totalEdgesForRequest = 1
		} else if ConfigStore.ExpansionDepth == 3 {
			totalEdgesForRequest = 21
		} else if ConfigStore.ExpansionDepth == 4 {
			totalEdgesForRequest = 85
		} else {
			panic("Count for this depth not implemented")
		}

		// lenLanguagesUsed := len(request.UsedLanguages)

		// for l := 1; l < common.ConfigStore.ExpansionDepth; l++ {
		// 	floatLevel := float64(l - 1)
		// 	langsAdj := float64(lenLanguagesUsed - 2)
		// 	totalEdgesForRequest = totalEdgesForRequest + int(math.Pow(langsAdj, floatLevel))*l
		// }

		_ = request
		total = total + totalEdgesForRequest
	}

	return total
}

func ProccessVerificationRequest(request *VerificationRequest, wg *sync.WaitGroup, results chan *VerificationResponse, bar *uiprogress.Bar) {
	defer wg.Done()
	translationEdge := &TranslationEdge{}

	if request.TestSuite == nil || len(request.TestSuite.UnitTestSuite) == 0 && len(request.TestSuite.FuzzySuite) == 0 {
		panic("Please provide unit tests")
	}

	//Fuzzy tests are language independent so we can evaluate all of them
	for _, test := range request.TestSuite.FuzzySuite {

		fuzzyTest := FuzzyTest{
			Input:          test.StdinInput,
			ExpectedOutput: test.ExpectedOutput,
		}

		translationEdge.FuzzyTests = append(translationEdge.FuzzyTests, fuzzyTest)
	}

	for _, unitTest := range request.TestSuite.UnitTestSuite {

		unitTest := UnitTest{
			SourceCode: unitTest.TestCase,
			Imports:    unitTest.Imports,
		}

		translationEdge.UnitTests = append(translationEdge.UnitTests, unitTest)
	}

	if len(request.TestSuite.UnitTestSuite) > 0 && len(request.TestSuite.FuzzySuite) > 0 {
		panic("We don't yet support evaluation with mixed fuzzy and unit tests. Feel free to contribute to this :=)")
	}

	translationEdge.InferenceOutput = request.InferenceOutput
	translationEdge.TargetLanguage = request.TargetLanguage
	translationEdge.InputLanguage = request.SourceLanguage
	translationEdge.ProcessingMutex = &sync.Mutex{}
	translationEdge.StatusMutex = &sync.Mutex{}
	translationEdge.SetStatus(PENDING)

	//Extract the source code
	if !common.ConfigStore.ApplyRegexInferenceOnly {
		panic("We only support extracting from inference output at this time. Please set ApplyRegexInferenceOnly to true")
	}

	//FIXME: This is hardcoded
	translationEdge.RegexTemplate = GetRegexTemplate("temperature")
	extracted, extractedOk := ExtractSourceCode("", translationEdge.RegexTemplate, request.InferenceOutput)

	//Can't process downstream edges as we weren't able to extract the code
	if !extractedOk {
		translationEdge.SetStatus(FAILED_NO_EXTRACTED)
	} else {
		translationEdge.ExtractedSourceCode = extracted
		PerformEdgeExecution(translationEdge, translationEdge.TargetLanguage)
	}

	responseFuzzyTests := []*ResponseFuzzyTestCase{}
	responseUnitTests := []*ResponseUnitTestCase{}

	for _, test := range translationEdge.FuzzyTests {
		responseFuzzyTests = append(responseFuzzyTests, test.ToResponse())
	}

	for _, test := range translationEdge.UnitTests {
		responseUnitTests = append(responseUnitTests, test.ToResponse())
	}

	verificationResponse := &VerificationResponse{
		FuzzyTests:          responseFuzzyTests,
		UnitTests:           responseUnitTests,
		VerificationRequest: request,
		Status:              translationEdge.GetStatus().String(),
	}

	results <- verificationResponse
	bar.Incr()
}

func BatchRunVerification(batchRequest *BatchVerificationRequest) *BatchVerificationResponse {

	bar := uiprogress.AddBar(len(batchRequest.VerificationRequests)).AppendCompleted().AppendElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("%s (%d/%d)", batchRequest.Id, b.Current(), len(batchRequest.VerificationRequests))
	})

	wg := sync.WaitGroup{}
	chanResponses := make(chan *VerificationResponse, len(batchRequest.VerificationRequests))

	for _, request := range batchRequest.VerificationRequests {
		wg.Add(1)
		go ProccessVerificationRequest(request, &wg, chanResponses, bar)
	}

	wg.Wait()
	close(chanResponses)

	allResponses := []*VerificationResponse{}

	for verificationResponse := range chanResponses {
		allResponses = append(allResponses, verificationResponse)
	}

	return &BatchVerificationResponse{
		VerificationResponses: allResponses,
	}

}

func DirectCAK(batchRequest *BatchTranslationRequest) *BatchTranslationResponse {
	var wtg sync.WaitGroup

	//We assign an id to the request
	//TODO: Should be done on the client side and returned immediately for the client to see
	uuidObj := uuid.New()
	shortUUID := uuidObj.String()[:6]
	batchRequest.Id = shortUUID

	//Keep references to preserve the order when we return the responses
	responseChannel := make(chan *TranslationResponse, len(batchRequest.TranslationRequests))

	//Initialize progress bar
	totalEdges := len(batchRequest.TranslationRequests) * 10

	bar := uiprogress.AddBar(totalEdges).AppendCompleted().AppendElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("%s (%d/%d)", shortUUID, b.Current(), totalEdges)
	})

	semCtx := context.TODO()

	//TODO: This batch size is hardcoded
	var (
		maxBatch = 250
		sem      = semaphore.NewWeighted(int64(maxBatch))
	)

	for _, request := range batchRequest.TranslationRequests {
		wtg.Add(1)

		if err := sem.Acquire(semCtx, 1); err != nil {
			panic("Failed to acquire semaphore")
		}

		go RequestTranslationDirectorCAK(request, &wtg, bar, responseChannel, sem)

	}

	wtg.Wait()

	//We are not expecting more values
	close(responseChannel)

	allResponses := []*TranslationResponse{}

	for response := range responseChannel {
		allResponses = append(allResponses, response)
	}

	//Return sorted by translation id
	sort.SliceStable(allResponses, func(i, j int) bool {
		return allResponses[i].TranslationRequest.Id < allResponses[j].TranslationRequest.Id
	})

	response := &BatchTranslationResponse{
		RequestId:            shortUUID,
		TranslationResponses: allResponses,
	}

	if batchRequest.FileBaseName != "" && batchRequest.FileSavePath != "" {
		//We should save to the disk instead
		SaveBatchResponseToFile(batchRequest.FileBaseName, batchRequest.FileSavePath, response)
		return &BatchTranslationResponse{
			RequestId:            shortUUID,
			TranslationResponses: []*TranslationResponse{},
			ReturnedToDisk:       true,
		}
	}
	return response
}

func InterTrans(batchRequest *BatchTranslationRequest) *BatchTranslationResponse {
	var wtg sync.WaitGroup

	//We assign an id to the request
	//FIXME: Should be done on the client side and returned immediately for the client to see
	uuidObj := uuid.New()
	shortUUID := uuidObj.String()[:6]
	batchRequest.Id = shortUUID

	//Keep references to preserve the order when we return the responses
	responseChannel := make(chan *TranslationResponse, len(batchRequest.TranslationRequests))

	//Initialize progress bar
	totalEdges := GetTotalEdgesCountIntermediates(batchRequest)

	bar := uiprogress.AddBar(totalEdges).AppendCompleted().AppendElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("%s (%d/%d)", shortUUID, b.Current(), totalEdges)
	})

	semCtx := context.TODO()

	//FIXME: This batch size is hardcoded
	var (
		maxBatch = 250
		sem      = semaphore.NewWeighted(int64(maxBatch))
	)

	for _, request := range batchRequest.TranslationRequests {
		wtg.Add(1)

		if err := sem.Acquire(semCtx, 1); err != nil {
			panic("Failed to acquire semaphore")
		}

		go RequestTranslationDirector(request, &wtg, bar, responseChannel, sem)

	}

	wtg.Wait()

	//We are not expecting more values
	close(responseChannel)

	allResponses := []*TranslationResponse{}

	for response := range responseChannel {
		allResponses = append(allResponses, response)
	}

	//Return sorted by translation id
	sort.SliceStable(allResponses, func(i, j int) bool {
		return allResponses[i].TranslationRequest.Id < allResponses[j].TranslationRequest.Id
	})

	response := &BatchTranslationResponse{
		RequestId:            shortUUID,
		TranslationResponses: allResponses,
	}

	if batchRequest.FileBaseName != "" && batchRequest.FileSavePath != "" {
		//We should save to the disk instead
		SaveBatchResponseToFile(batchRequest.FileBaseName, batchRequest.FileSavePath, response)
		return &BatchTranslationResponse{
			RequestId:            shortUUID,
			TranslationResponses: []*TranslationResponse{},
			ReturnedToDisk:       true,
		}
	}
	return response
}

func FindFailureReason(edge *TranslationEdge) (bool, bool) {
	isCompilationRuntimeError := false
	isTestError := false

	if edge.FuzzyTests != nil {
		//Find out the reason for the failure
		for _, test := range edge.FuzzyTests {

			if !test.ExitCodeZero {
				isCompilationRuntimeError = true
				break
			}

			if !test.Passed {
				isTestError = true
				break
			}
		}
	} else {
		for _, test := range edge.UnitTests {

			if !test.ExitCodeZero {
				isCompilationRuntimeError = true
				break
			}

			if !test.Passed {
				isTestError = true
				break
			}
		}
	}

	return isCompilationRuntimeError, isTestError
}

func signalCancelProcessing(allPaths []Path) {
	for _, path := range allPaths {

		for _, edge := range path.Edges {
			if edge.GetStatus() == PENDING || edge.GetStatus() == PROCESSING {
				edge.SetStatus(SKIPPED_TRANSLATION_FOUND)
			}
		}
	}
}

func processParentEdge(edge *TranslationEdge, translationPath Path, allPaths []Path) {
	parentEdge := edge.ParentEdge

	switch parentEdge.GetStatus() {
	case FAILED, SKIPPED_PARENT_FAILED, FAILED_NO_EXTRACTED, FAILED_NO_INFERENCE, FAILED_EXECUTION, FAILED_VERIFICATION, FAILED_EXECUTION_TIMEOUT:
		edge.SetStatus(SKIPPED_PARENT_FAILED)
	case TRANSLATION_FOUND, SKIPPED_TRANSLATION_FOUND:
		edge.SetStatus(SKIPPED_TRANSLATION_FOUND)
		if common.ConfigStore.EarlyStopOnTranslationSuccess {
			signalCancelProcessing(allPaths)
		}
	case SUCCESS, TRANSLATED:
		edge.SourceCode = parentEdge.ExtractedSourceCode
		prompt := PreparePrompt(edge)
		edge.Prompt = prompt
		PerformTranslationStep(edge, translationPath.FinalTarget)
	default:
		fmt.Println(parentEdge.GetStatus())
		panic("There is a bug. Code should not reach here ever")
	}
}

func processRootNode(edge *TranslationEdge, translationPath Path, allPaths []Path) {
	if edge.GetStatus() != SKIPPED_TRANSLATION_FOUND {
		prompt := PreparePrompt(edge)
		edge.Prompt = prompt
		PerformTranslationStep(edge, translationPath.FinalTarget)
		if edge.GetStatus() == TRANSLATION_FOUND && common.ConfigStore.EarlyStopOnTranslationSuccess {
			signalCancelProcessing(allPaths)
		}
	}
}

func processTranslationPath(translationPath Path, allPaths []Path, processedChannel chan Path, progressbar *uiprogress.Bar, wg *sync.WaitGroup) {
	defer wg.Done()

	for _, edge := range translationPath.Edges {
		//This is to force dependencies in the execution order of the edges across the concurrent Path translations
		edge.ProcessingMutex.Lock()

		//If the current edge is not pending, it was already explored in another sub path and we can reuse it
		if edge.GetStatus() == PENDING {
			translationPath.UsedMemoizedEdgeIndex = append(translationPath.UsedMemoizedEdgeIndex, false)
			if edge.ParentEdge != nil {
				processParentEdge(edge, translationPath, allPaths)
			} else {
				processRootNode(edge, translationPath, allPaths)
			}
		} else {
			translationPath.UsedMemoizedEdgeIndex = append(translationPath.UsedMemoizedEdgeIndex, true)
		}

		edge.ProcessingMutex.Unlock()
	}

	progressbar.Incr()
	processedChannel <- translationPath
}

func locateFunctionNameCPP(code string) (string, string) {
	// Remove single-line comments
	re := regexp.MustCompile(`//.+?\n`)
	code4func := re.ReplaceAllString(code, "")

	// Regex pattern to match function signature
	pattern := regexp.MustCompile(`([\w\s\*]+)\s(\w+)\s?\(\s?(\w+.*\w*)?\s?\)`)
	methodInfo := pattern.FindStringSubmatch(code4func)

	if len(methodInfo) == 0 {
		return "", ""
	}

	// Compile the pattern to match the Java method with its body
	startIndex := pattern.FindStringIndex(code)

	if startIndex == nil {
		return "", ""
	}

	openBraces := 0
	endIndex := startIndex[1]
	inMethod := false

	for i := startIndex[1]; i < len(code); i++ {
		if code[i] == '{' {
			openBraces++
			inMethod = true
		} else if code[i] == '}' {
			openBraces--
		}

		if inMethod && openBraces == 0 {
			endIndex = i + 1
			break
		}
	}

	extractedBody := code[startIndex[0]:endIndex]

	return methodInfo[2], extractedBody
}

func locateFunctionNameJava(code string) (string, string, []string) {
	// Compile the pattern to match the Java method signature
	pattern := regexp.MustCompile(`(public|private|protected)?\s?(static)?\s?(\w+|\w+\[\])\s(\w+)\s?\(\s?(\w+.*\w*)?\s?\)`)
	methodInfo := pattern.FindStringSubmatch(code)

	// Compile patterns to match parameter names
	patternPa1 := regexp.MustCompile(`\w+\s(\w+)`)
	patternPa2 := regexp.MustCompile(`\w+\s?[\[\s?\]]+\s(\w+)`)

	var varList []string

	if len(methodInfo) > 0 {
		params := strings.Split(methodInfo[5], ",")
		for _, param := range params {
			if matches := patternPa1.FindStringSubmatch(param); len(matches) > 0 {
				varList = append(varList, matches[1])
			} else if matches := patternPa2.FindStringSubmatch(param); len(matches) > 0 {
				varList = append(varList, matches[1])
			}
		}
		return methodInfo[4], methodInfo[3], varList
	}

	return "", "", nil
}

func extractJavaMethod(code, methodName string) string {
	pattern := regexp.MustCompile(fmt.Sprintf(`(public|private|protected)?\s?(static)?\s?(\w+|\w+\[\])\s%s\s?\(\s?(\w+.*\w*)?\s?\)\s?\{`, methodName))
	startIndex := pattern.FindStringIndex(code)

	if startIndex == nil {
		return ""
	}

	lastBraceIndex := -1

	for i := startIndex[1]; i < len(code); i++ {
		if code[i] == '}' {
			lastBraceIndex = i
		}
	}

	if lastBraceIndex == -1 {
		//Couldn't find the closing brace
		return ""
	}

	return code[startIndex[0] : lastBraceIndex+1]
}

func locateFunctionNameAndBodyPython(code string) (string, string) {
	// Remove comments and docstrings
	reComment := regexp.MustCompile(`#.*`)
	codeNoComments := reComment.ReplaceAllString(code, "")

	reDocstring := regexp.MustCompile(`("""(?:[^"\\]|\\.)*"""|'''(?:[^'\\]|\\.)*''')`)
	codeNoComments = reDocstring.ReplaceAllString(codeNoComments, "")

	// Regex pattern to match the function name and body
	pattern := regexp.MustCompile(`def\s+(\w+)\s*\(([^)]*)\)\s*:\s*([^#]*)(?:#.*)?`)
	matches := pattern.FindAllStringSubmatch(codeNoComments, -1)

	if len(matches) == 0 {
		return "", ""
	}

	functionName := matches[0][1]
	functionBody := matches[0][0]

	return functionName, functionBody
}

func ExtractFunctionForTranscoderTests(translationEdge *TranslationEdge) string {
	var extractedFunction string

	switch translationEdge.TargetLanguage {
	case "Java":
		functionName, _, _ := locateFunctionNameJava(translationEdge.ExtractedSourceCode)
		functionExtracted := extractJavaMethod(translationEdge.ExtractedSourceCode, functionName)
		extractedFunction = strings.ReplaceAll(functionExtracted, functionName, "f_filled")

		//TransCoder tests call static function f_filled
		if !strings.Contains(extractedFunction, "public static") {
			extractedFunction = strings.ReplaceAll(extractedFunction, "public", "public static")
		}

	case "C++":
		functionName, fullFunction := locateFunctionNameCPP(translationEdge.ExtractedSourceCode)
		extractedFunction = strings.ReplaceAll(fullFunction, functionName, "f_filled")
	case "Python":
		functionName, functionBody := locateFunctionNameAndBodyPython(translationEdge.ExtractedSourceCode)
		extractedFunction = strings.ReplaceAll(functionBody, functionName, "f_filled")
	default:
		panic("Unsupported language for TransCoder Test Suite")
	}

	return extractedFunction
}

func PreparePrompt(translationEdge *TranslationEdge) string {
	replacedCode := strings.ReplaceAll(translationEdge.PromptTemplate, "{input_code}", translationEdge.SourceCode)
	replacedInputLang := strings.ReplaceAll(replacedCode, "{input_lang}", translationEdge.InputLanguage)
	prompt := strings.ReplaceAll(replacedInputLang, "{target_lang}", translationEdge.TargetLanguage)

	if translationEdge.ExtraPromptData != "" {
		prompt = strings.ReplaceAll(prompt, "{extra_prompt_data}", translationEdge.ExtraPromptData)
	}

	if translationEdge.SuggestedTargetSignature != "" {
		prompt = strings.ReplaceAll(prompt, "{signature}", translationEdge.SuggestedTargetSignature)
	}

	//This is specific to the Transcoder Prompt
	if common.ConfigStore.UseTranscoderTestFormat {
		switch translationEdge.TargetLanguage {
		case "Python":
			prompt = strings.ReplaceAll(prompt, "{comment_separator}", "#")
		case "Java", "C++":
			prompt = strings.ReplaceAll(prompt, "{comment_separator}", "//")
		case "Go":
			prompt = strings.ReplaceAll(prompt, "{comment_separator}", "//")
		case "JavaScript":
			prompt = strings.ReplaceAll(prompt, "{comment_separator}", "//")
		case "Rust":
			prompt = strings.ReplaceAll(prompt, "{comment_separator}", "//")
		default:
			panic("Unsupported language comment separator")
		}
	}

	return prompt
}

func ExtractSourceCode(originalPrompt string, regexTemplate string, inferenceResult string) (string, bool) {
	// Compile the regex pattern
	re := regexp.MustCompile(regexTemplate)
	var match []string

	// Find the first match
	if ConfigStore.ApplyRegexInferenceOnly {
		match = re.FindStringSubmatch(inferenceResult)
	} else {
		match = re.FindStringSubmatch(originalPrompt + inferenceResult)
	}

	if len(match) > 1 {
		// The first capturing group is at index 1 in the match slice
		firstGroup := match[1]

		return strings.TrimSpace(firstGroup), true
	} else {
		return "", false
	}
}

func PerformEdgeExecution(translationEdge *TranslationEdge, finalPathTarget string) {
	executorQueue := GetExecutorQueueInstance()
	fuzzyPassed := 0
	totalFuzzyTests := len(translationEdge.FuzzyTests)
	unitTestPassed := 0
	totalUnitTests := len(translationEdge.UnitTests)
	totalExecutionTime := time.Duration(0)
	//TODO: This is a common variable because we don't support both fuzzy and unit at the same time
	//TODO: Disabled for now as it is necessary for PerformVerification functionality. Enabling this returns without executing all tests when one fails
	//finishEarly := false

	//Verify the results using the fuzzy test cases
	if translationEdge.FuzzyTests != nil {

		for index, test := range translationEdge.FuzzyTests {

			//FIXME: Disabled for now. This returns without executing all tests when one fails
			// if finishEarly {
			// 	break
			// }

			executionUnit := &ExecutionUnit{
				StdinData:     test.Input,
				SourceCode:    translationEdge.ExtractedSourceCode,
				Language:      translationEdge.TargetLanguage,
				OutputChannel: make(chan ExecutionUnit),
				ExecutionType: RUN,
			}

			//Send for execution
			executorQueue.InputChannel <- *executionUnit
			executionResult := <-executionUnit.OutputChannel
			totalExecutionTime += executionResult.WallTime

			//FIXME: Exit early if at least one of the tests fails to save computing
			if !executionResult.Success {
				translationEdge.UpdatePendingStatus(FAILED_EXECUTION)
				// finishEarly = true
			} else if executionResult.ExecutionOutput == "CMD_TIMEOUT_KILLED" {
				translationEdge.UpdatePendingStatus(FAILED_EXECUTION_TIMEOUT)
				// finishEarly = true
			} else if executionResult.ExecutionOutput == "FAIL_INVALID_UTF8_STRING" {
				translationEdge.UpdatePendingStatus(FAILED_EXECUTION)
				// finishEarly = true
			}

			//TODO: Could be better but depends on how we measure this
			if strings.TrimSpace(executionResult.ExecutionOutput) == strings.TrimSpace(test.ExpectedOutput) {
				fuzzyPassed++
				translationEdge.FuzzyTests[index].Passed = true
			} else {
				translationEdge.FuzzyTests[index].Passed = false
				//FIXME: Exit early if at least one of the tests fails to save computing
				translationEdge.UpdatePendingStatus(FAILED_VERIFICATION)
			}

			translationEdge.FuzzyTests[index].ActualOutput = strings.TrimSpace(executionResult.ExecutionOutput)
			translationEdge.FuzzyTests[index].ExecutedCode = executionResult.ExecutedCode
			translationEdge.FuzzyTests[index].ExitCodeZero = executionResult.Success
		}

	}

	//Verify the results using the fuzzy test cases
	if translationEdge.UnitTests != nil {

		for index, test := range translationEdge.UnitTests {

			//TODO: Disabled for now. This returns without executing all tests when one fails
			// if finishEarly {
			// 	break
			// }

			var codeWithTest string

			//Attach any imports as necessary (mostly for Go)
			if test.Imports != "" {
				codeWithTest = test.Imports + "\n" + translationEdge.ExtractedSourceCode + "\n" + test.SourceCode
			} else {
				if common.ConfigStore.UseTranscoderTestFormat {
					extractedFunction := ExtractFunctionForTranscoderTests(translationEdge)
					if translationEdge.TargetLanguage == "Java" || translationEdge.TargetLanguage == "C++" {
						codeWithTest = strings.ReplaceAll(test.SourceCode, "//TOFILL", extractedFunction)
					} else if translationEdge.TargetLanguage == "Python" {
						codeWithTest = strings.ReplaceAll(test.SourceCode, "#TOFILL", extractedFunction)
					}

				} else {
					//TODO: We concatenate the test as they do in HumanEval.
					codeWithTest = translationEdge.ExtractedSourceCode + "\n" + test.SourceCode
				}

			}

			executionUnit := &ExecutionUnit{
				SourceCode:    codeWithTest,
				Language:      translationEdge.TargetLanguage,
				OutputChannel: make(chan ExecutionUnit),
				ExecutionType: TEST,
			}

			//Send for execution
			executorQueue.InputChannel <- *executionUnit
			executionResult := <-executionUnit.OutputChannel
			totalExecutionTime += executionResult.WallTime

			//TODO: Exit early if at least one of the tests fails to save computing
			if !executionResult.Success {
				translationEdge.UpdatePendingStatus(FAILED_EXECUTION)
				// finishEarly = true
			} else if executionResult.ExecutionOutput == "CMD_TIMEOUT_KILLED" {
				translationEdge.UpdatePendingStatus(FAILED_EXECUTION_TIMEOUT)
				// finishEarly = true
			} else if executionResult.ExecutionOutput == "FAIL_INVALID_UTF8_STRING" {
				translationEdge.UpdatePendingStatus(FAILED_EXECUTION)
				// finishEarly = true
			}

			if common.ConfigStore.UseTranscoderTestFormat {
				//TODO: There should be an error code when the regex doesn't match
				if executionResult.Success {
					correct, _ := VerifyTranscoderTestCase(executionResult.ExecutionOutput)

					if correct {
						unitTestPassed++
						translationEdge.UnitTests[index].Passed = true
					} else {
						translationEdge.UpdatePendingStatus(FAILED_VERIFICATION)
						// finishEarly = true
					}
				}
			} else {
				if executionResult.Success {
					unitTestPassed++
					translationEdge.UnitTests[index].Passed = true
				} else {
					//For these cases in HumanEval-X, we need to parse the output to see if it was compilation or assertion error
					translationEdge.UpdatePendingStatus(FAILED)
					// finishEarly = true
				}
			}

			translationEdge.UnitTests[index].ActualOutput = executionResult.ExecutionOutput
			translationEdge.UnitTests[index].ExecutedCode = executionResult.ExecutedCode
			translationEdge.UnitTests[index].ExitCodeZero = executionResult.Success
		}

	}

	translationEdge.WallClockTestExecutionTime = time.Duration(totalExecutionTime)

	if fuzzyPassed == totalFuzzyTests && unitTestPassed == totalUnitTests {
		if translationEdge.TargetLanguage == finalPathTarget {
			translationEdge.UpdatePendingStatus(TRANSLATION_FOUND)
		} else {
			translationEdge.UpdatePendingStatus(SUCCESS)
		}
	} else {
		translationEdge.UpdatePendingStatus(FAILED)
	}
}

func PerformTranslationStep(translationEdge *TranslationEdge, finalPathTarget string) {
	translationEdge.SetStatus(PROCESSING)

	inferenceQueue := GetInferenceQueueInstance()

	var inferenceResult InferenceResult

	inferenceUnit := &InferenceUnit{
		Prompt:        translationEdge.Prompt,
		ModelName:     translationEdge.ModelName,
		OutputChannel: make(chan InferenceResult, 1),
	}

	inferenceQueue.InputChannel <- *inferenceUnit
	inferenceResult = <-inferenceUnit.OutputChannel
	translationEdge.InferenceOutput = inferenceResult.Response
	translationEdge.UsedInferenceCache = inferenceResult.IsCached

	if !inferenceResult.Success {
		translationEdge.SetStatus(FAILED_NO_INFERENCE)
		return
	}

	//Extract the source code
	extracted, extractedOk := ExtractSourceCode(translationEdge.Prompt, translationEdge.RegexTemplate, inferenceResult.Response)

	//Can't process downstream edges as we weren't able to extract the code
	if !extractedOk {
		translationEdge.SetStatus(FAILED_NO_EXTRACTED)
		return
	}

	translationEdge.ExtractedSourceCode = extracted

	if !common.ConfigStore.VerifyIntermediateTranslations && translationEdge.TargetLanguage != finalPathTarget {
		translationEdge.UpdatePendingStatus(TRANSLATED)
		return
	}

	PerformEdgeExecution(translationEdge, finalPathTarget)
}

func VerifyTranscoderTestCase(input string) (bool, error) {
	// Regular expression to match the two integers in the string
	re := regexp.MustCompile(`(\d+), (\d+)`)
	matches := re.FindStringSubmatch(input)

	// If the pattern doesn't match, return an error
	if len(matches) != 3 {
		fmt.Println("Error extracting from TransCoder unit test results")
		return false, fmt.Errorf("input string does not match expected format")
	}

	return matches[1] == matches[2], nil
}

func GetPromptTemplate(templateName string) string {

	//Get the requested prompt template
	for name, template := range common.ConfigStore.PromptTemplates {

		if name == templateName {
			return template
		}

	}

	panic("Requested template not found")
}

func GetRegexTemplate(templateName string) string {
	//Get the requested regex template
	for name, template := range common.ConfigStore.RegexTemplates {

		if name == templateName {
			return template
		}

	}

	panic("Requested regex template not found")
}

func RequestTranslationDirectorCAK(translationRequest *TranslationRequest, wtg *sync.WaitGroup, progressbar *uiprogress.Bar, responseChannel chan *TranslationResponse, semaphore *semaphore.Weighted) {
	defer wtg.Done()
	defer semaphore.Release(1)

	if common.ConfigStore.ExpansionDepth != 1 {
		panic("You need only direct translations")
	}

	if common.ConfigStore.Seed != -1 {
		panic("You need to disable the seed")
	}

	if common.ConfigStore.UseInferenceCache {
		panic("Inference cache must not be used for CA@k")
	}

	if common.ConfigStore.UseResponseCache {
		//Try to load from cache if this was already processed in another run
		response, err := LoadExistingResponse(translationRequest)

		if !err {
			fmt.Println("Used from cache")
			//FIXME: This should not be hardcoded
			for i := 0; i < 1; i++ {
				progressbar.Incr()
			}
			responseChannel <- &response
			return
		}

	}

	promptTemplate := GetPromptTemplate(translationRequest.PromptTemplateName)
	regexTemplate := GetRegexTemplate(translationRequest.RegexTemplateName)

	//For the Edge Id
	counter := NewCounter()

	translationPaths := []Path{}

	for range 10 {

		edge := &TranslationEdge{
			Id:              counter.Next(),
			TranslationId:   translationRequest.Id,
			InputLanguage:   translationRequest.SeedLanguage,
			TargetLanguage:  translationRequest.TargetLanguage,
			Level:           common.ConfigStore.ExpansionDepth,
			ProcessingMutex: &sync.Mutex{},
			StatusMutex:     &sync.Mutex{},
			SourceCode:      translationRequest.SeedCode,
			PromptTemplate:  promptTemplate,
			FuzzyTests:      []FuzzyTest{},
			UnitTests:       []UnitTest{},
			RegexTemplate:   regexTemplate,
			ModelName:       translationRequest.ModelName,
			ExtraPromptData: translationRequest.ExtraPromptData,
		}

		//Make sure to include unit tests in the edge
		AttachTestSuiteFromRequest(edge, translationRequest)

		//Unit tests may need the target signature to be leaked to work
		AttachTargetSignatureFromRequest(edge, translationRequest)

		path := Path{
			FinalTarget: translationRequest.TargetLanguage,
		}
		path.Add(edge)

		translationPaths = append(translationPaths, path)
	}

	allPaths := translationPaths

	//Sort them to prioritize translations to the request target
	PrioritizeShallowFirst(allPaths)

	// fmt.Println(len(allPaths))

	// levels := make(map[int]int)

	// for _, cpath := range allPaths {
	// 	for _, edge := range cpath.Edges {
	// 		levels[edge.Level] += 1
	// 	}
	// }

	// fmt.Println(levels)

	// panic("test")

	processedChannel := make(chan Path, len(allPaths))

	wg := sync.WaitGroup{}

	for _, path := range allPaths {
		wg.Add(1)

		//Disable concurrent branch processing for compute saving mode
		if common.ConfigStore.ComputeEfficientMode {
			processTranslationPath(path, allPaths, processedChannel, progressbar, &wg)
		} else {
			go processTranslationPath(path, allPaths, processedChannel, progressbar, &wg)
		}

	}

	// Wait for path translation goroutines to finish
	wg.Wait()

	//All goroutines ended, so we are not expecting new values
	close(processedChannel)
	translationResponse := ConvertPathsToResponse(processedChannel, translationRequest)

	if common.ConfigStore.UseResponseCache {
		common.SaveResponseToCache(translationRequest, translationResponse)
	}
	responseChannel <- translationResponse
}

func RequestTranslationDirector(translationRequest *TranslationRequest, wtg *sync.WaitGroup, progressbar *uiprogress.Bar, responseChannel chan *TranslationResponse, semaphore *semaphore.Weighted) {
	defer wtg.Done()
	defer semaphore.Release(1)

	if common.ConfigStore.UseResponseCache {
		//Try to load from cache if this was already processed in another run
		response, err := LoadExistingResponse(translationRequest)

		if !err {
			fmt.Println("Used from cache")
			//FIXME: This should not be hardcoded
			for i := 0; i < 21; i++ {
				progressbar.Incr()
			}
			responseChannel <- &response
			return
		}

	}

	initialPath := &Path{
		FinalTarget: translationRequest.TargetLanguage,
	}
	maxDepth := common.ConfigStore.ExpansionDepth

	promptTemplate := GetPromptTemplate(translationRequest.PromptTemplateName)
	regexTemplate := GetRegexTemplate(translationRequest.RegexTemplateName)

	//For the Edge Id
	counter := NewCounter()

	translationPaths := &TranslationPaths{
		Paths: []Path{},
	}

	BuildIntermediatesTranslationTree(translationRequest, promptTemplate, regexTemplate, translationRequest.UsedLanguages, translationRequest.SeedLanguage, translationRequest.TargetLanguage, translationRequest.SeedCode, 1, maxDepth, nil, initialPath, translationPaths, counter)

	allPaths := translationPaths.Paths

	//Sort them to prioritize translations to the request target
	PrioritizeShallowFirst(allPaths)

	// fmt.Println(len(allPaths))

	// levels := make(map[int]int)

	// for _, cpath := range allPaths {
	// 	for _, edge := range cpath.Edges {
	// 		levels[edge.Level] += 1
	// 	}
	// }

	// fmt.Println(levels)

	// panic("test")

	processedChannel := make(chan Path, len(allPaths))

	wg := sync.WaitGroup{}

	for _, path := range allPaths {
		wg.Add(1)

		//Disable concurrent branch processing for compute saving mode
		if common.ConfigStore.ComputeEfficientMode {
			processTranslationPath(path, allPaths, processedChannel, progressbar, &wg)
		} else {
			go processTranslationPath(path, allPaths, processedChannel, progressbar, &wg)
		}

	}

	// Wait for path translation goroutines to finish
	wg.Wait()

	//All goroutines ended, so we are not expecting new values
	close(processedChannel)
	translationResponse := ConvertPathsToResponse(processedChannel, translationRequest)

	if common.ConfigStore.UseResponseCache {
		common.SaveResponseToCache(translationRequest, translationResponse)
	}
	responseChannel <- translationResponse
}

func ConvertPathsToResponse(paths chan Path, translationRequest *TranslationRequest) *TranslationResponse {
	responsePaths := []*ResponseTranslationPath{}

	for path := range paths {

		responseEdges := []*ResponseTranslationEdge{}

		for _, edge := range path.Edges {
			responseEdges = append(responseEdges, ConvertToEdgeResponse(edge))
		}

		responsePath := &ResponseTranslationPath{
			TranslationEdges:  responseEdges,
			EdgeIndexMemoized: path.UsedMemoizedEdgeIndex,
		}

		responsePaths = append(responsePaths, responsePath)
	}

	response := &TranslationResponse{
		TranslationRequest: translationRequest,
		Paths:              responsePaths,
	}

	return response
}

func GetParentIdOrNone(edge *TranslationEdge) int {
	if edge.ParentEdge == nil {
		return -1
	} else {
		return edge.ParentEdge.Id
	}
}

func ConvertToEdgeResponse(edge *TranslationEdge) *ResponseTranslationEdge {

	fuzzyTests := []*ResponseFuzzyTestCase{}
	unitTests := []*ResponseUnitTestCase{}

	for _, test := range edge.FuzzyTests {
		fuzzyTests = append(fuzzyTests, test.ToResponse())
	}

	for _, test := range edge.UnitTests {
		unitTests = append(unitTests, test.ToResponse())
	}

	responseEdge := &ResponseTranslationEdge{
		EdgeId:                int32(edge.Id),
		PromptTemplate:        edge.PromptTemplate,
		Prompt:                edge.Prompt,
		InputLanguage:         edge.InputLanguage,
		TargetLanguage:        edge.TargetLanguage,
		InferenceOutput:       edge.InferenceOutput,
		SourceCode:            edge.SourceCode,
		ExtractedSourceCode:   edge.ExtractedSourceCode,
		Level:                 int32(edge.Level),
		ParentEdgeId:          int32(GetParentIdOrNone(edge)),
		Success:               edge.Success,
		Status:                edge.GetStatus().String(),
		FuzzyTests:            fuzzyTests,
		UnitTests:             unitTests,
		WallTimeInference:     edge.WallClockInferenceTime.Milliseconds(),
		WallTimeTestExecution: edge.WallClockTestExecutionTime.Milliseconds(),
		UsedInferenceCache:    edge.UsedInferenceCache,
	}

	return responseEdge
}

func ConvertFromEdgeResponse(responseEdge *ResponseTranslationEdge) *TranslationEdge {

	fuzzyTests := []FuzzyTest{}
	unitTests := []UnitTest{}

	for _, test := range responseEdge.FuzzyTests {
		fuzzyTests = append(fuzzyTests, common.FromResponseFuzzyTest(test))
	}

	for _, test := range responseEdge.UnitTests {
		unitTests = append(unitTests, common.FromResponseUnitTest(test))
	}

	edge := &TranslationEdge{
		Id:                         int(responseEdge.EdgeId),
		PromptTemplate:             responseEdge.PromptTemplate,
		Prompt:                     responseEdge.Prompt,
		InputLanguage:              responseEdge.InputLanguage,
		TargetLanguage:             responseEdge.TargetLanguage,
		InferenceOutput:            responseEdge.InferenceOutput,
		SourceCode:                 responseEdge.SourceCode,
		ExtractedSourceCode:        responseEdge.ExtractedSourceCode,
		Level:                      int(responseEdge.Level),
		ParentEdge:                 nil, // ParentEdge needs to be set separately if available
		ProcessingMutex:            &sync.Mutex{},
		StatusMutex:                &sync.Mutex{},
		FuzzyTests:                 fuzzyTests,
		UnitTests:                  unitTests,
		WallClockInferenceTime:     time.Duration(responseEdge.WallTimeInference) * time.Millisecond,
		WallClockTestExecutionTime: time.Duration(responseEdge.WallTimeTestExecution) * time.Millisecond,
		UsedInferenceCache:         responseEdge.UsedInferenceCache,
	}

	edge.SetStatus(common.ParseStatus(responseEdge.Status))

	return edge
}

func PrioritizeShallowFirst(allPaths []Path) {
	//Brings critical paths to the front of the list
	sort.SliceStable(allPaths, func(i, j int) bool {
		return len(allPaths[i].Edges) <= len(allPaths[j].Edges)
	})
}

func AttachTargetSignatureFromRequest(translationEdge *TranslationEdge, translationRequest *TranslationRequest) {

	if translationRequest.TargetSignatures != nil {

		for _, signature := range translationRequest.TargetSignatures {

			if signature.Language == translationEdge.TargetLanguage {
				translationEdge.SuggestedTargetSignature = signature.Signature
				break
			}
		}

	}
}

func AttachTestSuiteFromRequest(translationEdge *TranslationEdge, translationRequest *TranslationRequest) {

	if translationRequest.TestSuite == nil || len(translationRequest.TestSuite.UnitTestSuite) == 0 && len(translationRequest.TestSuite.FuzzySuite) == 0 {
		panic("Please provide unit tests")
	}

	//Fuzzy tests are language independent so we can evaluate all of them
	for _, test := range translationRequest.TestSuite.FuzzySuite {

		fuzzyTest := FuzzyTest{
			Input:          test.StdinInput,
			ExpectedOutput: test.ExpectedOutput,
		}

		translationEdge.FuzzyTests = append(translationEdge.FuzzyTests, fuzzyTest)
	}

	if len(translationRequest.TestSuite.UnitTestSuite) > 0 {
		//Compatible unit tests with the target language
		compatibleCases := []*UnitTestCase{}

		for _, unitTest := range translationRequest.TestSuite.UnitTestSuite {

			if unitTest.Language == translationEdge.TargetLanguage {
				compatibleCases = append(compatibleCases, unitTest)
			}
		}

		if len(compatibleCases) == 0 && ConfigStore.VerifyIntermediateTranslations {
			fmt.Println(translationRequest.TargetLanguage)
			panic("If you use unit tests, you must include test cases for all intermediate languages.")
		}

		for _, compatibleCase := range compatibleCases {
			unitTest := UnitTest{
				SourceCode: compatibleCase.TestCase,
				Imports:    compatibleCase.Imports,
			}

			translationEdge.UnitTests = append(translationEdge.UnitTests, unitTest)
		}

	}

	if len(translationRequest.TestSuite.UnitTestSuite) > 0 && len(translationRequest.TestSuite.FuzzySuite) > 0 {
		panic("We don't yet support evaluation with mixed fuzzy and unit tests. Feel free to contribute to this :=)")
	}

}

// BuildTranslationTree builds a translation tree and collects all paths.
func BuildIntermediatesTranslationTree(translationRequest *TranslationRequest, promptTemplate string, regexTemplate string, languages []string, inputLanguage string, requestTargetLanguage string, seedCode string, depth int, maxDepth int, parent *TranslationEdge, currentPath *Path, allPaths *TranslationPaths, counter *Counter) {

	if depth > maxDepth {
		return
	}

	for _, language := range languages {

		if language != inputLanguage {

			edge := &TranslationEdge{
				Id:              counter.Next(),
				TranslationId:   translationRequest.Id,
				InputLanguage:   inputLanguage,
				TargetLanguage:  language,
				Level:           depth,
				ParentEdge:      parent,
				ProcessingMutex: &sync.Mutex{},
				StatusMutex:     &sync.Mutex{},
				SourceCode:      seedCode,
				PromptTemplate:  promptTemplate,
				FuzzyTests:      []FuzzyTest{},
				UnitTests:       []UnitTest{},
				RegexTemplate:   regexTemplate,
				ModelName:       translationRequest.ModelName,
			}

			//Make sure to include unit tests in the edge
			AttachTestSuiteFromRequest(edge, translationRequest)

			//Unit tests may need the target signature to be leaked to work
			AttachTargetSignatureFromRequest(edge, translationRequest)

			// Add the current edge to the path
			newPath := currentPath.Copy()
			newPath.Add(edge)

			//If this path visits the target language of the request, mark it as priority to early exit on successful translation
			if edge.TargetLanguage == newPath.FinalTarget {
				finalPath := newPath.Copy()
				allPaths.Add(finalPath)
			} else {
				// Continue to build the tree
				BuildIntermediatesTranslationTree(translationRequest, promptTemplate, regexTemplate, languages, language, requestTargetLanguage, "", depth+1, maxDepth, edge, newPath, allPaths, counter)

			}
		}
	}

}
