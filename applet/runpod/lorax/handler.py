import runpod
import subprocess
import time
import requests
import json
import os
from threading import Thread

# Global variable to track server status
server_process = None
server_ready = False

def start_lorax_server():
    """Start the LoRAX server in a separate thread"""
    global server_process, server_ready
    
    model_id = os.environ.get('MODEL_ID', 'mistralai/Mistral-7B-Instruct-v0.1')
    max_input_length = int(os.environ.get('MAX_INPUT_LENGTH', '4096'))
    max_total_tokens = int(os.environ.get('MAX_TOTAL_TOKENS', '5096'))
    
    cmd = [
        'lorax-launcher',
        "--json-output",
        '--model-id', model_id,
        '--port', '80',
    ]
    
    print(f"Starting LoRAX server with command: {' '.join(cmd)}")
    
    server_process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    
    # Wait for server to be ready
    max_retries = 60
    for i in range(max_retries):
        try:
            response = requests.get('http://localhost:80/health')
            if response.status_code == 200:
                server_ready = True
                print("LoRAX server is ready!")
                return
        except requests.exceptions.ConnectionError:
            pass
        
        time.sleep(5)
        print(f"Waiting for server to start... ({i+1}/{max_retries})")
    
    print("Failed to start LoRAX server")

def generate_text(prompt, adapter_id=None, max_new_tokens=100, temperature=0.7, top_p=0.95):
    """Generate text using LoRAX API"""
    if not server_ready:
        return {"error": "Server not ready"}
    
    url = "http://localhost:80/generate"
    
    payload = {
        "inputs": prompt,
        "parameters": {
            "max_new_tokens": max_new_tokens,
            "temperature": temperature,
            "top_p": top_p,
            "do_sample": True
        }
    }
    
    # Add adapter_id if provided
    if adapter_id:
        payload["parameters"]["adapter_id"] = adapter_id
    
    try:
        response = requests.post(url, json=payload, timeout=300)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        return {"error": f"Request failed: {str(e)}"}

def handler(event):
    """RunPod handler function"""
    global server_ready
    
    # Start server if not already running
    if not server_ready:
        print("Starting LoRAX server...")
        start_lorax_server()
    
    # Extract parameters from event
    input_data = event.get('input', {})
    prompt = input_data.get('prompt', '')
    adapter_id = input_data.get('adapter_id', None)
    max_new_tokens = input_data.get('max_new_tokens', 100)
    temperature = input_data.get('temperature', 0.7)
    top_p = input_data.get('top_p', 0.95)
    
    if not prompt:
        return {"error": "No prompt provided"}
    
    # Generate text
    result = generate_text(
        prompt=prompt,
        adapter_id=adapter_id,
        max_new_tokens=max_new_tokens,
        temperature=temperature,
        top_p=top_p
    )
    
    return result

def upload_adapter(adapter_path, adapter_id):
    """Upload a LoRA adapter to the server"""
    if not server_ready:
        return {"error": "Server not ready"}
    
    url = "http://localhost:80/adapter"
    
    files = {'file': open(adapter_path, 'rb')}
    data = {'adapter_id': adapter_id}
    
    try:
        response = requests.post(url, files=files, data=data)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        return {"error": f"Upload failed: {str(e)}"}

# Start the server in a separate thread
server_thread = Thread(target=start_lorax_server)
server_thread.daemon = True
server_thread.start()

if __name__ == "__main__":
    # Start RunPod serverless
    runpod.serverless.start({"handler": handler})
