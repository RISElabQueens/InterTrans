from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ResponseStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PENDING: _ClassVar[ResponseStatus]
    PROCESSING: _ClassVar[ResponseStatus]
    FAILED: _ClassVar[ResponseStatus]
    DONE: _ClassVar[ResponseStatus]
    TRANSLATION_FOUND: _ClassVar[ResponseStatus]
    SKIPPED_PARENT_FAILED: _ClassVar[ResponseStatus]
    SKIPPED_TRANSLATION_FOUND: _ClassVar[ResponseStatus]
PENDING: ResponseStatus
PROCESSING: ResponseStatus
FAILED: ResponseStatus
DONE: ResponseStatus
TRANSLATION_FOUND: ResponseStatus
SKIPPED_PARENT_FAILED: ResponseStatus
SKIPPED_TRANSLATION_FOUND: ResponseStatus

class TestSuite(_message.Message):
    __slots__ = ("fuzzy_suite", "unit_test_suite")
    FUZZY_SUITE_FIELD_NUMBER: _ClassVar[int]
    UNIT_TEST_SUITE_FIELD_NUMBER: _ClassVar[int]
    fuzzy_suite: _containers.RepeatedCompositeFieldContainer[FuzzyTestCase]
    unit_test_suite: _containers.RepeatedCompositeFieldContainer[UnitTestCase]
    def __init__(self, fuzzy_suite: _Optional[_Iterable[_Union[FuzzyTestCase, _Mapping]]] = ..., unit_test_suite: _Optional[_Iterable[_Union[UnitTestCase, _Mapping]]] = ...) -> None: ...

class FuzzyTestCase(_message.Message):
    __slots__ = ("stdin_input", "expected_output")
    STDIN_INPUT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    stdin_input: str
    expected_output: str
    def __init__(self, stdin_input: _Optional[str] = ..., expected_output: _Optional[str] = ...) -> None: ...

class ResponseFuzzyTestCase(_message.Message):
    __slots__ = ("stdin_input", "expected_output", "actual_output", "passed", "executed_code")
    STDIN_INPUT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_CODE_FIELD_NUMBER: _ClassVar[int]
    stdin_input: str
    expected_output: str
    actual_output: str
    passed: bool
    executed_code: str
    def __init__(self, stdin_input: _Optional[str] = ..., expected_output: _Optional[str] = ..., actual_output: _Optional[str] = ..., passed: bool = ..., executed_code: _Optional[str] = ...) -> None: ...

class ResponseUnitTestCase(_message.Message):
    __slots__ = ("source_code", "actual_output", "passed", "executed_code")
    SOURCE_CODE_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_CODE_FIELD_NUMBER: _ClassVar[int]
    source_code: str
    actual_output: str
    passed: bool
    executed_code: str
    def __init__(self, source_code: _Optional[str] = ..., actual_output: _Optional[str] = ..., passed: bool = ..., executed_code: _Optional[str] = ...) -> None: ...

class UnitTestCase(_message.Message):
    __slots__ = ("language", "test_case", "imports")
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    TEST_CASE_FIELD_NUMBER: _ClassVar[int]
    IMPORTS_FIELD_NUMBER: _ClassVar[int]
    language: str
    test_case: str
    imports: str
    def __init__(self, language: _Optional[str] = ..., test_case: _Optional[str] = ..., imports: _Optional[str] = ...) -> None: ...

class TargetSignature(_message.Message):
    __slots__ = ("language", "signature")
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    language: str
    signature: str
    def __init__(self, language: _Optional[str] = ..., signature: _Optional[str] = ...) -> None: ...

class TranslationRequest(_message.Message):
    __slots__ = ("id", "seed_language", "target_language", "seed_code", "test_suite", "used_languages", "prompt_template_name", "target_signatures", "regex_template_name", "model_name", "extra_prompt_data")
    ID_FIELD_NUMBER: _ClassVar[int]
    SEED_LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    TARGET_LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    SEED_CODE_FIELD_NUMBER: _ClassVar[int]
    TEST_SUITE_FIELD_NUMBER: _ClassVar[int]
    USED_LANGUAGES_FIELD_NUMBER: _ClassVar[int]
    PROMPT_TEMPLATE_NAME_FIELD_NUMBER: _ClassVar[int]
    TARGET_SIGNATURES_FIELD_NUMBER: _ClassVar[int]
    REGEX_TEMPLATE_NAME_FIELD_NUMBER: _ClassVar[int]
    MODEL_NAME_FIELD_NUMBER: _ClassVar[int]
    EXTRA_PROMPT_DATA_FIELD_NUMBER: _ClassVar[int]
    id: str
    seed_language: str
    target_language: str
    seed_code: str
    test_suite: TestSuite
    used_languages: _containers.RepeatedScalarFieldContainer[str]
    prompt_template_name: str
    target_signatures: _containers.RepeatedCompositeFieldContainer[TargetSignature]
    regex_template_name: str
    model_name: str
    extra_prompt_data: str
    def __init__(self, id: _Optional[str] = ..., seed_language: _Optional[str] = ..., target_language: _Optional[str] = ..., seed_code: _Optional[str] = ..., test_suite: _Optional[_Union[TestSuite, _Mapping]] = ..., used_languages: _Optional[_Iterable[str]] = ..., prompt_template_name: _Optional[str] = ..., target_signatures: _Optional[_Iterable[_Union[TargetSignature, _Mapping]]] = ..., regex_template_name: _Optional[str] = ..., model_name: _Optional[str] = ..., extra_prompt_data: _Optional[str] = ...) -> None: ...

