#include "fed.h"

#define SA struct sockaddr

using json = nlohmann::json;

void FedSocketItem::writeRawUpdate(std::string targetType, std::string targetId, std::string key, char *updatePack, uint32_t len)
{
	std::cerr << "preparing update..." << std::endl;

	std::string signature = this->core->signPacket(std::string(updatePack, len));
	const char *signBytes = signature.c_str();
	auto signBytesLen = Utils::getInstance().convertIntToData(signature.size());

	std::string target = targetType + "::" + targetId;
	const char *targetBytes = target.c_str();
	auto targetBytesLen = Utils::getInstance().convertIntToData(target.size());

	const char *keyBytes = key.c_str();
	auto keyBytesLen = Utils::getInstance().convertIntToData(key.size());

	uint32_t packetSize = 1 + 4 + signature.size() + 4 + target.size() + 4 + key.size() + len;
	auto packet = new char[packetSize];
	uint32_t pointer = 1;

	packet[0] = 0x01;

	memcpy(packet + pointer, signBytesLen, 4);
	pointer += 4;
	memcpy(packet + pointer, signBytes, signature.size());
	pointer += signature.size();
	delete signBytesLen;
	memcpy(packet + pointer, targetBytesLen, 4);
	pointer += 4;
	memcpy(packet + pointer, targetBytes, target.size());
	pointer += target.size();
	delete targetBytesLen;
	memcpy(packet + pointer, keyBytesLen, 4);
	pointer += 4;
	memcpy(packet + pointer, keyBytes, key.size());
	pointer += key.size();
	delete keyBytesLen;
	memcpy(packet + pointer, updatePack, len);
	pointer += len;

	std::cerr << "appending to buffer..." << std::endl;

	std::lock_guard<std::mutex> lock(this->lock);
	this->buffer.push({packet, packetSize});
	this->pushBuffer();
}

void FedSocketItem::writeObjUpdate(std::string targetType, std::string targetId, std::string key, json updatePack)
{
	std::string data = updatePack.dump();
	this->writeRawUpdate(targetType, targetId, key, &data[0], data.size());
}

void FedSocketItem::writeRawResponse(std::string requestId, int resCode, char *response, uint32_t len)
{
	std::cerr << "preparing response..." << std::endl;

	std::string signature = this->core->signPacket(std::string(response, len));
	const char *signBytes = signature.c_str();
	auto signBytesLen = Utils::getInstance().convertIntToData(signature.size());

	const char *b1 = requestId.c_str();
	char *b1Len = Utils::getInstance().convertIntToData(requestId.size());

	char *b2 = Utils::getInstance().convertIntToData(resCode);

	uint32_t packetSize = 1 + 4 + signature.size() + 4 + requestId.size() + 4 + len;
	char *packet = new char[packetSize];

	uint32_t pointer = 1;

	packet[0] = 0x02;

	memcpy(packet + pointer, signBytesLen, 4);
	pointer += 4;
	delete signBytesLen;
	memcpy(packet + pointer, signBytes, signature.size());
	pointer += signature.size();
	memcpy(packet + pointer, b1Len, 4);
	pointer += 4;
	delete b1Len;
	memcpy(packet + pointer, b1, requestId.size());
	pointer += requestId.size();
	memcpy(packet + pointer, b2, 4);
	pointer += 4;
	delete b2;
	memcpy(packet + pointer, response, len);
	pointer += len;

	std::cerr << "appending to buffer..." << std::endl;

	std::lock_guard<std::mutex> lock(this->lock);
	this->buffer.push({packet, packetSize});
	this->pushBuffer();
}

void FedSocketItem::writeObjResponse(std::string requestId, int resCode, json response)
{
	std::string data = response.dump();
	this->writeRawResponse(requestId, resCode, &data[0], data.size());
}

