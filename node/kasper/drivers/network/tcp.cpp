#include "tcp.h"

using json = nlohmann::json;

template <typename T>
void SafeQueue<T>::push(const T &item)
{
	{
		std::lock_guard<std::mutex> lock(mutex_);
		queue_.push(item);
	}
	cond_var_.notify_one();
}

template <typename T>
void SafeQueue<T>::push(T &&item)
{
	{
		std::lock_guard<std::mutex> lock(mutex_);
		queue_.push(std::move(item));
	}
	cond_var_.notify_one();
}

template <typename T>
T SafeQueue<T>::wait_and_pop()
{
	std::unique_lock<std::mutex> lock(mutex_);
	cond_var_.wait(lock, [this]()
				   { return !queue_.empty(); });

	T item = std::move(queue_.front());
	queue_.pop();
	return item;
}

template <typename T>
std::optional<T> SafeQueue<T>::try_pop()
{
	std::lock_guard<std::mutex> lock(mutex_);
	if (queue_.empty())
	{
		return std::nullopt;
	}

	T item = std::move(queue_.front());
	queue_.pop();
	return item;
}

template <typename T>
bool SafeQueue<T>::empty() const
{
	std::lock_guard<std::mutex> lock(mutex_);
	return queue_.empty();
}

template <typename T>
size_t SafeQueue<T>::size() const
{
	std::lock_guard<std::mutex> lock(mutex_);
	return queue_.size();
}

template <typename T>
ValPack SafeQueue<T>::peek() const
{
	std::lock_guard<std::mutex> lock(mutex_);
	return queue_.front();
}

void SocketItem::writeRawUpdate(std::string key, char *updatePack, uint32_t len)
{
	std::cerr << "preparing update..." << std::endl;

	const char *keyBytes = key.c_str();
	auto keyBytesLen = Utils::getInstance().convertIntToData(key.size());

	uint32_t packetSize = 1 + 4 + key.size() + len;
	auto packet = new char[packetSize];
	uint32_t pointer = 1;

	packet[0] = 0x01;

	memcpy(packet + pointer, keyBytesLen, 4);
	pointer += 4;
	delete keyBytesLen;
	memcpy(packet + pointer, keyBytes, key.size());
	pointer += key.size();
	memcpy(packet + pointer, updatePack, len);
	pointer += len;

	std::cerr << "appending to buffer..." << std::endl;

	std::lock_guard<std::mutex> lock(this->lock);
	this->buffer.push({packet, packetSize});
	this->pushBuffer();
}

void SocketItem::writeObjUpdate(std::string key, json updatePack)
{
	std::string data = updatePack.dump();
	this->writeRawUpdate(key, &data[0], data.size());
}

void SocketItem::writeRawResponse(std::string requestId, int resCode, char *response, uint32_t len)
{
	std::cerr << "preparing response..." << std::endl;

	const char *b1 = requestId.c_str();
	char *b1Len = Utils::getInstance().convertIntToData(requestId.size());

	char *b2 = Utils::getInstance().convertIntToData(resCode);

	uint32_t packetSize = 1 + 4 + requestId.size() + 4 + len;
	char *packet = new char[packetSize];

	uint32_t pointer = 1;

	packet[0] = 0x02;

	memcpy(packet + pointer, b1Len, 4);
	pointer += 4;
	delete b1Len;
	memcpy(packet + pointer, b1, requestId.size());
	pointer += requestId.size();
	memcpy(packet + pointer, b2, 4);
	pointer += 4;
	memcpy(packet + pointer, response, len);
	pointer += len;

	std::cerr << "appending to buffer..." << std::endl;

	std::lock_guard<std::mutex> lock(this->lock);
	this->buffer.push({packet, packetSize});
	this->pushBuffer();
}

void SocketItem::writeObjResponse(std::string requestId, int resCode, json response)
{
	std::string data = response.dump();
	this->writeRawResponse(requestId, resCode, &data[0], data.size());
}

void SocketItem::pushBuffer()
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

std::function<void(std::string, std::any, size_t)> SocketItem::connectListener(std::string uid)
{
	auto lis = [this](std::string key, std::any data, size_t len)
	{
		if (len == 0)
		{
			this->writeObjUpdate(key, std::any_cast<json>(data));
		}
		else
		{
			this->writeRawUpdate(key, std::any_cast<char *>(data), len);
		}
	};
	this->ack = true;
	return lis;
}

