package common

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	NumExecutionWorkers            int               `yaml:"numExecutionWorkers"`
	NumInferenceWorkers            int               `yaml:"numInferenceWorkers"`
	InferenceApiBaseUrls           []string          `yaml:"inferenceApiBaseUrls"`
	InferenceApiToken              string            `yaml:"inferenceApiToken"`
	ServerAddress                  string            `yaml:"serverAddress"`
	ServerPort                     string            `yaml:"serverPort"`
	ExpansionDepth                 int               `yaml:"expansionIntermediaryNodes"`
	PromptTemplates                map[string]string `yaml:"promptTemplates"`
	RegexTemplates                 map[string]string `yaml:"regexTemplates"`
	ExecutionContainers            map[string]string `yaml:"executionContainers"`
	ComputeEfficientMode           bool              `yaml:"useComputeEfficientMode"`
	ApplyRegexInferenceOnly        bool              `yaml:"applyRegexInferenceOnly"`
	EarlyStopOnTranslationSuccess  bool              `yaml:"earlyStop"`
	UseTranscoderTestFormat        bool              `yaml:"useTranscoderTestFormat"`
	VerifyIntermediateTranslations bool              `yaml:"verifyIntermediateTranslations"`
	StopOnDirectTranslation        bool              `yaml:"stopOnDirectTranslation"`
	UseIntermediatesMemoization    bool              `yaml:"useCrossPathIntermediatesMemoization"`
	UseInferenceCache              bool              `yaml:"useInferenceCache"`
	UseResponseCache               bool              `yaml:"useResponseCache"`
	UseExecutionCache              bool              `yaml:"useExecutionCache"`
	MaxGeneratedTokens             int               `yaml:"maxGeneratedTokens"`
	TopP                           float32           `yaml:"top-p"`
	TopK                           int               `yaml:"top-k"`
	Temperature                    float32           `yaml:"temperature"`
	Seed                           int               `yaml:"inferenceSeed"`
	DatabasePath                   string            `yaml:"cacheDatabasePath"`
	InferenceBackend               string            `yaml:"inferenceBackend"`
}

var ConfigStore AppConfig

func LoadConfig(filename string) error {

	data, err := os.ReadFile(filename)

	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	err = yaml.Unmarshal(data, &ConfigStore)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

var db *badger.DB

func StoreDatabase(database *badger.DB) {
	db = database
}

func GetDatabase() *badger.DB {
	return db
}

func SaveResponseToCache(request *TranslationRequest, response *TranslationResponse) {
	key := GetResponseKey(request)

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	if err := encoder.Encode(response); err != nil {
		return
	}

	err := db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(key), buf.Bytes())
		return err
	})

	if err != nil {
		fmt.Println("Error saving to cache database")
	}

}

func SaveExecutionToCache(request *ExecutionUnit) {
	key := GetExecutionKey(request)

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	if err := encoder.Encode(request); err != nil {
		return
	}

	if buf.Len() == 0 {
		fmt.Println("Buffer is empty, not saving to cache database")
		return
	}

	err := db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(key), buf.Bytes())
		if err != nil {
			return err
		}
		return nil
	})

	LoadExistingExecutionResults(request)

	if err != nil {
		fmt.Println("Error saving execution to cache database")
	}

}

func LoadExistingExecutionResults(request *ExecutionUnit) (ExecutionUnit, bool) {
	key := GetExecutionKey(request)
	obj := []byte{}

	err := db.View(func(txn *badger.Txn) error {

		item, err := txn.Get([]byte(key))

		if err != nil {
			return err
		}

		err_item := item.Value(func(val []byte) error {

			// Copying or parsing val is valid.
			obj = append(obj, val...)
			return nil
		})

		if err_item != nil {
			return err_item
		}

		return nil

	})

	if err != nil {
		return ExecutionUnit{}, true
	}

	var response ExecutionUnit

	if len(obj) == 0 {
		return ExecutionUnit{}, true
	}

	buf := bytes.NewBuffer(obj)
	decoder := gob.NewDecoder(buf)

	if err := decoder.Decode(&response); err != nil {
		fmt.Println("Error decoding cached execution")
		fmt.Println(err)
		return ExecutionUnit{}, true
	}

	return response, false

}