void FedSocketItem::pushBuffer()
{
	if (this->ack)
	{
		if (this->buffer.size() > 0)
		{
			this->ack = false;
			auto data = this->buffer.peek();
			char *packetLen = Utils::getInstance().convertIntToData(data.len);
			auto res = send(this->conn, packetLen, 4, 0);
			delete packetLen;
			if (res == -1)
			{
				this->ack = true;
				std::cerr << "error writing to socket." << std::endl;
			}
			send(this->conn, data.data, data.len, 0);
			if (res == -1)
			{
				this->ack = true;
				std::cerr << "error writing to socket." << std::endl;
			}
		}
	}
}

std::string userTargetPrefix = "user::";
std::string pointTargetPrefix = "point::";

FedSocketItem *FedSocketItem::openSocket(std::string origin)
{
	int sockfd, connfd;
	struct sockaddr_in servaddr, cli;
	sockfd = socket(AF_INET, SOCK_STREAM, 0);
	if (sockfd == -1)
	{
		printf("socket creation failed...\n");
		return NULL;
	}
	else
		printf("Socket successfully created..\n");
	bzero(&servaddr, sizeof(servaddr));
	servaddr.sin_family = AF_INET;
	servaddr.sin_addr.s_addr = inet_addr(origin.c_str());
	servaddr.sin_port = htons(8081);
	if (connect(sockfd, (SA *)&servaddr, sizeof(servaddr)) != 0)
	{
		printf("connection with the server failed...\n");
		return NULL;
	}
	else
		printf("connected to the server..\n");
	return new FedSocketItem(this->fed, sockfd, this->core);
}