void SocketItem::processPacket(char *packet, uint32_t len)
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

	uint32_t pointer = 0;
	char *tempBytes = new char[4];
	memcpy(tempBytes, packet + pointer, 4);
	std::cerr << "kasper " << static_cast<int>(packet[0]) << " " << static_cast<int>(packet[1]) << " " << static_cast<int>(packet[2]) << " " << static_cast<int>(packet[3]) << std::endl;
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

	try
	{
		if (path == "authenticate")
		{
			std::function<void(std::string, std::any, size_t)> lis;
			auto checkRes = this->core->getTools()->getSecurity()->authWithSignature(userId, std::string(payload.data, payload.len), &signature[0]);
			if (checkRes.verified)
			{
				lis = this->connectListener(userId);
				json res;
				res["message"] = "authenticated";
				this->writeObjResponse(packetId, 0, res);
				this->core->getTools()->getSignaler()->listenToSingle(userId, lis);
				json obj;
				lis("old_queue_end", obj, 0);
			}
			else
			{
				json res;
				res["message"] = "authentication failed";
				this->writeObjResponse(packetId, 4, res);
			}
			delete payloadRaw;
			return;
		}
		auto action = this->core->getActor()->findActionAsSecure(path);
		if (action == NULL)
		{
			json res;
			res["message"] = "action not found";
			this->writeObjResponse(packetId, 1, res);
			delete payloadRaw;
			return;
		}
		auto response = action->run(this->core->getIp(), [this](std::function<void(StateTrx *)> fn)
									{ this->core->modifyState(fn); }, core->getTools(), userId, payload, signature);
		if (response.err != "")
		{
			json data;
			data["message"] = response.err;
			this->writeObjResponse(packetId, response.resCode, data);
			delete payloadRaw;
			return;
		}
		this->writeObjResponse(packetId, 0, response.data);
	}
	catch (const std::exception &e)
	{
		std::cerr << "Standard exception caught: " << e.what() << std::endl;
		json data;
		data["message"] = e.what();
		this->writeObjResponse(packetId, 2, data);
	}
	catch (...)
	{
		std::cerr << "Unknown exception caught" << std::endl;
	}
	delete payloadRaw;
}

Tcp::Tcp(ICore *core)
{
	this->core = core;
	this->idCounter = 0;
	this->sockets = {};
}

std::shared_future<void> Tcp::run(int port)
{
	std::cerr << "starting tcp server..." << std::endl;
	return std::async(std::launch::async, [port, this]
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
				int clientSocket = accept(serverSocket, nullptr, nullptr);
				std::cerr << "new client connected." << std::endl;
				auto id = this->idCounter++;
				std::thread t([this, clientSocket, id]{
					this->handleConnection(id, clientSocket);
					this->sockets.erase(id);
					close(clientSocket);
				});
				t.detach();
	 		} })
		.share();
}

void Tcp::handleConnection(uint64_t connId, int conn)
{
	auto socket = new SocketItem{conn, {}, true, this->core};
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
				std::cerr << std::endl
						  << "keyhan " << oldReadCount << " " << length << " " << static_cast<int>(nextBuf[0]) << " " << static_cast<int>(nextBuf[1]) << " " << static_cast<int>(nextBuf[2]) << " " << static_cast<int>(nextBuf[3]) << " " << std::endl
						  << std::endl;
				memcpy(readData + oldReadCount, nextBuf, length - oldReadCount);
				memcpy(nextBuf, nextBuf + (readLength - (readCount - length)), readCount - length);
				remainedReadLength = (readCount - length);
				std::cerr << std::endl
						  << "keyhan -- " << static_cast<int>(readData[0]) << " " << static_cast<int>(readData[1]) << " " << static_cast<int>(readData[2]) << " " << static_cast<int>(readData[3]) << " " << std::endl
						  << std::endl;
				std::cerr << "packet received" << std::endl;
				char *packet = new char[length];
				memcpy(packet, readData, length);
				delete readData;
				std::cerr << std::endl
						  << "konstantin -- " << static_cast<int>(packet[0]) << " " << static_cast<int>(packet[1]) << " " << static_cast<int>(packet[2]) << " " << static_cast<int>(packet[3]) << " " << std::endl
						  << std::endl;

				std::thread t([&socket, length](char *packet)
							  {
								   try
								   {
									   socket->processPacket(packet, length);
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
