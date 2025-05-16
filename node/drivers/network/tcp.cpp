
#include <queue>
#include <mutex>
#include <condition_variable>
#include "../../core/core/core.cpp"
#include <string>
#include <future>
#include <unordered_map>
#include <cstring>
#include <iostream>
#include <netinet/in.h>
#include <sys/socket.h>
#include <unistd.h>
#include "../../utils/number.cpp"
#include "../../utils/nlohmann/json.hpp"
#include <any>
#include <optional>
#include "itcp.h"
#include "../../core/core/siglock.cpp"

using json = nlohmann::json;

template <typename T>
class SafeQueue
{
public:
	ThreadSafeQueue() = default;
	~ThreadSafeQueue() = default;

	// Disable copy
	ThreadSafeQueue(const ThreadSafeQueue &) = delete;
	SafeQueue &operator=(const SafeQueue &) = delete;

	// Add item to the queue
	void push(const T &item)
	{
		{
			std::lock_guard<std::mutex> lock(mutex_);
			queue_.push(item);
		}
		cond_var_.notify_one();
	}

	void push(T &&item)
	{
		{
			std::lock_guard<std::mutex> lock(mutex_);
			queue_.push(std::move(item));
		}
		cond_var_.notify_one();
	}

	// Wait and pop item from the queue
	T wait_and_pop()
	{
		std::unique_lock<std::mutex> lock(mutex_);
		cond_var_.wait(lock, [this]()
					   { return !queue_.empty(); });

		T item = std::move(queue_.front());
		queue_.pop();
		return item;
	}

	// Try to pop item without waiting
	std::optional<T> try_pop()
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

	// Check if queue is empty
	bool empty() const
	{
		std::lock_guard<std::mutex> lock(mutex_);
		return queue_.empty();
	}

	// Get size (not always useful in multithreaded context)
	size_t size() const
	{
		std::lock_guard<std::mutex> lock(mutex_);
		return queue_.size();
	}

	// Get top item
	ValPack peek() const
	{
		std::lock_guard<std::mutex> lock(mutex_);
		return queue_.front();
	}

private:
	mutable std::mutex mutex_;
	std::condition_variable cond_var_;
	std::queue<T> queue_;
};

class Socket
{
public:
	int conn;
	SafeQueue<ValPack> buffer;
	bool ack;
	ICore *core;
	std::mutex lock;
	void processPacket(char *packet, uint32_t len);
	void writeRawUpdate(std::string key, char *updatePack, uint32_t len)
	{
		printf("preparing update...\n");

		const char *keyBytes = key.c_str();
		auto keyBytesLen = convertIntToData(key.size()).data();

		auto packet = new char[1 + sizeof(keyBytesLen) + sizeof(keyBytes) + len];
		uint32_t pointer = 1;

		packet[0] = 0x01;

		memcpy(packet + pointer, keyBytesLen, sizeof(keyBytesLen));
		pointer += sizeof(keyBytesLen);
		memcpy(packet + pointer, keyBytes, sizeof(keyBytes));
		pointer += sizeof(keyBytes);
		memcpy(packet + pointer, updatePack, len);
		pointer += len;

		printf("appending to buffer...\n");

		std::lock_guard<std::mutex> lock(this->lock);
		this->buffer.push({packet, sizeof(packet)});
		this->pushBuffer();
	}

	void writeObjUpdate(std::string key, json updatePack)
	{
		std::string data = updatePack.dump();
		this->writeRawUpdate(key, &data[0], data.size());
	}

	void writeRawResponse(std::string requestId, int resCode, char *response, uint32_t len)
	{
		printf("preparing response...\n");

		const char *b1 = requestId.c_str();
		char *b1Len = convertIntToData(requestId.size()).data();

		char *b2 = convertIntToData(resCode).data();

		char *packet = new char(1 + sizeof(b1Len) + sizeof(b1) + sizeof(b2) + len);

		uint32_t pointer = 1;

		packet[0] = 0x02;

		memcpy(packet + pointer, b1Len, sizeof(b1Len));
		pointer += sizeof(b1Len);
		memcpy(packet + pointer, b1, sizeof(b1));
		pointer += sizeof(b1);
		memcpy(packet + pointer, b2, sizeof(b2));
		pointer += sizeof(b2);
		memcpy(packet + pointer, response, len);
		pointer += len;

		printf("appending to buffer...\n");

		std::lock_guard<std::mutex> lock(this->lock);
		this->buffer.push({packet, sizeof(packet)});
		this->pushBuffer();
	}

	void writeObjResponse(std::string requestId, int resCode, json response)
	{
		std::string data = response.dump();
		this->writeRawResponse(requestId, resCode, &data[0], data.size());
	}

	void pushBuffer()
	{
		if (this->ack)
		{
			if (this->buffer.size() > 0)
			{
				this->ack = false;
				auto data = this->buffer.peek();
				char *packetLen = convertIntToData(data.len).data();
				auto res = send(this->conn, packetLen, sizeof(packetLen), 0);
				if (res == -1)
				{
					this->ack = true;
					printf("error writing to socket.\n");
				}
				send(this->conn, data.data, data.len, 0);
				if (res == -1)
				{
					this->ack = true;
					printf("error writing to socket.\n");
				}
			}
		}
	}

	std::function<void(std::string, std::any, size_t)> Socket::connectListener(std::string uid)
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