void FedSocketItem::processPacket(std::string origin, char *packet, uint32_t len)
{
	if (len == 1 && packet[0] == 0x01)
	{
		std::lock_guard<std::mutex> lock(this->lock);
		this->ack = true;
		if (this->buffer.size() > 0)
		{
			auto top = this->buffer.peek();
			this->buffer.try_pop();
			delete top.data;
			this->pushBuffer();
		}
		return;
	}

	std::cerr << "received packet length: " << len << std::endl;

	std::string signature = "";
	std::string userId = "";
	std::string path = "";
	std::string packetId = "";
	DataPack payload;

	uint32_t pointer = 1;

	if (packet[0] == 0x01)
	{
		std::string signature = "";
		std::string target = "";
		std::string key = "";

		std::cerr << "received update" << std::endl;

		char *tempBytes = new char[4];
		memcpy(tempBytes, packet + pointer, 4);
		uint32_t signatureLength = Utils::getInstance().parseDataAsInt(tempBytes);
		delete tempBytes;
		std::cerr << "signature length: " << signatureLength << std::endl;
		pointer += 4;
		if (signatureLength > 0)
		{
			char *sign = new char[signatureLength];
			memcpy(sign, packet + pointer, signatureLength);
			signature = std::string(sign, signatureLength);
			delete sign;
			pointer += signatureLength;
		}
		std::cerr << "signature: " << signature << std::endl;
		char *targetBytesLen = new char[4];
		memcpy(targetBytesLen, packet + pointer, 4);
		pointer += 4;
		uint32_t targetLen = Utils::getInstance().parseDataAsInt(targetBytesLen);
		std::cerr << "target length: " << targetLen << std::endl;
		delete targetBytesLen;
		if (targetLen > 0)
		{
			char *targetBytes = new char[targetLen];
			memcpy(targetBytes, packet + pointer, targetLen);
			pointer += targetLen;
			target = std::string(targetBytes, targetLen);
			delete targetBytes;
		}

		char *keyBytesLen = new char[4];
		memcpy(keyBytesLen, packet + pointer, 4);
		pointer += 4;
		uint32_t keyLen = Utils::getInstance().parseDataAsInt(keyBytesLen);
		std::cerr << "key length: " << keyLen << std::endl;
		delete keyBytesLen;
		if (keyLen > 0)
		{
			char *keyBytes = new char[keyLen];
			memcpy(keyBytes, packet + pointer, keyLen);
			pointer += keyLen;
			key = std::string(keyBytes, keyLen);
			delete keyBytes;
		}
		std::cerr << "key: " << key << std::endl;
		uint32_t payloadLen = len - pointer;
		std::cerr << "payload length: " << payloadLen << std::endl;
		if (payloadLen > 0)
		{
			char *payload = new char[payloadLen];
			memcpy(payload, packet + pointer, payloadLen);
			std::string updatePack = std::string(payload, payloadLen);
			std::cerr << "payload: " << updatePack << std::endl;
			pointer += payloadLen;

			if (key == "/points/join")
			{
				std::string jsonStr = std::string(payload, payloadLen);
				json j = json::parse(jsonStr);
				std::string pointId = j["point"]["id"].template get<std::string>();
				std::string userId = j["user"]["id"].template get<std::string>();
				this->core->getTools()->getSignaler()->join(userId, pointId);
			}
			else if (key == "/points/leave")
			{
				std::string jsonStr = std::string(payload, payloadLen);
				json j = json::parse(jsonStr);
				std::string pointId = j["point"]["id"].template get<std::string>();
				std::string userId = j["user"]["id"].template get<std::string>();
				this->core->getTools()->getSignaler()->leave(userId, pointId);
			}

			if (Utils::getInstance().stringStartsWith(target, userTargetPrefix))
			{
				this->core->getTools()->getSignaler()->signalUserAsBytes(target.substr(userTargetPrefix.length()), key, payload, payloadLen);
			}
			else
			{
				this->core->getTools()->getSignaler()->signalPointAsBytes(target.substr(pointTargetPrefix.length()), key, payload, payloadLen, {});
			}
		}
	}
	else if (packet[0] == 0x02)
	{
		std::string signature = "";
		std::string requestId = "";

		std::cerr << "received response" << std::endl;

		char *tempBytes = new char[4];
		memcpy(tempBytes, packet + pointer, 4);
		uint32_t signatureLength = Utils::getInstance().parseDataAsInt(tempBytes);
		delete tempBytes;
		std::cerr << "signature length: " << signatureLength << std::endl;
		pointer += 4;
		if (signatureLength > 0)
		{
			char *sign = new char[signatureLength];
			memcpy(sign, packet + pointer, signatureLength);
			signature = std::string(sign, signatureLength);
			delete sign;
			pointer += signatureLength;
		}

		char *b1Len = new char[4];
		memcpy(b1Len, packet + pointer, 4);
		pointer += 4;
		uint32_t requestIdLen = Utils::getInstance().parseDataAsInt(b1Len);
		std::cerr << "requestId length: " << requestIdLen << std::endl;
		delete b1Len;
		if (requestIdLen > 0)
		{
			char *b1 = new char[4];
			memcpy(b1, packet + pointer, requestIdLen);
			pointer += requestIdLen;
			requestId = std::string(b1, requestIdLen);
			std::cerr << "requestId: " << requestId << std::endl;
			delete b1;
		}
		char *b2 = new char[4];
		memcpy(b2, packet + pointer, 4);
		pointer += 4;
		int resCode = Utils::getInstance().parseDataAsInt(b2);
		std::cerr << "resCode: " << resCode << std::endl;
		delete b2;
		int payloadLength = len - pointer;
		if (payloadLength > 0)
		{
			char *payload = new char[payloadLength];
			memcpy(payload, packet + pointer, payloadLength);
			std::string response = std::string(payload, payloadLength);
			std::cerr << "response: " << response << std::endl;
			pointer += payloadLength;

			if (auto req = this->fed->findRequest(requestId); req != NULL)
			{
				if (resCode == 0)
				{
					if (req->key == "/points/create")
					{
						json res = json::parse(response);
						this->core->getTools()->getSignaler()->createPoint({
							{"id", res["point"]["id"].template get<std::string>()},
							{"owner", res["point"]["owner"].template get<std::string>()},
							{"isPublic", (res["point"]["isPublic"].template get<std::string>().c_str()[0] == 0x01)},
							{"persHist", (res["point"]["persHist"].template get<std::string>().c_str()[0] == 0x01)},
						});
					}
					else if (req->key == "/points/join")
					{
						this->core->getTools()->getSignaler()->join(userId, this->core->getActor()->findActionAsSecure(req->key)->getIntel()->extractMeta(req->input.data).pointId);
					}
					else if (req->key == "/points/leave")
					{
						this->core->getTools()->getSignaler()->leave(userId, this->core->getActor()->findActionAsSecure(req->key)->getIntel()->extractMeta(req->input.data).pointId);
					}
				}
				req->callback(resCode, response);
				this->fed->clearRequest(requestId);
				delete req;
			}
		}
	}
	else if (packet[0] == 0x03)
	{
		char *tempBytes = new char[4];
		memcpy(tempBytes, packet + pointer, 4);
		uint32_t signatureLength = Utils::getInstance().parseDataAsInt(tempBytes);
		delete tempBytes;
		std::cerr << "signature length: " << signatureLength << std::endl;
		pointer += 4;
		if (signatureLength > 0)
		{
			char *sign = new char[signatureLength];
			memcpy(sign, packet + pointer, signatureLength);
			signature = std::string(sign, signatureLength);
			delete sign;
			pointer += signatureLength;
		}
		std::cerr << "signature: " << signature << std::endl;

		char *tempBytes3 = new char[4];
		memcpy(tempBytes3, packet + pointer, 4);
		uint32_t userIdLength = Utils::getInstance().parseDataAsInt(tempBytes3);
		delete tempBytes3;
		std::cerr << "userId length: " << userIdLength << std::endl;
		pointer += 4;
		if (userIdLength > 0)
		{
			char *tempBytes4 = new char[userIdLength];
			memcpy(tempBytes4, packet + pointer, userIdLength);
			userId = std::string(tempBytes4, userIdLength);
			delete tempBytes4;
			pointer += userIdLength;
		}
		std::cerr << "userId: " << userId << std::endl;

		char *tempBytes5 = new char[4];
		memcpy(tempBytes5, packet + pointer, 4);
		uint32_t pathLength = Utils::getInstance().parseDataAsInt(tempBytes5);
		delete tempBytes5;
		std::cerr << "path length: " << pathLength << std::endl;
		pointer += 4;
		if (pathLength > 0)
		{
			char *tempBytes6 = new char[pathLength];
			memcpy(tempBytes6, packet + pointer, pathLength);
			path = std::string(tempBytes6, pathLength);
			delete tempBytes6;
			pointer += pathLength;
		}
		std::cerr << "path: " << path << std::endl;

		char *tempBytes7 = new char[4];
		memcpy(tempBytes7, packet + pointer, 4);
		uint32_t packetIdLength = Utils::getInstance().parseDataAsInt(tempBytes7);
		delete tempBytes7;
		std::cerr << "packetId length: " << packetIdLength << std::endl;
		pointer += 4;
		if (packetIdLength > 0)
		{
			char *tempBytes8 = new char[packetIdLength];
			memcpy(tempBytes8, packet + pointer, packetIdLength);
			packetId = std::string(tempBytes8, packetIdLength);
			delete tempBytes8;
			pointer += packetIdLength;
		}
		std::cerr << "packetId: " << packetId << std::endl;

		char *payloadRaw = new char[len - pointer];
		memcpy(payloadRaw, packet + pointer, len - pointer);
		payload = DataPack{payloadRaw, len - pointer};
		std::cerr << "payload: " << payload.data << std::endl;

		auto s = this->openSocket(origin);

		if (s != NULL)
		{
			try
			{
				auto action = this->core->getActor()->findActionAsSecure(path);
				if (action == NULL)
				{
					json res;
					res["message"] = "action not found";
					s->writeObjResponse(packetId, 1, res);
					delete payloadRaw;
					return;
				}
				auto response = action->runAsFed([this](std::function<void(StateTrx *)> fn)
												 { this->core->modifyState(fn); }, core->getTools(), userId, payload, signature);
				if (response.err != "")
				{
					json data;
					data["message"] = response.err;
					s->writeObjResponse(packetId, response.resCode, data);
					delete payloadRaw;
					return;
				}
				s->writeObjResponse(packetId, 0, response.data);
			}
			catch (const std::exception &e)
			{
				std::cerr << "Standard exception caught: " << e.what() << std::endl;
				json data;
				data["message"] = e.what();
				s->writeObjResponse(packetId, 2, data);
			}
			catch (...)
			{
				std::cerr << "Unknown exception caught" << std::endl;
			}
			delete payloadRaw;

			close(s->conn);
			delete s;
		}
	}
}