func LoadExistingResponse(request *TranslationRequest) (TranslationResponse, bool) {
	key := GetResponseKey(request)
	obj := []byte{}

	err := db.View(func(txn *badger.Txn) error {

		item, err := txn.Get([]byte(key))

		if err != nil {
			return err
		}

		err_item := item.Value(func(val []byte) error {
			obj = append(obj, val...)
			return nil
		})

		if err_item != nil {
			return err_item
		}

		return nil

	})

	if err != nil {
		return TranslationResponse{}, true
	}

	var response TranslationResponse
	buf := bytes.NewBuffer(obj)
	decoder := gob.NewDecoder(buf)

	if err := decoder.Decode(&response); err != nil {
		fmt.Println("Error decoding cached response")
		return TranslationResponse{}, true
	}

	fmt.Println("Load from disk")
	return response, false

}

func GetResponseKey(request *TranslationRequest) string {
	conf := strconv.Itoa(ConfigStore.ExpansionDepth) + strconv.FormatBool(ConfigStore.EarlyStopOnTranslationSuccess)
	s := request.SeedLanguage + request.TargetLanguage + request.SeedCode + request.ModelName + request.PromptTemplateName + request.RegexTemplateName + request.Id + conf
	hash := sha256.Sum256([]byte(s))
	hashString := fmt.Sprintf("%x", hash)
	return hashString
}

func GetExecutionKey(unit *ExecutionUnit) string {
	s := unit.SourceCode + unit.Language + unit.StdinData + unit.ExecutedCode + unit.ExecutionType.String()
	hash := sha256.Sum256([]byte(s))
	hashString := fmt.Sprintf("%x", hash)
	return hashString
}

func SaveInferenceResponseToCache(prompt string, modelName string, response InferenceResult) {
	key := GetInferenceKey(prompt, modelName)

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	if err := encoder.Encode(response); err != nil {
		return
	}

	err := db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(key), buf.Bytes())
		return err
	})

	if err != nil {
		fmt.Println("Error saving inference to cache database")
	}

}

func LoadInferenceExistingResponse(prompt string, modelName string) (InferenceResult, bool) {
	key := GetInferenceKey(prompt, modelName)
	var obj []byte

	err := GetDatabase().View(func(txn *badger.Txn) error {

		item, err := txn.Get([]byte(key))

		if err != nil {
			return err
		}

		err_item := item.Value(func(val []byte) error {
			obj = append([]byte{}, val...)
			return nil
		})

		if err_item != nil {
			return err_item
		}

		return nil

	})

	if err != nil {
		return InferenceResult{}, true
	}

	var response InferenceResult
	buf := bytes.NewBuffer(obj)
	decoder := gob.NewDecoder(buf)

	if err := decoder.Decode(&response); err != nil {
		fmt.Println("Error decoding cached response")
		return InferenceResult{}, true
	}

	return response, false

}

func GetInferenceKey(prompt string, modelName string) string {

	//FIXME: Not good enough for general usage maybe
	key := fmt.Sprintf("%s%s%d%t%f%f%d%d",
		modelName,
		prompt,
		ConfigStore.MaxGeneratedTokens,
		ConfigStore.Temperature,
		ConfigStore.TopP,
		ConfigStore.TopK,
		ConfigStore.Seed)

	hash := sha256.Sum256([]byte(key))
	hashString := fmt.Sprintf("%x", hash)
	return hashString
}

