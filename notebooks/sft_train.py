import copy
import os
import torch
from time import time
from datasets import load_dataset, Dataset
from peft import LoraConfig, PeftModel, prepare_model_for_kbit_training, get_peft_model
from transformers import (
    AutoConfig,
    AutoModelForCausalLM,
    AutoTokenizer,
    BitsAndBytesConfig,
    TrainingArguments,
    EarlyStoppingCallback
)
from accelerate import Accelerator
from trl import SFTConfig, SFTTrainer, DataCollatorForCompletionOnlyLM


completion_template = """```
{response}
```"""

def load_and_prepare_model(model_name, bnb_config):
    model_config = AutoConfig.from_pretrained(
        model_name,
        trust_remote_code=True,
        max_new_tokens=4096,
    )
    model = AutoModelForCausalLM.from_pretrained(
        model_name,
        trust_remote_code=True,
        config=model_config,
        quantization_config=bnb_config,
        attn_implementation="flash_attention_2",
        torch_dtype=torch.bfloat16,
    )
    return model

def print_tokens_with_ids(tokens, txt,tokenizer):
    tokens_names = tokenizer.tokenize(txt, add_special_tokens=False, padding="max_length", max_length=2048, truncation=False)
    print(list(zip(tokens_names, tokens)))

def return_tokens_with_ids(tokens, txt,tokenizer):
    tokens_names = tokenizer.tokenize(txt, add_special_tokens=False, padding="max_length", max_length=2048, truncation=False)
    return list(zip(tokens_names, tokens))
    
def main(model_name, dataset_path, output_dir, dataset_name):

    if dataset_name == "codenet":
        prompt = """You are a skilled software developer proficient in multiple programming languages. Your task is to re-write the input source code. Below is the input source code written in {input_lang} that you should re-write into {target_lang} programming language. You must respond with the {target_lang} output code only. 

Source code:
```
{input_code}
```"""

    else:
        prompt = """You are a skilled software developer proficient in multiple programming languages. Your task is to re-write the input source code. Below is the input source code written in {input_lang} that you should re-write into {target_lang} programming language. You must respond with the {target_lang} output code only. 

Source code:
```
{input_code}
```

Your {target_lang} code must have this signature and include imports.
{signature}"""


    accelerator = Accelerator()
    device = accelerator.device

    os.environ["CUDA_VISIBLE_DEVICES"] = "0,1,2,6"

    compute_dtype = torch.bfloat16
    bnb_config = BitsAndBytesConfig(
        load_in_4bit=True,
        bnb_4bit_quant_type="nf4",
        bnb_4bit_compute_dtype=compute_dtype,
        bnb_4bit_use_double_quant=False
    )

    # Load Dataset
    dataset = load_dataset("json", data_files=dataset_path, split="train")
    print(f"Original dataset size: {len(dataset)}")


    time_start = time()
    tokenizer = AutoTokenizer.from_pretrained(model_name)
    tokenizer.add_special_tokens({'pad_token': '[PAD]'})
    tokenizer.pad_token_id = tokenizer.convert_tokens_to_ids('[PAD]')
    tokenizer.padding_side = "left"

    def preprocess_function(example, dataset_name, prompt):

        if dataset_name == "codenet":
            formatted_prompt = prompt.format(
                input_lang=example['source_lang'],
                target_lang=example['target_lang'],
                input_code=example['input_code']
            )
        else:
            formatted_prompt = prompt.format(
                input_lang=example['source_lang'],
                target_lang=example['target_lang'],
                input_code=example['input_code'],
                signature=example['target_signature']
            )

        completion = completion_template.format(response=example['ground_truth'])

        chat = [
            {"role": "user", "content": formatted_prompt},
            {"role": "assistant", "content": completion},
        ]

        chat_text = tokenizer.apply_chat_template(chat, tokenize=False, continue_final_message=False, add_generation_prompt=False)
        return {"text" : chat_text}

    all_dataset = dataset.map(lambda x: preprocess_function(x, dataset_name, prompt), batched=False)

    split_dataset = all_dataset.train_test_split(test_size=0.1, seed=1)
    train_dataset = split_dataset['train']
    eval_dataset = split_dataset['test']

    model = load_and_prepare_model(model_name, bnb_config)
    model.resize_token_embeddings(len(tokenizer))
    time_end = time()
    print(f"Prepare model, tokenizer: {round(time_end-time_start, 3)} sec.")

    model = prepare_model_for_kbit_training(model)

    lora_config = LoraConfig(
        r=16,
        lora_alpha=32,
        lora_dropout=0.1,
        bias="none",
        target_modules=[
            "q_proj",
            "o_proj",
            "k_proj",
            "v_proj",
            "gate_proj",
            "up_proj",
            "down_proj",
        ],
        task_type="CAUSAL_LM",
    )

    # Apply LoRA to model
    model = get_peft_model(model, lora_config)
    model.print_trainable_parameters()

    early_stopping = EarlyStoppingCallback(early_stopping_patience=3)

    config = SFTConfig(
        output_dir=output_dir,
        max_seq_length=2048,
        per_device_train_batch_size=16,
        per_device_eval_batch_size=1,
        learning_rate=1e-4,
        num_train_epochs=30,  # Change based on your requirement
        evaluation_strategy="steps",
        eval_steps=150,
        save_steps=150,
        save_strategy="steps",
        load_best_model_at_end=True,
        save_total_limit=100,
        logging_steps=10,
        report_to="wandb",
    )


    trainer = SFTTrainer(
        model,
        train_dataset=train_dataset,
        eval_dataset=eval_dataset,
        args=config,
        processing_class=tokenizer,
        callbacks=[early_stopping]
    )

    trainer.train(resume_from_checkpoint = False)

    trainer.save_model(output_dir)

