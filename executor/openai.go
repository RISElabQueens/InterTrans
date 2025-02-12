package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/anonymoussubmission/codetransengine/common"
)

type ChatCompletionRequest struct {
	Model             string                 `json:"model"`
	Messages          []Message              `json:"messages"`
	ExtraFields       map[string]interface{} `json:"extra_fields,omitempty"`
	MaxTokens         int                    `json:"max_tokens,omitempty"`
	SkipSpecialTokens bool                   `json:"skip_special_tokens,omitempty"`
	Temperature       float32                `json:"temperature,omitempty"`
	TopP              float32                `json:"top_p,omitempty"`
	TopK              int                    `json:"top_k,omitempty"`
	Seed              int                    `json:"seed,omitempty"`
	N                 int                    `json:"n,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int      `json:"created"`
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type RoundRobinApiCaller struct {
	currentIndex int
	baseUrls     []string
}

var roundRobinApiCaller *RoundRobinApiCaller
var roundRobinMutex sync.Mutex

func GetRoundRobin() *RoundRobinApiCaller {
	roundRobinMutex.Lock()
	if roundRobinApiCaller == nil {
		roundRobinApiCaller = &RoundRobinApiCaller{
			currentIndex: 0,
			baseUrls:     common.ConfigStore.InferenceApiBaseUrls,
		}
	}
	roundRobinMutex.Unlock()
	return roundRobinApiCaller
}

func (roundRobinApiCaller *RoundRobinApiCaller) GetNext() string {
	roundRobinMutex.Lock()
	returned := roundRobinApiCaller.baseUrls[roundRobinApiCaller.currentIndex]
	roundRobinApiCaller.currentIndex++

	if roundRobinApiCaller.currentIndex == len(roundRobinApiCaller.baseUrls) {
		roundRobinApiCaller.currentIndex = 0
	}
	roundRobinMutex.Unlock()
	return returned
}

func GetChatCompletion(apiKey, message string, modelName string) (string, error) {
	var requestBody ChatCompletionRequest
	if common.ConfigStore.Seed != -1 {
		requestBody = ChatCompletionRequest{
			Model: modelName,
			Messages: []Message{
				{
					Role:    "user",
					Content: message,
				},
			},
			MaxTokens:   common.ConfigStore.MaxGeneratedTokens,
			Temperature: common.ConfigStore.Temperature,
			TopP:        common.ConfigStore.TopP,
		}

		if common.ConfigStore.InferenceBackend == "vllm" {
			requestBody.SkipSpecialTokens = true
			requestBody.TopK = common.ConfigStore.TopK
			requestBody.Seed = common.ConfigStore.Seed
		}

	} else {
		requestBody = ChatCompletionRequest{
			Model: modelName,
			Messages: []Message{
				{
					Role:    "user",
					Content: message,
				},
			},
			MaxTokens:   common.ConfigStore.MaxGeneratedTokens,
			Temperature: common.ConfigStore.Temperature,
			TopP:        common.ConfigStore.TopP,
		}

		if common.ConfigStore.InferenceBackend == "vllm" {
			requestBody.SkipSpecialTokens = true
			requestBody.TopK = common.ConfigStore.TopK
		}
	}

	baseUrl := GetRoundRobin().GetNext()
	openaiURL := baseUrl + "/chat/completions"

	requestBodyBytes, err := json.Marshal(requestBody)

	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest("POST", openaiURL, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform HTTP request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		error_msg := fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
		fmt.Println(error_msg)
		return "", error_msg
	}

	var completionResponse ChatCompletionResponse
	if err := json.Unmarshal(body, &completionResponse); err != nil {
		return "", fmt.Errorf("failed to unmarshal response body: %v", err)
	}

	if len(completionResponse.Choices) > 0 {
		response := completionResponse.Choices[0].Message.Content
		return response, nil
	}

	return "", fmt.Errorf("no choices found in response")
}
