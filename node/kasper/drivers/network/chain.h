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
#include "ichain.h"
#include "request.h"
#include "../../utils/utils.h"
#include "../../core/core/icore.h"
#include "../../core/core/actionio.h"
#include "../file/datapack.h"
#include <thread>
#include "queue.h"
#include <functional>
#include <vector>

using json = nlohmann::json;

class ChainSocketItem
{
public:
	IFed *fed;
	int conn;
	SafeQueue<DataPack> buffer;
	bool ack;
	ICore *core;
	std::mutex lock;
	ChainSocketItem(IChain *chain, int conn, ICore* core);
	void writeRawUpdate(std::string targetType, std::string targetId, std::string key, char *updatePack, uint32_t len);
	void writeObjUpdate(std::string targetType, std::string targetId, std::string key, json updatePack);
	void writeRawResponse(std::string requestId, int resCode, char *response, uint32_t len);
	void writeObjResponse(std::string requestId, int resCode, json response);
	void pushBuffer();
	std::function<void(std::string, std::any, size_t)> connectListener(std::string uid);
	void processPacket(std::string origin, char *packet, uint32_t len);
	ChainSocketItem *openSocket(std::string origin);
};

class Block {
	uint64_t index;
	std::vector<std::pair<std::string, std::string>> trxs;
};

class Chain : public IChain
{
public:
	ICore *core;
	std::vector<std::pair<std::string, std::string>> pendingTrxs;
	std::vector<Block> blocks;
	std::unordered_map<std::string, ChainSocketItem *> sockets;

	Chain(ICore *core);
	void handleConnection(std::string origin, int conn);
	void run(int port) override;
	void submitTrx(std::string data) override;
};
