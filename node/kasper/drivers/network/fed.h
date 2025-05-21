#pragma once

#include <queue>
#include <mutex>
#include <condition_variable>
#include <string>
#include <future>
#include <unordered_map>
#include <cstring>
#include <iostream>
#include <netinet/in.h>
#include <sys/socket.h>
#include <unistd.h>
#include "../../utils/nlohmann/json.hpp"
#include <any>
#include <optional>
#include "itcp.h"
#include "../../utils/utils.h"
#include "../../core/core/icore.h"
#include "../file/datapack.h"
#include <thread>
#include "queue.h"
#include <functional>

using json = nlohmann::json;

class SocketItem
{
public:
	Fed *fed;
	int conn;
	SafeQueue<ValPack> buffer;
	bool ack;
	ICore *core;
	std::mutex lock;
	void writeRawUpdate(std::string targetType, std::string targetId, std::string key, char *updatePack, uint32_t len);
	void writeObjUpdate(std::string targetType, std::string targetId, std::string key, json updatePack);
	void writeRawResponse(std::string requestId, int resCode, char *response, uint32_t len);
	void writeObjResponse(std::string requestId, int resCode, json response);
	void pushBuffer();
	std::function<void(std::string, std::any, size_t)> connectListener(std::string uid);
	void processPacket(char *packet, uint32_t len);
};

struct Request
{
public:
	std::string userId;
	std::string requestId;
	std::string key;
	ActionInput input;
	std::function<void(int resCode, std::string response)> callback;
};

class Fed : public IFed
{
public:
	ICore *core;
	std::unordered_map<uint64_t, SocketItem *> sockets;
	std::unordered_map<std::string, Request*> requests;
	uint64_t idCounter;
	uint64_t reqCounter;

	Fed(ICore *core);
	std::shared_future<void> run(int port) override;
	void handleConnection(uint64_t connId, int conn);
	void request(std::string origin, std::string userId, std::string key, std::string payload, std::string signature, ActionInput input, std::function<void(int, std::string)> callback) override;
};
