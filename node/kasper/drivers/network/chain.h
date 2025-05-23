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
#include "block.h"

using json = nlohmann::json;

class ChainSocketItem
{
public:
	IChain *chain;
	int conn;
	SafeQueue<DataPack> buffer;
	bool ack;
	ICore *core;
	EVP_PKEY *pkey;
	std::mutex lock;
	ChainSocketItem(IChain *chain, int conn, ICore *core);
	void writeRawUpdate(std::string targetType, std::string targetId, std::string key, char *updatePack, uint32_t len);
	void writeObjUpdate(std::string targetType, std::string targetId, std::string key, json updatePack);
	void writeRawResponse(std::string requestId, int resCode, char *response, uint32_t len);
	void writeObjResponse(std::string requestId, int resCode, json response);
	void pushBuffer();
	std::function<void(std::string, std::any, size_t)> connectListener(std::string uid);
	void processPacket(std::string origin, char *packet, uint32_t len);
	ChainSocketItem *openSocket(std::string origin);
};

class Chain : public IChain
{
public:
	ICore *core;
	std::vector<std::pair<std::string, std::string>> pendingTrxs;
	std::vector<Event*> pendingEvents;
	std::vector<Block> blocks;
	std::unordered_map<std::string, Event *> proofEvents;
	std::unordered_map<std::string, ChainSocketItem *> shardPeers;
	std::mutex lock;

	Chain(ICore *core);
	void handleConnection(std::string origin, int conn);
	void run(int port) override;
	void submitTrx(std::string t, std::string data) override;
	void addPendingEvent(Event *e) override;
	bool addBackedProof(std::string proof, std::string origin, std::string signedProof) override;
	void broadcastInShard(char* payload, uint32_t len) override;
	bool memorizeResponseBacked(std::string proof, std::string origin) override;
	Event *getEventByProof(std::string proof) override;
	int getOrderIndexOfEvent(std::string proof) override;
};