func SaveBatchResponseToFile(baseFileName string, baseFilePath string, response *BatchTranslationResponse) {
	fullFilePathNoExtension := filepath.Join(baseFilePath, baseFileName)

	data, err := proto.Marshal(response)

	if err != nil {
		fmt.Printf("Failed to serialize BatchTranslationResponse: %v\n", err)
	} else {
		// Write the serialized data to disk
		err = os.WriteFile(fullFilePathNoExtension+".bin", data, 0644)
		if err != nil {
			fmt.Printf("Failed to write BatchTranslationResponse to file: %v\n", err)
		}
	}

	jsonData, err := json.Marshal(response)

	if err != nil {
		fmt.Printf("Failed to serialize BatchTranslationResponse to JSON: %v\n", err)
	} else {
		err = os.WriteFile(fullFilePathNoExtension+".json", jsonData, 0644)

		if err != nil {
			fmt.Printf("Failed to write BatchTranslationResponse JSON to file: %v\n", err)
		}
	}

}

type InferenceResult struct {
	Response string
	IsCached bool
	WallTime time.Duration
	Success  bool
}

// Define the Path struct that behaves like a list
type Path struct {
	Edges                 []*TranslationEdge
	FinalTarget           string
	PriorityPathWeight    int
	UsedMemoizedEdgeIndex []bool
}

type TranslationPaths struct {
	Paths []Path
}

func (p *TranslationPaths) Add(path *Path) {
	p.Paths = append(p.Paths, *path)
}

// Define enum for Status
type ExecutionType int

const (
	UNK ExecutionType = iota
	TEST
	RUN
)

func (s ExecutionType) String() string {
	switch s {
	case TEST:
		return "TEST"
	case RUN:
		return "RUN"
	default:
		panic("ExcutionType not found")
	}
}

type ExecutionUnit struct {
	SourceCode         string
	Language           string
	StdinData          string
	ExecutionOutput    string
	Success            bool
	ExecutedCode       string
	OutputChannel      chan ExecutionUnit
	ExecutionType      ExecutionType
	WallTime           time.Duration
	UsedExecutionCache bool
}

type InferenceUnit struct {
	Prompt        string
	ModelName     string
	OutputChannel chan InferenceResult
	WallTime      time.Duration
}

type FuzzyTest struct {
	Input          string
	ExpectedOutput string
	ActualOutput   string
	Passed         bool
	ExecutedCode   string
	ExitCodeZero   bool
}

func (unit *FuzzyTest) ToResponse() *ResponseFuzzyTestCase {

	response := &ResponseFuzzyTestCase{
		StdinInput:     unit.Input,
		ExpectedOutput: unit.ExpectedOutput,
		ActualOutput:   unit.ActualOutput,
		Passed:         unit.Passed,
		ExecutedCode:   unit.ExecutedCode,
	}

	return response
}

func FromResponseFuzzyTest(response *ResponseFuzzyTestCase) FuzzyTest {

	return FuzzyTest{
		Input:          response.StdinInput,
		ExpectedOutput: response.ExpectedOutput,
		ActualOutput:   response.ActualOutput,
		Passed:         response.Passed,
		ExecutedCode:   response.ExecutedCode,
	}
}

type UnitTest struct {
	SourceCode   string
	ActualOutput string
	ExecutedCode string
	Passed       bool
	Imports      string
	ExitCodeZero bool
}

func (unit *UnitTest) ToResponse() *ResponseUnitTestCase {

	response := ResponseUnitTestCase{
		SourceCode:   unit.SourceCode,
		ActualOutput: unit.ActualOutput,
		Passed:       unit.Passed,
		ExecutedCode: unit.ExecutedCode,
	}

	return &response
}

func FromResponseUnitTest(response *ResponseUnitTestCase) UnitTest {
	return UnitTest{
		SourceCode:   response.SourceCode,
		ActualOutput: response.ActualOutput,
		Passed:       response.Passed,
		ExecutedCode: response.ExecutedCode,
	}
}

// Implement the Stringer interface for Path
func (p Path) String() string {
	edgeStrings := make([]string, len(p.Edges))
	for i, edge := range p.Edges {
		edgeStrings[i] = edge.String()
	}
	return strings.Join(edgeStrings, " >> ") + " -| "
}

func (p Path) AlreadyVisited(te *TranslationEdge) bool {
	// Loop through all edges in the path
	for _, edge := range p.Edges {
		// Check if the target language of the new edge already exists in the path
		if edge.TargetLanguage == te.TargetLanguage {
			return true
		}
	}
	return false
}

