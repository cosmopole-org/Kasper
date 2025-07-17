import runpod
import time

def handler(event):
    """RunPod handler function"""
    while True:
        time.sleep(5)
    
    return {"error": "No prompt provided"}

if __name__ == "__main__":
    # Start RunPod serverless
    runpod.serverless.start({"handler": handler})