FedSocketItem::FedSocketItem(IFed *fed, int conn, ICore *core)
{
	this->fed = fed;
	this->conn = conn;
	this->core = core;
	this->ack = true;
}

Fed::Fed(ICore *core)
{
	this->core = core;
	this->idCounter = 0;
	this->sockets = {};
}

void Fed::run(int port)
{
	std::cerr << "starting fed server on port " << port << "..." << std::endl;
	std::thread t([port, this]
				  {
			int serverSocket = socket(AF_INET, SOCK_STREAM, 0);
			sockaddr_in serverAddress;
			serverAddress.sin_family = AF_INET;
			serverAddress.sin_port = htons(port);
			serverAddress.sin_addr.s_addr = INADDR_ANY;
			bind(serverSocket, (struct sockaddr*)&serverAddress,
				 sizeof(serverAddress));
			listen(serverSocket, 5);
			while (true)
			{
				struct sockaddr_in client_addr{};
				socklen_t client_len = sizeof(client_addr);

				int clientSocket = accept(serverSocket, (struct sockaddr*)&client_addr, &client_len);
				std::cerr << "new client connected." << std::endl;

				char client_ip[INET_ADDRSTRLEN];
				inet_ntop(AF_INET, &client_addr.sin_addr, client_ip, sizeof(client_ip));
				std::string origin = std::string(client_ip, sizeof(client_ip));
				std::cerr << "connection from origin: " << origin << std::endl;

				auto id = this->idCounter++;
				std::thread t([this, clientSocket, id, origin]{
					this->handleConnection(origin, id, clientSocket);
					this->sockets.erase(id);
					close(clientSocket);
				});
				t.detach();
	 		} });
	t.detach();
}

