import runpod
import time

# Initialize RunPod client
runpod.api_key = "rpa_1O58OQHD3WZ06YWJLBQRWWT33G8HXS0KCPWXHO8Iw8unx0"

# Your endpoint ID (get this from RunPod dashboard)
endpoint_id = "4gi3crdziy6rim"

def test_lorax_generation():
    """Test basic text generation"""
    
    # Example 1: Basic generation without adapter
    job = runpod.Endpoint(endpoint_id).run({
        "input": {
            "prompt": "What is the capital of France?",
            "max_new_tokens": 50,
            "temperature": 0.7
        }
    })
    
    print("Basic generation result:")
    print(job.output())
    
    # Example 2: Generation with a specific LoRA adapter
    job_with_adapter = runpod.Endpoint(endpoint_id).run({
        "input": {
            "prompt": "Write a creative story about a robot:",
            "adapter_id": "my-creative-writing-adapter",
            "max_new_tokens": 200,
            "temperature": 0.8,
            "top_p": 0.95
        }
    })
    
    print("\nGeneration with adapter result:")
    print(job_with_adapter.output())

def test_async_generation():
    """Test async generation for multiple requests"""
    
    prompts = [
        "Explain quantum computing in simple terms:",
        "Write a haiku about technology:",
        "What are the benefits of renewable energy?"
    ]
    
    jobs = []
    
    # Submit all jobs
    for prompt in prompts:
        job = runpod.Endpoint(endpoint_id).run({
            "input": {
                "prompt": prompt,
                "max_new_tokens": 100,
                "temperature": 0.7
            }
        })
        jobs.append(job)
    
    # Collect results
    for i, job in enumerate(jobs):
        print(f"\nResult {i+1}:")
        print(job.output())

if __name__ == "__main__":
    print("Testing LoRAX on RunPod Serverless...")
    test_lorax_generation()
    print("\n" + "="*50 + "\n")
    test_async_generation()
