from transformers import AutoModelForCausalLM, AutoTokenizer
import torch


class Arkadia:
    def __init__(self):
        self.tokenizer = AutoTokenizer.from_pretrained(
            "microsoft/DialoGPT-medium", device_map="auto", torch_dtype=torch.bfloat16
        )
        self.model = AutoModelForCausalLM.from_pretrained("microsoft/DialoGPT-medium")
        self.chat_history_ids = None
        self.role = "Arkadia"
        self.temperature = 0.5

    def think(self, query):
        chat = [
            {
                "role": "assistant",
                "content": "Your name is Arkadia.",
            },
            {
                "role": "user",
                "content": query,
            },
        ]
        formatted_chat = self.tokenizer.apply_chat_template(
            chat, tokenize=False, add_generation_prompt=True
        )
        print("Formatted chat:\n", formatted_chat)
        inputs = self.tokenizer(
            formatted_chat, return_tensors="pt", add_special_tokens=False
        )
        inputs = {key: tensor.to(self.model.device) for key, tensor in inputs.items()}
        outputs = self.model.generate(**inputs, max_new_tokens=512, do_sample=True, temperature=0.5)
        decoded_output = self.tokenizer.decode(
            outputs[0][inputs["input_ids"].size(1) :], skip_special_tokens=True
        )
        print("Decoded output:", decoded_output)

        return decoded_output