if __name__ == "__main__":
    model_configs = [
        {
            "model_name": "codellama/CodeLlama-13b-Instruct-hf",
            "dataset_path": "../data/datasets/humanevalx_dataset_training.jsonl",
            "output_dir": "../data/models/CodeLlama-13b-Instruct-hf-eval",
            "dataset" : "humanevalx"
        },
        {
            "model_name": "bigcode/starcoder2-15b-instruct-v0.1",
            "dataset_path": "../data/datasets/humanevalx_dataset_training.jsonl",
            "output_dir": "../data/models/starcoder2-15b-instruct-v0.1-eval",
            "dataset" : "humanevalx"
        },
        {
            "model_name": "ise-uiuc/Magicoder-S-DS-6.7B",
            "dataset_path": "../data/datasets/humanevalx_dataset_training.jsonl",
            "output_dir": "../data/models/Magicoder-S-DS-6.7B-eval",
            "dataset" : "humanevalx"
        },
        {
            "model_name": "codellama/CodeLlama-13b-Instruct-hf",
            "dataset_path": "../data/datasets/codenet_dataset_training.jsonl",
            "output_dir": "../data/models/CodeLlama-13b-Instruct-hf-eval-codenet",
            "dataset" : "codenet"
        },
        {
            "model_name": "ise-uiuc/Magicoder-S-DS-6.7B",
            "dataset_path": "../data/datasets/codenet_dataset_training.jsonl",
            "output_dir": "../data/models/Magicoder-S-DS-6.7B-eval-codenet",
            "dataset" : "codenet"
        },
        {
            "model_name": "bigcode/starcoder2-15b-instruct-v0.1",
            "dataset_path": "../data/datasets/codenet_dataset_training.jsonl",
            "output_dir": "../data/models/starcoder2-15b-instruct-v0.1-eval-codenet",
            "dataset" : "codenet"
        }
    ]

    for config in model_configs:
        main(config["model_name"], config["dataset_path"], config["output_dir"], config["dataset"])