import pandas as pd
from google.protobuf.json_format import MessageToJson
import json
import os

def response_to_pandas(grpc_batch_response):
    all_edges = []

    for response in grpc_batch_response.translation_responses:
        seed_language = response.translation_request.seed_language
        target_language = response.translation_request.target_language
        id_request = response.translation_request.id
        seed_code = response.translation_request.seed_code

        for path in response.paths:

            for index, edge in enumerate(path.translation_edges):
                is_memoized = False

                if index in path.edge_index_memoized:
                    is_memoized = True

                obj = {}
                obj["seed_language"] = seed_language
                obj["request_target_language"] = target_language
                obj["request_id"] = id_request
                obj["input_languages"] = edge.input_language
                obj["target_languages"] = edge.target_language
                obj["level"] = edge.level
                obj["edge_id"] = int(edge.edge_id)
                obj["parent_edge_id"] = int(edge.parent_edge_id)
                obj["status"] = edge.status
                obj["memoized"] = is_memoized
                obj["failed_timeout"] = False
                obj["extracted_code"] = edge.extracted_source_code
                obj["inference_output"] = edge.inference_output

                if hasattr(edge, 'fuzzy_tests'):
                    tests = edge.fuzzy_tests
                elif hasattr(edge, 'unit_tests'):
                    tests = edge.unit_tests
                else:
                    tests = []

                for itest, test in enumerate(tests):
                    obj[f"test_{itest}_input"] = test.stdin_input
                    if test.actual_output is not None:
                        obj[f"test_{itest}_actual_output"] = test.actual_output
                        if "CMD_TIMEOUT_KILLED" in test.actual_output:
                            obj["failed_timeout"] = True

                all_edges.append(obj)

    return pd.DataFrame(all_edges)

def to_pandas(grpc_batch_response):
    all_edges = []

    for response in grpc_batch_response["translation_responses"]:
        seed_language = response['translation_request']['seed_language']
        target_language = response['translation_request']['target_language']
        id_request = response['translation_request']['id']
        seed_code = response['translation_request']['seed_code']

        for path in response['paths']:

            for index, edge in enumerate(path["translation_edges"]):
                is_memoized = False

                if index in path['edge_index_memoized']:
                    is_memoized = True

                obj = {}
                obj["seed_language"] = seed_language
                obj["request_target_language"] = target_language
                obj["request_id"] = id_request
                obj["input_languages"] = edge["input_language"]
                obj["target_languages"] = edge["target_language"]
                obj["level"] = edge["level"]
                obj["edge_id"] = int(edge["edge_id"])
                obj["parent_edge_id"] = int(edge["parent_edge_id"])
                obj["status"] = edge["status"]
                obj["memoized"] = is_memoized
                obj["failed_timeout"] = False
                obj["extracted_code"] = edge.get('extracted_source_code', "")
                obj["inference_output"] = edge.get('inference_output', "")

                if "fuzzy_tests" in edge.keys():
                    tests = edge['fuzzy_tests']
                elif 'unit_tests'  in edge.keys():
                    tests = edge['unit_tests']
                else:
                    tests = []

                for itest, test in enumerate(tests):
                    obj[f"test_{itest}_input"] = test.get('stdin_input', "")
                    if test.get('actual_output', None) is not None:
                        obj[f"test_{itest}_actual_output"] = test['actual_output']
                        if "CMD_TIMEOUT_KILLED" in test['actual_output']:
                            obj["failed_timeout"] = True

                all_edges.append(obj)

    return pd.DataFrame(all_edges)

def read_engine_output(path):
    with open(path) as f:
        return json.loads(f.read())
    
def get_ca_metric(df, k):
    if k > 10:
        raise ValueError("k must be less than 10")
    
    df = df.groupby('request_id').head(k)
    total_requests = df.groupby('request_id')['status'].any().sum().item()
    total_translations_found = df[(df['status'] == 'TRANSLATION_FOUND')]
    total_found_at_least_one_translation = total_translations_found.groupby('request_id')['status'].any().sum().item()
    ca_metric = total_found_at_least_one_translation / total_requests * 100
    return ca_metric
    
def load_as_df(path):
    json = read_engine_output(path)
    df = to_pandas(json)
    return df

def get_translation(request):
    for path in request.paths:
        for edge in path.translation_edges:
            if edge.status == "TRANSLATION_FOUND":
                return edge.extracted_source_code
    return None

def get_percentage_timeout(response):
    timeout = 0
    totaltests = 0

    for request in response['translation_responses']:
        for path in request['paths']:
            for edge in path['translation_edges']:

                if "fuzzy_tests" in edge.keys():
                    tests = edge['fuzzy_tests']
                else:
                    tests = edge['unit_tests']

                for test in tests:
                    if test.get('actual_output', None) is not None:
                        if "CMD_TIMEOUT_KILLED" in test['actual_output']:
                            timeout = timeout + 1
                        totaltests = totaltests + 1

    return timeout/totaltests * 100

def save_response(path, file_name, response):
    df_response = to_pandas(response)
    #Save flatenned response to csv
    df_response.to_csv(os.path.join(path,f"{file_name}.csv"), index=False)
    #Serialized protobuf (full data)
    serialized_data = response.SerializeToString()
    with open(os.path.join(path,f"{file_name}.bin"), "wb") as f:
        f.write(serialized_data)
    #JSON representation of the protobuf (easier to work with)
    serialized_json = MessageToJson(response)
    with open(os.path.join(path,f"{file_name}.json"), "w") as f:
        f.write(serialized_json)

def get_stats(df_response):
    #Filter rows for the requests
    direct_translations = df_response[(df_response['status'] == 'TRANSLATION_FOUND') & (df_response['parent_edge_id'] == -1)]
    with_intermediate_translations = df_response[(df_response['status'] == 'TRANSLATION_FOUND')]

    total = df_response.groupby('request_id')
    count_total = total["status"].any().sum().item()

    count_direct_translations = direct_translations.groupby('request_id')['status'].any().sum().item()
    count_intermediate_translations = with_intermediate_translations.groupby('request_id')['status'].any().sum().item()

    ca_direct = count_direct_translations / count_total * 100
    ca_intermediates = count_intermediate_translations / count_total * 100

    return ca_direct, ca_intermediates

def print_stats(df_response):
    ca_direct, ca_intermediates = get_stats(df_response)
    print(f"Computational Accuracy - Only direct (baseline): {ca_direct:.2f}%")
    print(f"Computational Accuracy - Including intermediate langs: {ca_intermediates:.2f}%")
    print(f"Difference from baseline: {(ca_intermediates-ca_direct):.2f}%")

