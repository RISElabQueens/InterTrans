import pandas as pd
from transformers import StoppingCriteria, StoppingCriteriaList, AutoModelForCausalLM, AutoTokenizer
import torch
import re
import os
import torch
from tqdm import tqdm

def main(original_model, model_name, dataset_path, output_path, dataset_name):
    tokenizer = AutoTokenizer.from_pretrained(model_name)
    tokenizer.pad_token = tokenizer.eos_token
    
    if dataset_name == "codenet":
        prompt = """@@ Instruction
You are a skilled software developer proficient in multiple programming languages. Your task is to re-write the input source code. Below is the input source code written in {input_lang} that you should re-write into {target_lang} programming language. You must respond with the {target_lang} output code only. 

Source code:
```
{input_code}
```

@@ Response"""

    else:
        prompt = """@@ Instruction
You are a skilled software developer proficient in multiple programming languages. Your task is to re-write the input source code. Below is the input source code written in {input_lang} that you should re-write into {target_lang} programming language. You must respond with the {target_lang} output code only. 

Source code:
```
{input_code}
```

Your {target_lang} code must have this signature and include imports.
{signature}

@@ Response"""

    # Load the dataset
    dataset_df = pd.read_json(dataset_path, orient='records', lines=True)

    # Load the model and tokenizer
    model = AutoModelForCausalLM.from_pretrained(original_model, device_map={"": "cuda:0"}, torch_dtype=torch.float16, attn_implementation="flash_attention_2")
    model.resize_token_embeddings(len(tokenizer))
    model.load_adapter(model_name)

    results = []
    regex = r"(?s)\x60\x60\x60(?:javascript|java|cpp|csharp|python|script|rust|c|go|C\+\+|Javascript|JavaScript|Java|Python|C#|C|Rust|Script|Go)?(.*?)\x60\x60\x60"

    for index, example in tqdm(dataset_df.iterrows(), total=len(dataset_df)):

        if dataset_name == "codenet":
            prompt_formatted = prompt.format(
                input_lang=example['source_lang'],
                target_lang=example['target_lang'],
                input_code=example['input_code'],
            )
        else:
            prompt_formatted = prompt.format(
                input_lang=example['source_lang'],
                target_lang=example['target_lang'],
                input_code=example['input_code'],
                signature=example['target_signature'],
            )

        chat = [
            {"role": "user", "content": prompt_formatted},
        ]

        chat_text = tokenizer.apply_chat_template(chat, tokenize=False, add_generation_prompt=True)

        inputs = tokenizer(chat_text, return_tensors="pt").to('cuda:0')

        outputs = model.generate(
            **inputs,
            do_sample=True,
            top_p=0.95,
            top_k=10,
            temperature=0.7,
            max_new_tokens=4096,
            num_return_sequences=10,
            pad_token_id=tokenizer.pad_token_id
        )

        for output in outputs:
            all_text = tokenizer.decode(output, skip_special_tokens=True)
            new_tokens = all_text.split("Response", 1)[1]

            format_match = re.search(regex, new_tokens, re.DOTALL)

            if format_match:
                extracted_code = format_match.group(1)
            else:
                extracted_code = new_tokens

            obj = {
                "input_code": example['input_code'],
                "ground_truth": example['ground_truth'],
                "source_lang": example['source_lang'],
                "target_lang": example['target_lang'],
                "inference_output": all_text,
                "extracted_code": extracted_code
            }

            if dataset_name == "codenet":
                obj["id"] = example['task_id'],
            else:
                obj["id"] = example['id'],
                obj["target_signature"] = example['target_signature'],

            results.append(obj)

    df_results = pd.DataFrame(results)
    df_results.to_json(output_path, orient='records', lines=True)
if __name__ == "__main__":
    model_configs = [
        {
            "original_model" : "codellama/CodeLlama-13b-Instruct-hf",
            "model_name": "../data/models/CodeLlama-13b-Instruct-hf-eval",
            "dataset_path": "../data/datasets/humanevalx_dataset_subset.jsonl",
            "output_path": "../data/notebooks/rebuttal/raw_sft_inferences/codellama-13b-humanevalx.jsonl",
            "dataset" : "humanevalx"
        },
        {
            "original_model" : "ise-uiuc/Magicoder-S-DS-6.7B",
            "model_name": "../data/models/Magicoder-S-DS-6.7B-eval",
            "dataset_path": "../data/datasets/humanevalx_dataset_subset.jsonl",
            "output_path": "../data/notebooks/rebuttal/raw_sft_inferences/Magicoder-S-DS-6.7B-humanevalx.jsonl",
            "dataset" : "humanevalx"
        },
        {
            "original_model" : "bigcode/starcoder2-15b-instruct-v0.1",
            "model_name": "../data/models/starcoder2-15b-instruct-v0.1-eval",
            "dataset_path": "../data/datasets/humanevalx_dataset_subset.jsonl",
            "output_path": "../data/notebooks/rebuttal/raw_sft_inferences/starcoder2-15b-humanevalx.jsonl",
            "dataset" : "humanevalx"
        },
        {
            "original_model" : "codellama/CodeLlama-13b-Instruct-hf",
            "model_name": "../data/models/CodeLlama-13b-Instruct-hf-eval-codenet",
            "dataset_path": "../data/datasets/codenet_dataset_subset.jsonl",
            "output_path": "../data/notebooks/rebuttal/raw_sft_inferences/codellama-13b-codenet.jsonl",
            "dataset" : "codenet"
        },
        {
            "original_model" : "ise-uiuc/Magicoder-S-DS-6.7B",
            "model_name": "../data/models/Magicoder-S-DS-6.7B-eval-codenet",
            "dataset_path": "../data/datasets/codenet_dataset_subset.jsonl",
            "output_path": "../data/notebooks/rebuttal/raw_sft_inferences/Magicoder-S-DS-6.7B-codenet.jsonl",
            "dataset" : "codenet"
        },
        {
            "original_model" : "bigcode/starcoder2-15b-instruct-v0.1",
            "model_name": "../data/models/starcoder2-15b-instruct-v0.1-eval-codenet/checkpoint-1650",
            "dataset_path": "../data/datasets/codenet_dataset_subset.jsonl",
            "output_path": "../data/notebooks/rebuttal/raw_sft_inferences/starcoder2-15b-codenet.jsonl",
            "dataset" : "codenet"
        },
    ]

    for config in model_configs:
        print("Model: ", config["model_name"])
        main(config["original_model"], config["model_name"], config["dataset_path"], config["output_path"], config["dataset"])