// Method to add an edge to the path
func (p *Path) Add(edge *TranslationEdge) {
	p.Edges = append(p.Edges, edge)
}

// Method to add an edge to the path
func (p *Path) IncreasePriorityWeight() {
	p.PriorityPathWeight++
}

// Method to remove the last edge from the path
func (p *Path) RemoveLast() {
	if len(p.Edges) > 0 {
		p.Edges = p.Edges[:len(p.Edges)-1]
	}
}

func (p *Path) Copy() *Path {
	newEdges := make([]*TranslationEdge, len(p.Edges))
	copy(newEdges, p.Edges)
	// Creating a new Path struct and assigning the same Edges slice
	return &Path{
		Edges:                 newEdges,
		PriorityPathWeight:    p.PriorityPathWeight,
		FinalTarget:           p.FinalTarget,
		UsedMemoizedEdgeIndex: []bool{},
	}
}

type TreeBranch struct {
	TranslationEdges   []TranslationEdge
	PriorityScheduling bool
}

// Counter struct to hold the current count
type Counter struct {
	count int
	mu    sync.Mutex // To ensure thread safety
}

// NewCounter initializes a new Counter
func NewCounter() *Counter {
	return &Counter{count: 0}
}

// Next increments the counter and returns the next number
func (c *Counter) Next() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
	return c.count
}

// Define enum for Status
type Status int

const (
	PENDING Status = iota
	PROCESSING
	FAILED
	SUCCESS
	SKIPPED_PARENT_FAILED
	SKIPPED_TRANSLATION_FOUND
	TRANSLATION_FOUND
	FAILED_NO_EXTRACTED
	FAILED_NO_INFERENCE
	TRANSLATED
	FAILED_EXECUTION
	FAILED_VERIFICATION
	FAILED_EXECUTION_TIMEOUT
)

// String method to convert Status to string
func (s Status) String() string {
	switch s {
	case PENDING:
		return "PENDING"
	case PROCESSING:
		return "PROCESSING"
	case FAILED:
		return "FAILED"
	case FAILED_EXECUTION:
		return "FAILED_EXECUTION"
	case FAILED_VERIFICATION:
		return "FAILED_VERIFICATION"
	case SUCCESS:
		return "SUCCESS"
	case TRANSLATION_FOUND:
		return "TRANSLATION_FOUND"
	case SKIPPED_PARENT_FAILED:
		return "SKIPPED_PARENT_FAILED"
	case SKIPPED_TRANSLATION_FOUND:
		return "SKIPPED_TRANSLATION_FOUND"
	case FAILED_NO_EXTRACTED:
		return "FAILED_NO_EXTRACTED"
	case FAILED_NO_INFERENCE:
		return "FAILED_NO_INFERENCE"
	case FAILED_EXECUTION_TIMEOUT:
		return "FAILED_EXECUTION_TIMEOUT"
	case TRANSLATED:
		return "TRANSLATED"
	default:
		return fmt.Sprintf("Unknown Status (%d)", s)
	}
}

func ParseStatus(status string) Status {
	switch status {
	case "PENDING":
		return PENDING
	case "PROCESSING":
		return PROCESSING
	case "TRANSLATED":
		return TRANSLATED
	case "SUCCESS":
		return SUCCESS
	case "FAILED":
		return FAILED
	case "FAILED_NO_INFERENCE":
		return FAILED_NO_INFERENCE
	case "FAILED_NO_EXTRACTED":
		return FAILED_NO_EXTRACTED
	case "FAILED_EXECUTION":
		return FAILED_EXECUTION
	case "FAILED_VERIFICATION":
		return FAILED_VERIFICATION
	case "FAILED_EXECUTION_TIMEOUT":
		return FAILED_EXECUTION_TIMEOUT
	case "SKIPPED_PARENT_FAILED":
		return SKIPPED_PARENT_FAILED
	case "SKIPPED_TRANSLATION_FOUND":
		return SKIPPED_TRANSLATION_FOUND
	case "TRANSLATION_FOUND":
		return TRANSLATION_FOUND
	default:
		panic("Unknown status")
	}
}

