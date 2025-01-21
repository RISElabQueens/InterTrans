---
title: Server Configuration
description: A reference page in my new Starlight docs site.
---

Below is an example of a complete configuration file.

```yaml
numExecutionWorkers: 4
numInferenceWorkers: 2
useComputeEfficientMode: true
serverAddress: 127.0.0.1
serverPort: 50051
earlyStop : true
verifyIntermediateTranslations : false
expansionIntermediaryNodes: 3
applyRegexInferenceOnly: true
useTranscoderTestFormat: false
useInferenceCache: false
useResponseCache: false
useExecutionCache: true
cacheDatabasePath: ../data/cache/
maxGeneratedTokens: 4096
temperature: 0.7
top-p: 0.95
top-k: 10
inferenceSeed: -1
regexTemplates:
  temperature: (?s)\x60\x60\x60(?:(?:javascript|java|cpp|csharp|python|script|rust|c|go|C\+\+|Javascript|JavaScript|Java|Python|C#|C|Rust|Script|Go))?(.+)\x60\x60\x60
inferenceApiBaseUrls:
  - http://localhost:8000/v1
inferenceApiToken: token
executionContainers:
  "Python":     "./singularity/img/python3.sif"
  "JavaScript": "./singularity/img/node.sif"
  "Java":       "./singularity/img/java.sif"
  "C++":        "./singularity/img/cpp-clang.sif"
  "Go":         "./singularity/img/golang.sif"
  "Rust":         "./singularity/img/rust.sif"
promptTemplates:
  prompt_codenet: |
    @@ Instruction
    You are a skilled software developer proficient in multiple programming languages. Your task is to re-write the input source code. Below is the input source code written in {input_lang} that you should re-write into {target_lang} programming language. You must respond with the {target_lang} output code only. 

    Source code:
    ```
    {input_code}
    ```

    @@ Response
```

## Fields

### numExecutionWorkers: integer
Controls the number of Singularity containers that can run concurrently to execute the translated code. In effect only when ```useComputeEfficientMode: false```
### numInferenceWorkers: integer
Controls the number of concurrent inference request on the OpenAI API compatible server (e.g. vLLM). ```useComputeEfficientMode: false```
### useComputeEfficientMode: boolean
When set to ``true``, it disables concurrency in InterTrans. This means that translations inside ToCT are processed sequentially and when there is an error in the path (e.g. can't extract source code, or inference failed) the algorithm moves on to the next path. When a translation is found, the algorithm returns immediately. Setting this to ``false`` enables higher throughput and faster translations, at the expense of possibly additional computations. When set to ``false`` finding a translation in a path does best-effort stopping computations in other paths.
### serverAddress: ip address
gRPC endpoint address for the server
### serverPort: ip address
gRPC port for the server
### earlyStop: boolean
In translations where there are multiple test cases to be executed to assess the accuracy of the translation. When ``true`` if one of the test cases fails, the algorithm returns and does not execute the other test cases. ``false`` would execute all the test cases regardless if the previous one failed. Useful to disable when information about the status of each test case is necessary.
### expansionIntermediaryNodes: integer
This is the maximum number of intermediate translations in other programming languages inside a path in the ToCT algorithm. Setting this value to ```1``` is the same as performing a Direct Translation. A value between ```2``` and ```4``` is recommended.
### applyRegexInferenceOnly: boolean
```true``` if the regular expression to extract the source code is only applied to the new tokens resulting from inference (ignoring tokens from the prompt)
### useTranscoderTestFormat: boolean
Enables processing for TransCoder test cases. Should be set to ```false``` if not using test cases similar to TransCoder test cases.
### useInferenceCache: boolean
When an inference is to be performed for an edge, return a cached version if available if set to ```true```. This should be set to false when looking to take advantage of randomness (such as performing Direct Translation @ K).
### useResponseCache: boolean
When set to ```true``` it allows to cache the results of a translation request. This is useful to resume experiments or add new samples, as previous samples would not have to be recomputed (if other options remain unchanged)
### useExecutionCache: boolean
If ```true```, whenever the LLM generates a program that was previously seen, it returns the results of the previous execution for such program instead of executing it again
### cacheDatabasePath: boolean
Path for the cache database
### temperature: float
Temperature for sampling during inference
### top-p: float
Top-P sampling. A value of ```-1``` disables this feature.
### top-k: integer
Top-K sampling. A value of ```-1``` disables this feature.
### inferenceSeed: integer
Seed to use for the pseudorandom generator of vLLM. This ensures that runs are replicable when using sampling during inference.
### regexTemplates: list
List of regex to use for extracting source code, compliant with Go regex library.
### inferenceApiBaseUrls: list
OpenAPI Compatible Server endpoints to send the inference requests. If more than one endpoint is specified, the engine will submit the requests in round robin to spread the load.
### inferenceApiToken
Token for the OpenAPI endpoint
### executionContainers: dict
Each key in the dictionary corresponds to a target programming language enabled in InterTrans engine. The value of the dictionary is the ```path``` containing the .sif file (Singularity container) capable of executing code for such language.
### promptTemplates: list
List of prompt templates to be used during the ToCT algorithm. Please see the section [Prompt templates](/reference/prompt) to understand supported parameters for the prompt.
### inferenceBackend: enum (optional)
If this field is not set, the inference backend would default to an OpenAI compatible API. If set to ```vllm`` it would enable vLLM-specific parameters in the OpenAI API request to vLLM.

