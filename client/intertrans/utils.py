import grpc
import intertrans.protos_pb2_grpc as ptgrpc
import intertrans.protos_pb2 as ptpb
import time

def json_response_to_grpc(translation_responses):
    new_response = ptpb.BatchTranslationResponse()

    for response in translation_responses["translation_responses"]:

        translation_response = ptpb.TranslationResponse()

        translation_request=ptpb.TranslationRequest(
                seed_language=response['translation_request']['seed_language'],
                target_language=response['translation_request']['target_language'],
                id=response['translation_request']['id'],
                seed_code=response['translation_request']['seed_code']
        )


        for path in response['paths']:
            translation_path = ptpb.ResponseTranslationPath

            for index, edge in enumerate(path["translation_edges"]):

                translation_edge = ptpb.ResponseTranslationEdge(
                    prompt_template=edge.get("prompt_template", None),
                    prompt=edge.get("prompt", None),
                    translation_id=edge.get("translation_id", None),
                    input_language=edge.get("input_language", None),
                    target_language=edge.get("target_language", None),
                    level=edge.get("level", None),
                    success=edge.get("success", None),
                    inference_output=edge.get('inference_output', None),
                    execution_output=edge.get('execution_output', None),
                    source_code=edge.get('source_code', None),
                    extracted_source_code=edge.get('extracted_source_code', None),
                    parent_edge_id=int(edge.get("parent_edge_id", 0)),
                    status=edge.get("status", None),
                    edge_id=int(edge.get("edge_id", 0)),
                    wallTimeInference=int(edge.get("wallTimeInference", 0)),
                    wallTimeTestExecution=int(edge.get("wallTimeTestExecution", 0)),
                    usedMemoization=edge.get("usedMemoization", False),
                    usedInferenceCache=edge.get("usedInferenceCache", False)
                )

                if "fuzzy_tests" in edge:
                    tests = edge['fuzzy_tests']
                    test_case = ptpb.FuzzyTestCase()

                    for test in tests:
                        test_case = ptpb.FuzzyTestCase(
                            input=test['input'],
                            expected_output=test['expected_output'],
                            actual_output=test['actual_output'],
                            status=test['status']
                        )

                        translation_edge.fuzzy_tests.extend([test_case])
                    
                elif 'unit_tests' in edge:
                    tests = edge['unit_tests']
                    test_case = ptpb.UnitTestCase()

                    for test in tests:
                        test_case = ptpb.UnitTestCase(
                            language=test['language'],
                            test_case=test['test_case'],
                            imports=test['imports'],
                            status=test['status'],
                        )

                        translation_edge.unit_tests.extend([test_case])
                else:
                    raise Exception("Should not happen")

                translation_edge.tests.extend(test_case)

            translation_request.paths.append(translation_path)

        translation_response.translation_request = translation_request
            
    return ptpb.BatchTranslationResponse(translation_responses=translation_response)
    

def submit_request(batch_request, grpc_channel_address):
    options = [
    ('grpc.max_send_message_length', 1000 * 1024 * 1024 * 2), 
    ('grpc.max_receive_message_length', 1000 * 1024 * 1024 * 2) 
    ]

    with grpc.insecure_channel(grpc_channel_address, options=options) as channel:
        stub = ptgrpc.TranslationServiceStub(channel)
        response = stub.BatchTranslate(batch_request)

    return response

def submit_request_cak(batch_request, grpc_channel_address):
    options = [
    ('grpc.max_send_message_length', 1000 * 1024 * 1024 * 2), 
    ('grpc.max_receive_message_length', 1000 * 1024 * 1024 * 2) 
    ]

    with grpc.insecure_channel(grpc_channel_address, options=options) as channel:
        stub = ptgrpc.TranslationServiceStub(channel)
        response = stub.BatchTranslateCAK(batch_request)

    return response