type TranslationPair struct {
	InputLanguage  string
	TargetLanguage string
}

// TranslationEdge struct with thread-safe Status
type TranslationEdge struct {
	Id                         int
	PromptTemplate             string
	Prompt                     string
	TranslationId              string
	InputLanguage              string
	TargetLanguage             string
	Level                      int
	Success                    bool
	InferenceOutput            string
	ExecutionOutput            string
	SourceCode                 string
	ExtractedSourceCode        string
	SuggestedTargetSignature   string
	RegexTemplate              string
	ModelName                  string
	ParentEdge                 *TranslationEdge
	FuzzyTests                 []FuzzyTest
	UnitTests                  []UnitTest
	WallClockInferenceTime     time.Duration
	WallClockTestExecutionTime time.Duration
	UsedMemoization            bool
	UsedInferenceCache         bool
	ExtraPromptData            string

	status          Status      // Status property
	StatusMutex     *sync.Mutex // Mutex for thread safety
	ProcessingMutex *sync.Mutex
}

// Setter method for Status
func (e *TranslationEdge) SetStatus(newStatus Status) {
	e.StatusMutex.Lock()
	defer e.StatusMutex.Unlock()
	e.status = newStatus
}

func (e *TranslationEdge) UpdatePendingStatus(newStatus Status) {
	e.StatusMutex.Lock()
	defer e.StatusMutex.Unlock()
	if e.status == PENDING || e.status == PROCESSING {
		e.status = newStatus
	}
}

// Getter method for Status
func (e *TranslationEdge) GetStatus() Status {
	e.StatusMutex.Lock()
	defer e.StatusMutex.Unlock()
	return e.status
}

// Implement the Stringer interface for TranslationEdge
func (e TranslationEdge) String() string {
	return fmt.Sprintf("%s -> %s", e.InputLanguage, e.TargetLanguage)
}

func (te *TranslationEdge) GetDepth() int {
	counter := 0
	currentEdge := te

	for currentEdge != nil {
		currentEdge = currentEdge.ParentEdge
		counter++
	}

	return counter
}

func (te *TranslationEdge) IsDirectPathToTarget(targetLanguage string) bool {
	directPathToTarget := true
	currentEdge := te

	for currentEdge != nil {

		if currentEdge.TargetLanguage == targetLanguage {
			directPathToTarget = false
			break
		}

		currentEdge = currentEdge.ParentEdge
	}

	return directPathToTarget
}

func GetFileExtensionsMap() map[string]string {
	fileExtensionsMap := map[string]string{
		"Python":       ".py",
		"JavaScript":   ".js",
		"Java":         ".java",
		"C":            ".c",
		"C++":          ".cpp",
		"C#":           ".cs",
		"Go":           ".go",
		"Ruby":         ".rb",
		"PHP":          ".php",
		"Swift":        ".swift",
		"Kotlin":       ".kt",
		"TypeScript":   ".ts",
		"HTML":         ".html",
		"CSS":          ".css",
		"R":            ".R",
		"MATLAB":       ".m",
		"Shell Script": ".sh",
		"Perl":         ".pl",
		"Scala":        ".scala",
		"Objective-C":  ".m",
		"Rust":         ".rs",
		"Haskell":      ".hs",
		"Lua":          ".lua",
		"Dart":         ".dart",
		"Elixir":       ".ex",
		"Erlang":       ".erl",
		"F#":           ".fs",
		"Fortran":      ".f90",
		"Groovy":       ".groovy",
		"Pascal":       ".pas",
		"VHDL":         ".vhd",
		"Verilog":      ".v",
		"COBOL":        ".cob",
		"Assembly":     ".asm",
		"Tcl":          ".tcl",
		"Ada":          ".adb",
		"Prolog":       ".pl",
		"Julia":        ".jl",
		"Visual Basic": ".vb",
		"SQL":          ".sql",
	}

	return fileExtensionsMap
}

func GetExecutorForLanguageMap() map[string]string {
	return ConfigStore.ExecutionContainers
}