class ResponseTranslationEdge(_message.Message):
    __slots__ = ("prompt_template", "prompt", "translation_id", "input_language", "target_language", "level", "success", "inference_output", "execution_output", "source_code", "extracted_source_code", "parent_edge_id", "status", "fuzzy_tests", "unit_tests", "edge_id", "wallTimeInference", "wallTimeTestExecution", "usedMemoization", "usedInferenceCache")
    PROMPT_TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    TRANSLATION_ID_FIELD_NUMBER: _ClassVar[int]
    INPUT_LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    TARGET_LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    INFERENCE_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_CODE_FIELD_NUMBER: _ClassVar[int]
    EXTRACTED_SOURCE_CODE_FIELD_NUMBER: _ClassVar[int]
    PARENT_EDGE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    FUZZY_TESTS_FIELD_NUMBER: _ClassVar[int]
    UNIT_TESTS_FIELD_NUMBER: _ClassVar[int]
    EDGE_ID_FIELD_NUMBER: _ClassVar[int]
    WALLTIMEINFERENCE_FIELD_NUMBER: _ClassVar[int]
    WALLTIMETESTEXECUTION_FIELD_NUMBER: _ClassVar[int]
    USEDMEMOIZATION_FIELD_NUMBER: _ClassVar[int]
    USEDINFERENCECACHE_FIELD_NUMBER: _ClassVar[int]
    prompt_template: str
    prompt: str
    translation_id: str
    input_language: str
    target_language: str
    level: int
    success: bool
    inference_output: str
    execution_output: str
    source_code: str
    extracted_source_code: str
    parent_edge_id: int
    status: str
    fuzzy_tests: _containers.RepeatedCompositeFieldContainer[ResponseFuzzyTestCase]
    unit_tests: _containers.RepeatedCompositeFieldContainer[ResponseUnitTestCase]
    edge_id: int
    wallTimeInference: int
    wallTimeTestExecution: int
    usedMemoization: bool
    usedInferenceCache: bool
    def __init__(self, prompt_template: _Optional[str] = ..., prompt: _Optional[str] = ..., translation_id: _Optional[str] = ..., input_language: _Optional[str] = ..., target_language: _Optional[str] = ..., level: _Optional[int] = ..., success: bool = ..., inference_output: _Optional[str] = ..., execution_output: _Optional[str] = ..., source_code: _Optional[str] = ..., extracted_source_code: _Optional[str] = ..., parent_edge_id: _Optional[int] = ..., status: _Optional[str] = ..., fuzzy_tests: _Optional[_Iterable[_Union[ResponseFuzzyTestCase, _Mapping]]] = ..., unit_tests: _Optional[_Iterable[_Union[ResponseUnitTestCase, _Mapping]]] = ..., edge_id: _Optional[int] = ..., wallTimeInference: _Optional[int] = ..., wallTimeTestExecution: _Optional[int] = ..., usedMemoization: bool = ..., usedInferenceCache: bool = ...) -> None: ...

class ResponseTranslationPath(_message.Message):
    __slots__ = ("translation_edges", "edge_index_memoized")
    TRANSLATION_EDGES_FIELD_NUMBER: _ClassVar[int]
    EDGE_INDEX_MEMOIZED_FIELD_NUMBER: _ClassVar[int]
    translation_edges: _containers.RepeatedCompositeFieldContainer[ResponseTranslationEdge]
    edge_index_memoized: _containers.RepeatedScalarFieldContainer[bool]
    def __init__(self, translation_edges: _Optional[_Iterable[_Union[ResponseTranslationEdge, _Mapping]]] = ..., edge_index_memoized: _Optional[_Iterable[bool]] = ...) -> None: ...

class TranslationResponse(_message.Message):
    __slots__ = ("translation_request", "paths")
    TRANSLATION_REQUEST_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    translation_request: TranslationRequest
    paths: _containers.RepeatedCompositeFieldContainer[ResponseTranslationPath]
    def __init__(self, translation_request: _Optional[_Union[TranslationRequest, _Mapping]] = ..., paths: _Optional[_Iterable[_Union[ResponseTranslationPath, _Mapping]]] = ...) -> None: ...

class BatchTranslationRequest(_message.Message):
    __slots__ = ("translation_requests", "id", "file_base_name", "file_save_path")
    TRANSLATION_REQUESTS_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    FILE_BASE_NAME_FIELD_NUMBER: _ClassVar[int]
    FILE_SAVE_PATH_FIELD_NUMBER: _ClassVar[int]
    translation_requests: _containers.RepeatedCompositeFieldContainer[TranslationRequest]
    id: str
    file_base_name: str
    file_save_path: str
    def __init__(self, translation_requests: _Optional[_Iterable[_Union[TranslationRequest, _Mapping]]] = ..., id: _Optional[str] = ..., file_base_name: _Optional[str] = ..., file_save_path: _Optional[str] = ...) -> None: ...

