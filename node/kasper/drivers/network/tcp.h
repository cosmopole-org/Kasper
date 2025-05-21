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

using json = nlohmann::json;

class SocketItem : public ISocketItem
{
public:
	int conn;
	SafeQueue<ValPack> buffer;
	bool ack;
	ICore *core;
	std::mutex lock;
	SocketItem(int conn, ICore *core);
	void writeRawUpdate(std::string key, char *updatePack, uint32_t len);
	void writeObjUpdate(std::string key, json updatePack);
	void writeRawResponse(std::string requestId, int resCode, char *response, uint32_t len);
	void writeObjResponse(std::string requestId, int resCode, json response);
	void pushBuffer();
	std::function<void(std::string, std::any, size_t)> connectListener(std::string uid);
	void processPacket(char *packet, uint32_t len);
};

class Tcp : public ITcp
{
	ICore *core;
	std::unordered_map<uint64_t, SocketItem *> sockets;
	uint64_t idCounter;

public:
	Tcp(ICore *core);
	std::shared_future<void> run(int port) override;
	void handleConnection(uint64_t connId, int conn);
};