def submit_request_execute(batch_request, grpc_channel_address):
    options = [
    ('grpc.max_send_message_length', 1000 * 1024 * 1024 * 2), 
    ('grpc.max_receive_message_length', 1000 * 1024 * 1024 * 2) 
    ]

    with grpc.insecure_channel(grpc_channel_address, options=options) as channel:
        stub = ptgrpc.TranslationServiceStub(channel)
        response = stub.BatchRunVerification(batch_request)

    return response

def submit_infra_request(infra_request, grpc_channel_address):
    options = [
    ('grpc.max_send_message_length', 1000 * 1024 * 1024),  # 1000 MB
    ('grpc.max_receive_message_length', 1000 * 1024 * 1024)  # 1000 MB
    ]

    with grpc.insecure_channel(grpc_channel_address, options=options) as channel:
        stub = ptgrpc.InfrastructureServiceStub(channel)
        response = stub.BatchTranslate(batch_request)

    return response

def stop_inference_endpoints(ids, grpc_channel_address):
    results = []
    options = [
    ('grpc.max_send_message_length', 1000 * 1024 * 1024),  # 1000 MB
    ('grpc.max_receive_message_length', 1000 * 1024 * 1024)  # 1000 MB
    ]

    with grpc.insecure_channel(grpc_channel_address, options=options) as channel:
        stub = ptgrpc.InfrastructureServiceStub(channel)

        for launch_id in ids:
            request = ptpb.StopEndpointRequest()
            request.launch_id = launch_id
            response = stub.StopInferenceEndpoint(request)
            results.append(response)

    print("Sleeping for 60 seconds to allow the endpoints to shutdown")
    time.sleep(60)

    return results

def launch_inference_endpoints(model, grpc_channel_address, lora_path=None):
    options = [
    ('grpc.max_send_message_length', 1000 * 1024 * 1024),  
    ('grpc.max_receive_message_length', 1000 * 1024 * 1024) 
    ]

    #Launch four instances
    one = ptpb.StartEndpointRequest()
    one.model_name = model
    one.gpu_id = "3"
    one.port = "8000"
    one.seed = 51291074
    one.api_token = "token"
    one.lora_path = lora_path

    two = ptpb.StartEndpointRequest()
    two.model_name = model
    two.gpu_id = "4"
    two.port = "8001"
    two.seed = 51291074
    two.api_token = "token"
    two.lora_path = lora_path

    three = ptpb.StartEndpointRequest()
    three.model_name = model
    three.gpu_id = "5"
    three.port = "8002"
    three.seed = 51291074
    three.api_token = "token"
    three.lora_path = lora_path

    four = ptpb.StartEndpointRequest()
    four.model_name = model
    four.gpu_id = "6"
    four.port = "8003"
    four.seed = 51291074
    four.api_token = "token"

    # five = ptpb.StartEndpointRequest()
    # five.model_name = model
    # five.gpu_id = "5"
    # five.port = "8004"
    # five.seed = 51291074
    # five.api_token = "token"

    # six = ptpb.StartEndpointRequest()
    # six.model_name = model
    # six.gpu_id = "6"
    # six.port = "8005"
    # six.seed = 51291074
    # six.api_token = "token"

    with grpc.insecure_channel(grpc_channel_address, options=options) as channel:
        stub = ptgrpc.InfrastructureServiceStub(channel)

        response_one = stub.LaunchInferenceEndpoint(one)
        response_two = stub.LaunchInferenceEndpoint(two)
        response_three = stub.LaunchInferenceEndpoint(three)
        response_four = stub.LaunchInferenceEndpoint(four)
        # response_five = stub.LaunchInferenceEndpoint(five)
        # response_six = stub.LaunchInferenceEndpoint(six)

    print("Sleeping for 60 seconds to allow the endpoints to be ready")
    time.sleep(60)

    return [
        response_one.launch_id,
        response_two.launch_id,
        response_three.launch_id,
        response_four.launch_id,
        # response_five.launch_id,
        # response_six.launch_id
    ]