class BatchTranslationResponse(_message.Message):
    __slots__ = ("translation_responses", "request_id", "returnedToDisk")
    TRANSLATION_RESPONSES_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    RETURNEDTODISK_FIELD_NUMBER: _ClassVar[int]
    translation_responses: _containers.RepeatedCompositeFieldContainer[TranslationResponse]
    request_id: str
    returnedToDisk: bool
    def __init__(self, translation_responses: _Optional[_Iterable[_Union[TranslationResponse, _Mapping]]] = ..., request_id: _Optional[str] = ..., returnedToDisk: bool = ...) -> None: ...

class StartEndpointRequest(_message.Message):
    __slots__ = ("model_name", "gpu_id", "port", "seed", "api_token", "lora_path")
    MODEL_NAME_FIELD_NUMBER: _ClassVar[int]
    GPU_ID_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    API_TOKEN_FIELD_NUMBER: _ClassVar[int]
    LORA_PATH_FIELD_NUMBER: _ClassVar[int]
    model_name: str
    gpu_id: str
    port: str
    seed: int
    api_token: str
    lora_path: str
    def __init__(self, model_name: _Optional[str] = ..., gpu_id: _Optional[str] = ..., port: _Optional[str] = ..., seed: _Optional[int] = ..., api_token: _Optional[str] = ..., lora_path: _Optional[str] = ...) -> None: ...

class StopEndpointRequest(_message.Message):
    __slots__ = ("launch_id",)
    LAUNCH_ID_FIELD_NUMBER: _ClassVar[int]
    launch_id: int
    def __init__(self, launch_id: _Optional[int] = ...) -> None: ...

class LaunchResponse(_message.Message):
    __slots__ = ("launch_id",)
    LAUNCH_ID_FIELD_NUMBER: _ClassVar[int]
    launch_id: int
    def __init__(self, launch_id: _Optional[int] = ...) -> None: ...

class VerificationRequest(_message.Message):
    __slots__ = ("id", "test_suite", "inferenceOutput", "targetLanguage", "sourceLanguage")
    ID_FIELD_NUMBER: _ClassVar[int]
    TEST_SUITE_FIELD_NUMBER: _ClassVar[int]
    INFERENCEOUTPUT_FIELD_NUMBER: _ClassVar[int]
    TARGETLANGUAGE_FIELD_NUMBER: _ClassVar[int]
    SOURCELANGUAGE_FIELD_NUMBER: _ClassVar[int]
    id: str
    test_suite: TestSuite
    inferenceOutput: str
    targetLanguage: str
    sourceLanguage: str
    def __init__(self, id: _Optional[str] = ..., test_suite: _Optional[_Union[TestSuite, _Mapping]] = ..., inferenceOutput: _Optional[str] = ..., targetLanguage: _Optional[str] = ..., sourceLanguage: _Optional[str] = ...) -> None: ...

class VerificationResponse(_message.Message):
    __slots__ = ("verification_request", "fuzzy_tests", "unit_tests", "status")
    VERIFICATION_REQUEST_FIELD_NUMBER: _ClassVar[int]
    FUZZY_TESTS_FIELD_NUMBER: _ClassVar[int]
    UNIT_TESTS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    verification_request: VerificationRequest
    fuzzy_tests: _containers.RepeatedCompositeFieldContainer[ResponseFuzzyTestCase]
    unit_tests: _containers.RepeatedCompositeFieldContainer[ResponseUnitTestCase]
    status: str
    def __init__(self, verification_request: _Optional[_Union[VerificationRequest, _Mapping]] = ..., fuzzy_tests: _Optional[_Iterable[_Union[ResponseFuzzyTestCase, _Mapping]]] = ..., unit_tests: _Optional[_Iterable[_Union[ResponseUnitTestCase, _Mapping]]] = ..., status: _Optional[str] = ...) -> None: ...

class BatchVerificationRequest(_message.Message):
    __slots__ = ("verification_requests", "id")
    VERIFICATION_REQUESTS_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    verification_requests: _containers.RepeatedCompositeFieldContainer[VerificationRequest]
    id: str
    def __init__(self, verification_requests: _Optional[_Iterable[_Union[VerificationRequest, _Mapping]]] = ..., id: _Optional[str] = ...) -> None: ...

class BatchVerificationResponse(_message.Message):
    __slots__ = ("verification_requests", "verification_responses")
    VERIFICATION_REQUESTS_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_RESPONSES_FIELD_NUMBER: _ClassVar[int]
    verification_requests: VerificationRequest
    verification_responses: _containers.RepeatedCompositeFieldContainer[VerificationResponse]
    def __init__(self, verification_requests: _Optional[_Union[VerificationRequest, _Mapping]] = ..., verification_responses: _Optional[_Iterable[_Union[VerificationResponse, _Mapping]]] = ...) -> None: ...