Request *Fed::findRequest(std::string requestId)
{
	if (auto req = this->requests.find(requestId); req != this->requests.end())
	{
		return req->second;
	}
	return NULL;
}

void Fed::clearRequest(std::string requestId)
{
	this->requests.erase(requestId);
}

void Fed::handleConnection(std::string origin, uint64_t connId, int conn)
{
	auto socket = new FedSocketItem(this, conn, this->core);
	this->sockets.insert({connId, socket});
	char lenBuf[4];
	char buf[1024];
	char nextBuf[2048];
	uint64_t readCount = 0;
	uint64_t oldReadCount = 0;
	bool enough = false;
	bool beginning = true;
	uint32_t length = 0;
	uint32_t readLength = 0;
	uint32_t remainedReadLength = 0;
	char *readData;
	int counter = 0;
	while (true)
	{
		if (!enough)
		{
			readLength = recv(conn, buf, sizeof(buf), 0);
			if (readLength == 0)
			{
				std::cerr << "socket closed" << std::endl;
				return;
			}
			else if (readLength == -1)
			{
				std::cerr << "socket error" << std::endl;
				return;
			}
			oldReadCount = readCount;
			readCount += readLength;
			memcpy(nextBuf + remainedReadLength, buf, readLength);
			remainedReadLength += readLength;
		}

		if (beginning)
		{
			if (readCount >= 4)
			{
				memcpy(lenBuf, nextBuf, 4);
				remainedReadLength -= 4;
				memcpy(nextBuf, nextBuf + 4, remainedReadLength);
				length = Utils::getInstance().parseDataAsInt(lenBuf);
				readData = new char[length];
				memset(readData, 0, length);
				readCount -= 4;
				beginning = false;
				enough = true;
			}
			else
			{
				enough = false;
			}
		}
		else
		{
			if (remainedReadLength == 0)
			{
				enough = false;
			}
			else if (readCount >= length)
			{
				memcpy(readData + oldReadCount, nextBuf, length - oldReadCount);
				memcpy(nextBuf, nextBuf + (readLength - (readCount - length)), readCount - length);
				remainedReadLength = (readCount - length);
				std::cerr << "packet received" << std::endl;
				char *packet = new char[length];
				memcpy(packet, readData, length);
				delete readData;

				std::thread t([&socket, length, origin](char *packet)
							  {
								   try
								   {
									   socket->processPacket(origin, packet, length);
								   }
								   catch (const std::exception &e)
								   {
									   std::cerr << "Standard exception caught: " << e.what() << std::endl;
								   }
								   catch (...)
								   {
									   std::cerr << "Unknown exception caught" << std::endl;
								   }
								   delete packet; }, packet);
				t.detach();
				readCount -= length;
				enough = true;
				beginning = true;
			}
			else
			{
				memcpy(readData + oldReadCount, nextBuf, readCount - oldReadCount);
				remainedReadLength = 0;
				enough = true;
			}
		}
	}
}