	void processPacket(char *packet, uint32_t len)
	{
		if ((len == sizeof("packet_received")) && (std::string(packet, len) == "packet_received"))
		{
			auto send = [this]()
			{
				std::lock_guard<std::mutex> lock(this->lock);
				this->ack = true;
				if (this->buffer.size() > 0)
				{
					this->buffer.try_pop();
					this->pushBuffer();
				}
			};
			send();
			return;
		}
		uint32_t pointer = 0;
		char *tempBytes = new char[4];
		memcpy(tempBytes, packet + pointer, 4);
		uint32_t signatureLength = parseDataAsInt(tempBytes);
		delete tempBytes;
		printf("signature length: %d\n", signatureLength);
		pointer += 4;
		char *signature = new char[signatureLength];
		memcpy(signature, packet + pointer, signatureLength);
		pointer += signatureLength;
		printf("signature: %s\n", signature);

		char *tempBytes3 = new char[4];
		memcpy(tempBytes3, packet + pointer, 4);
		uint32_t userIdLength = parseDataAsInt(tempBytes3);
		delete tempBytes3;
		printf("userId length: %d\n", userIdLength);
		pointer += 4;
		char *tempBytes4 = new char[userIdLength];
		memcpy(tempBytes4, packet + pointer, userIdLength);
		std::string userId = std::string(tempBytes4);
		delete tempBytes4;
		pointer += userIdLength;
		printf("userId: %s\n", userId);

		char *tempBytes5 = new char[4];
		memcpy(tempBytes5, packet + pointer, 4);
		uint32_t pathLength = parseDataAsInt(tempBytes5);
		delete tempBytes5;
		printf("path length: %d\n", pathLength);
		pointer += 4;
		char *tempBytes6 = new char[pathLength];
		memcpy(tempBytes6, packet + pointer, pathLength);
		std::string path = std::string(tempBytes6);
		delete tempBytes6;
		pointer += pathLength;
		printf("path: %s\n", path);

		char *tempBytes7 = new char[4];
		memcpy(tempBytes7, packet + pointer, 4);
		uint32_t packetIdLength = parseDataAsInt(tempBytes7);
		delete tempBytes7;
		printf("packetId length: %d\n", packetIdLength);
		pointer += 4;
		char *tempBytes8 = new char[packetIdLength];
		memcpy(tempBytes8, packet + pointer, packetIdLength);
		std::string packetId = std::string(tempBytes8);
		delete tempBytes8;
		pointer += packetIdLength;
		printf("packetId: %s\n", packetId);

		char *payload = new char[len - pointer];
		memcpy(payload, packet + pointer, len - pointer);
		printf(payload);

		try
		{
			if (path == "authenticate")
			{
				std::function<void(std::string, std::any, size_t)> lis;
				auto checkRes = this->core->getTools()->getSecurity()->authWithSignature(userId, payload, &signature[0]);
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
				return;
			}
			auto action = this->core->getActor()->findActionAsSecure(path);
			if (action == NULL)
			{
				json res;
				res["message"] = "action not found";
				this->writeObjResponse(packetId, 1, res);
				return;
			}
			auto response = action->run(this->core, userId, payload, signature);
			if (response.err != "")
			{
				json data;
				data["message"] = response.err;
				this->writeObjResponse(packetId, response.resCode, data);
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
	}
};

class Tcp : public ITcp
{
	ICore *core;
	std::unordered_map<uint64_t, Socket *> sockets;
	uint64_t idCounter;

public:
	Tcp(ICore *core)
	{
		this->core = core;
		this->idCounter = 0;
		this->sockets = {};
	}

	void run(int port) override
	{
		std::async(std::launch::async, [port, this]
				   {
			int serverSocket = socket(AF_INET, SOCK_STREAM, 0);
			sockaddr_in serverAddress;
			serverAddress.sin_family = AF_INET;
			serverAddress.sin_port = htons(port);
			serverAddress.sin_addr.s_addr = INADDR_ANY;
			bind(serverSocket, (struct sockaddr*)&serverAddress,
				 sizeof(serverAddress));
			listen(serverSocket, 5);
			int clientSocket = accept(serverSocket, nullptr, nullptr);
			auto id = this->idCounter++;
			std::async(std::launch::async, [this, clientSocket, id]{
				this->handleConnection(id, clientSocket);
				this->sockets.erase(id);
				close(clientSocket);
			}); });
	}

	void handleConnection(uint64_t connId, int conn) override
	{
		auto socket = new Socket(conn, {}, true, this->core);
		this->sockets.insert({connId, socket});
		char lenBuf[4];
		char buf[1024];
		uint64_t readCount = 0;
		uint64_t oldReadCount = 0;
		while (true)
		{
			recv(conn, lenBuf, sizeof(lenBuf), 0);
			auto length = parseDataAsInt(lenBuf);
			char *readData = new char[length];
			while (true)
			{
				auto readLength = recv(conn, buf, sizeof(buf), 0);
				if (readLength == -1)
				{
					printf("socket error\n");
					return;
				}
				oldReadCount = readCount;
				readCount += readLength;
				if (readCount >= length)
				{
					uint64_t j = 0;
					for (uint64_t i = oldReadCount; i < sizeof(readData); i++)
					{
						readData[i] = buf[j];
						j++;
					}
					j = readLength - (readCount - length);
					for (uint64_t i = 0; i < readCount - length; i++)
					{
						buf[i];
						j++;
					}
					printf("packet received\n");
					printf(readData);
					std::async(std::launch::async, [&socket, &readData, length]
							   {
								   try
								   {
									   socket->processPacket(readData, length);
								   }
								   catch (const std::exception &e)
								   {
									   std::cerr << "Standard exception caught: " << e.what() << std::endl;
								   }
								   catch (...)
								   {
									   std::cerr << "Unknown exception caught" << std::endl;
								   }
							   });
					readCount -= length;
					break;
				}
				else
				{
					uint64_t j = 0;
					for (uint64_t i = oldReadCount; i < readCount; i++)
					{
						readData[i] = buf[j];
						j++;
					}
				}
			}
		}
	}
};
