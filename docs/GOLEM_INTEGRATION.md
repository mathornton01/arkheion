# Golem / LLM Training Integration

This guide explains how to use Arkheion as a text corpus source for LLM training pipelines — specifically [Golem](https://github.com/example/golem) and any Hugging Face-compatible training setup.

---

## Overview

Arkheion extracts text from every uploaded PDF and EPUB using Apache Tika. The extracted text is stored in PostgreSQL and indexed in Meilisearch.

The `GET /api/v1/export` endpoint streams this text as JSONL — a format accepted by most training frameworks without conversion.

```
Upload Book → Tika Extracts Text → PostgreSQL Stores Text → Export JSONL → Train LLM
```

---

## Export Endpoint

```
GET /api/v1/export?format=jsonl
```

**Optional filters:**

| Parameter | Description | Example |
|-----------|-------------|---------|
| `tag` | Filter by tag slug | `tag=philosophy` |
| `category` | Filter by category | `category=Science` |
| `language` | Filter by ISO 639-1 code | `language=en` |

**Response format:** `application/x-ndjson` (newline-delimited JSON)

Each line:
```json
{"id":"uuid","title":"Book Title","authors":["Author Name"],"categories":["Science"],"language":"en","text":"Full extracted text content..."}
```

---

## Basic Usage

### Download All Books

```bash
curl "http://localhost:8080/api/v1/export?format=jsonl" \
  -H "X-API-Key: your-api-key" \
  -o my_library.jsonl

wc -l my_library.jsonl   # Number of books exported
```

### Download with Filters

```bash
# Only English philosophy books
curl "http://localhost:8080/api/v1/export?format=jsonl&tag=philosophy&language=en" \
  -H "X-API-Key: your-api-key" \
  -o philosophy_en.jsonl

# Science category only
curl "http://localhost:8080/api/v1/export?format=jsonl&category=Science" \
  -H "X-API-Key: your-api-key" \
  -o science.jsonl
```

### Stream to Processing Pipeline

```bash
# Pipe directly to a processing script without saving to disk
curl -s "http://localhost:8080/api/v1/export?format=jsonl" \
  -H "X-API-Key: your-api-key" \
  | python3 my_training_script.py
```

---

## Golem Pipeline Configuration

Configure Arkheion as a data source in Golem:

```yaml
# golem-pipeline.yaml
data_sources:
  - name: arkheion_library
    type: jsonl_http
    url: https://arkheion.example.com/api/v1/export
    params:
      format: jsonl
      language: en
    headers:
      X-API-Key: "${ARKHEION_API_KEY}"
    refresh_interval: daily

preprocessing:
  - type: text_field
    field: text
    min_length: 500          # Skip very short extractions
    max_length: 500000       # Truncate very long books
    deduplicate: true

training:
  format: causal_lm
  text_field: text
  metadata_fields:
    - title
    - authors
    - categories
```

---

## Hugging Face Datasets

Load directly from Arkheion into a Hugging Face dataset:

```python
import requests
import json
from datasets import Dataset

def load_arkheion(base_url: str, api_key: str, **filters) -> Dataset:
    """Load Arkheion library export as a Hugging Face Dataset."""
    params = {"format": "jsonl", **filters}
    headers = {"X-API-Key": api_key}

    response = requests.get(
        f"{base_url}/export",
        params=params,
        headers=headers,
        stream=True
    )
    response.raise_for_status()

    records = []
    for line in response.iter_lines():
        if line:
            records.append(json.loads(line))

    return Dataset.from_list(records)


# Usage
ds = load_arkheion(
    base_url="http://localhost:8080/api/v1",
    api_key="your-api-key",
    language="en",
    tag="philosophy"
)

print(ds)
# Dataset({features: ['id', 'title', 'authors', 'categories', 'language', 'text'], num_rows: 142})

# Push to Hugging Face Hub
ds.push_to_hub("your-username/my-library-corpus", private=True)
```

---

## Fine-tuning with LLaMA / Mistral

Example using the `trl` library:

```python
from datasets import Dataset
from trl import SFTTrainer
from transformers import AutoModelForCausalLM, AutoTokenizer
import requests
import json

# 1. Load books from Arkheion
def load_books():
    resp = requests.get(
        "http://localhost:8080/api/v1/export?format=jsonl",
        headers={"X-API-Key": "your-key"},
        stream=True
    )
    return [json.loads(line) for line in resp.iter_lines() if line]

books = load_books()

# 2. Format for instruction tuning
def format_for_training(book):
    return {
        "text": f"### Book: {book['title']}\n### Author: {', '.join(book['authors'])}\n\n{book['text'][:4096]}"
    }

dataset = Dataset.from_list([format_for_training(b) for b in books if len(b["text"]) > 200])

# 3. Train
model = AutoModelForCausalLM.from_pretrained("mistralai/Mistral-7B-v0.1")
tokenizer = AutoTokenizer.from_pretrained("mistralai/Mistral-7B-v0.1")

trainer = SFTTrainer(
    model=model,
    train_dataset=dataset,
    dataset_text_field="text",
    max_seq_length=4096,
)
trainer.train()
```

---

## Text Quality Notes

- **PDF text quality** varies by PDF type. Native PDFs (created digitally) produce clean text. Scanned PDFs require OCR — Tika includes Tesseract for basic OCR support, but quality depends on scan resolution.
- **EPUB text** is generally high quality (stored as structured HTML internally).
- Tika's output is whitespace-normalized by Arkheion (triple newlines collapsed, trailing spaces removed).
- Consider deduplication across books (some texts appear in multiple editions).

### Pre-processing Recommendations

```python
import re

def clean_text(text: str) -> str:
    # Remove page numbers (e.g. "\n42\n")
    text = re.sub(r'\n\d+\n', '\n', text)
    # Remove chapter headers that are just numbers
    text = re.sub(r'\n[IVXLC]+\n', '\n', text)
    # Normalize whitespace
    text = re.sub(r'\n{3,}', '\n\n', text)
    # Remove very short lines (likely formatting artifacts)
    lines = [l for l in text.split('\n') if len(l.strip()) > 3 or l.strip() == '']
    return '\n'.join(lines).strip()
```

---

## Webhook Trigger

Set up a webhook to automatically re-export when new books are added and extracted:

```python
# Simple webhook receiver (Flask example)
from flask import Flask, request
import subprocess

app = Flask(__name__)

@app.route("/webhooks/arkheion", methods=["POST"])
def handle_webhook():
    event = request.headers.get("X-Arkheion-Event")

    if event == "book.text_extracted":
        # Trigger incremental export
        subprocess.Popen(["python3", "update_training_corpus.py"])

    return "", 200
```
