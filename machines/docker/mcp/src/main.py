from fastapi import FastAPI
from contextlib import asynccontextmanager
import asyncio
import uvicorn
import json
from mcp_use import MCPClient

shared_resources = {}
config = {}
with open('/app/config.json', 'r') as file:
    config = json.load(file)
client = MCPClient.from_dict(config)
shared_resources["mcp_client"] = client

app = FastAPI()

# Define a POST endpoint that calls the async function
@app.post("/run")
async def process_data(data: dict):
    print("received request.")
    request_body = data
    tool_name = request_body["name"]
    args = request_body["args"]
    
    print(tool_name)
    print(args)

    client = shared_resources.get("mcp_client")
    await client.create_all_sessions()
    session = client.get_session("redis")
    result = await session.call_tool(
        name=tool_name,
        arguments=args
    )

    return {"result": result.content[0].text}

@asynccontextmanager
async def lifespan(app: FastAPI):
    print("Application is starting up...")
    yield
    print("Application is shutting down...")
    shared_resources.clear()
    print("mcp client closed.")

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=3000)