void Fed::request(std::string origin, std::string userId, std::string key, std::string payload, std::string signature, ActionInput input, std::function<void(int, std::string)> callback)
{
	int sockfd, connfd;
	struct sockaddr_in servaddr, cli;
	sockfd = socket(AF_INET, SOCK_STREAM, 0);
	if (sockfd == -1)
	{
		printf("socket creation failed...\n");
		return;
	}
	else
		printf("Socket successfully created..\n");
	bzero(&servaddr, sizeof(servaddr));
	servaddr.sin_family = AF_INET;
	servaddr.sin_addr.s_addr = inet_addr(origin.c_str());
	servaddr.sin_port = htons(8081);
	if (connect(sockfd, (SA *)&servaddr, sizeof(servaddr)) != 0)
	{
		printf("connection with the server failed...\n");
		return;
	}
	else
		printf("connected to the server..\n");

	uint32_t newReqNum = this->reqCounter++;
	std::string pid = std::to_string(newReqNum);

	uint32_t packetLen = 1 + 4 + signature.size() + 4 + userId.size() + 4 + key.size() + 4 + pid.size() + payload.size();
	int pointer = 1;
	char *packet = new char[packetLen];
	packet[0] = 0x03;
	int signatureSize = signature.size();
	char *signatureLength = Utils::getInstance().convertIntToData(signatureSize);
	memcpy(packet + pointer, signatureLength, 4);
	pointer += 4;
	delete signatureLength;
	memcpy(packet + pointer, signature.c_str(), signatureSize);
	pointer += signatureSize;

	int userIdSize = userId.size();
	char *userIdLength = Utils::getInstance().convertIntToData(userIdSize);
	memcpy(packet + pointer, userIdLength, 4);
	pointer += 4;
	delete userIdLength;
	memcpy(packet + pointer, userId.c_str(), userIdSize);
	pointer += userIdSize;

	int keySize = key.size();
	char *keyLength = Utils::getInstance().convertIntToData(keySize);
	memcpy(packet + pointer, keyLength, 4);
	pointer += 4;
	delete keyLength;
	memcpy(packet + pointer, key.c_str(), keySize);
	pointer += keySize;

	int pidSize = pid.size();
	char *pidLength = Utils::getInstance().convertIntToData(pidSize);
	memcpy(packet + pointer, pidLength, 4);
	pointer += 4;
	delete pidLength;
	memcpy(packet + pointer, pid.c_str(), pidSize);
	pointer += pidSize;

	memcpy(packet + pointer, payload.c_str(), payload.size());

	this->requests.insert({pid,
						   new Request{
							   userId,
							   pid,
							   key,
							   input,
							   callback}});
	char *packetLenBytes = Utils::getInstance().convertIntToData(packetLen);
	send(sockfd, packetLenBytes, 4, 0);
	send(sockfd, packet, packetLen, 0);
	delete packet;
	delete packetLenBytes;

	close(sockfd);
}
