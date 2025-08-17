import json
import requests
import argparse
from urllib.parse import unquote

# Create parser
parser = argparse.ArgumentParser(description="")

# Add arguments/flags
parser.add_argument("-payload", type=str, help="")

# Parse arguments
args = parser.parse_args()

decoded = unquote(args.payload)
decoded = decoded[1:decoded.__len__() - 1]

# Use them
payload = json.loads(decoded)

# Example URL
url = "http://localhost:3000/run"

headers = {
    "Content-Type": "application/json"
}

# Sending POST request
response = requests.post(url, json=payload, headers=headers)

# Print response
print(json.dumps(response.json()))
