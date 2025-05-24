#include "chain.h"

#define SA struct sockaddr

using json = nlohmann::json;

void ChainSocketItem::processPacket(std::string origin, char *packet, uint32_t len)
{
	std::cerr << "received packet length: " << len << std::endl;

	uint32_t pointer = 1;

	if (packet[0] == 0x01)
	{
		std::string signature = "";
		std::string data = "";

		std::cerr << "received consensus packet phase 1" << std::endl;

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

		char *tempBytes2 = new char[4];
		memcpy(tempBytes2, packet + pointer, 4);
		uint32_t dataLength = Utils::getInstance().parseDataAsInt(tempBytes2);
		delete tempBytes2;
		std::cerr << "data length: " << dataLength << std::endl;
		pointer += 4;
		if (dataLength > 0)
		{
			char *da = new char[dataLength];
			memcpy(da, packet + pointer, dataLength);
			data = std::string(da, dataLength);
			delete da;
			pointer += dataLength;
		}
		std::cerr << "data: " << data << std::endl;

		json eventObj = json::parse(data);
		json dataObj = eventObj["trxs"];
		std::string claimedOrigin = eventObj["origin"].template get<std::string>();
		if (claimedOrigin != origin)
		{
			return;
		}

		std::vector<std::pair<std::string, std::string>> trxs{};
		for (json::iterator item = dataObj.begin(); item != dataObj.end(); ++item)
		{
			trxs.push_back({item.value()["type"].template get<std::string>(), item.value()["data"].template get<std::string>()});
		}
		Event *e = new Event{claimedOrigin, trxs, eventObj["proof"].template get<std::string>(), {}};
		this->chain->addPendingEvent(e);
		std::string proofSign = this->core->signPacket(e->proof);
		const char *proofSignBytes = proofSign.c_str();
		uint32_t proofSignLen = proofSign.size();
		char *proofSignLenBytes = Utils::getInstance().convertIntToData(proofSignLen);
		const char *proofBytes = e->proof.c_str();
		uint32_t proofLen = e->proof.size();
		char *proofLenBytes = Utils::getInstance().convertIntToData(proofLen);

		uint32_t updateLen = 1 + 4 + proofSignLen + 4 + proofLen;

		char *update = new char[updateLen];
		uint32_t pointer = 1;
		update[0] = 0x04;
		memcpy(update + pointer, proofSignLenBytes, 4);
		pointer += 4;
		memcpy(update + pointer, proofSignBytes, proofSignLen);
		pointer += proofSignLen;
		memcpy(update + pointer, proofLenBytes, 4);
		pointer += 4;
		memcpy(update + pointer, proofBytes, proofLen);
		pointer += proofLen;
		e->myUpdate = std::string(update, updateLen);

		delete update;

		uint32_t responseLen = 1 + 4 + proofLen;
		char *response = new char[responseLen];
		response[0] = 0x02;
		pointer = 1;
		memcpy(response + pointer, proofLenBytes, 4);
		pointer += 4;
		memcpy(response + pointer, proofBytes, proofLen);
		pointer += proofLen;
		char *responseLenBytes = Utils::getInstance().convertIntToData(responseLen);
		send(this->conn, responseLenBytes, 4, 0);
		send(this->conn, response, 1, 0);

		delete response;
		delete responseLenBytes;
	}
	else if (packet[0] == 0x02)
	{
		std::string proof = "";

		std::cerr << "received consensus packet phase 2" << std::endl;

		char *tempBytes2 = new char[4];
		memcpy(tempBytes2, packet + pointer, 4);
		uint32_t dataLength = Utils::getInstance().parseDataAsInt(tempBytes2);
		delete tempBytes2;
		std::cerr << "data length: " << dataLength << std::endl;
		pointer += 4;
		if (dataLength > 0)
		{
			char *da = new char[dataLength];
			memcpy(da, packet + pointer, dataLength);
			proof = std::string(da, dataLength);
			delete da;
			pointer += dataLength;
		}
		std::cerr << "proof: " << proof << std::endl;

		auto done = this->chain->memorizeResponseBacked(proof, origin);

		if (!done)
		{
			return;
		}

		char *proofLenBytes = Utils::getInstance().convertIntToData(proof.size());
		const char *proofBytes = proof.c_str();

		uint32_t reqLen = 1 + 4 + proof.size();
		char *req = new char[reqLen];
		uint32_t pointer = 1;
		req[0] = 0x03;
		memcpy(req + pointer, proofLenBytes, 4);
		pointer += 4;
		memcpy(req + pointer, proofBytes, proof.size());
		pointer += proof.size();

		this->chain->broadcastInShard(req, reqLen);
		delete req;
	}
	else if (packet[0] == 0x03)
	{
		std::string proof = "";

		std::cerr << "received consensus packet phase 3" << std::endl;

		char *tempBytes2 = new char[4];
		memcpy(tempBytes2, packet + pointer, 4);
		uint32_t dataLength = Utils::getInstance().parseDataAsInt(tempBytes2);
		delete tempBytes2;
		std::cerr << "data length: " << dataLength << std::endl;
		pointer += 4;
		if (dataLength > 0)
		{
			char *da = new char[dataLength];
			memcpy(da, packet + pointer, dataLength);
			proof = std::string(da, dataLength);
			delete da;
			pointer += dataLength;
		}
		std::cerr << "proof: " << proof << std::endl;

		this->chain->pushNewElection();
	}
	else if (packet[0] == 0x04)
	{
		std::string proof = "";

		std::cerr << "received consensus packet phase 4" << std::endl;

		std::string signature = "";
		std::string proof = "";

		char *tempBytes = new char[4];
		memcpy(tempBytes, packet + pointer, 4);
		pointer += 4;
		uint32_t signLength = Utils::getInstance().parseDataAsInt(tempBytes);
		delete tempBytes;
		std::cerr << "signature length: " << signLength << std::endl;
		if (signLength > 0)
		{
			char *da = new char[signLength];
			memcpy(da, packet + pointer, signLength);
			signature = std::string(da, signLength);
			delete da;
			pointer += signLength;
		}
		std::cerr << "signature: " << signature << std::endl;

		char *tempBytes2 = new char[4];
		memcpy(tempBytes2, packet + pointer, 4);
		pointer += 4;
		uint32_t dataLength = Utils::getInstance().parseDataAsInt(tempBytes2);
		delete tempBytes2;
		std::cerr << "data length: " << dataLength << std::endl;
		if (dataLength > 0)
		{
			char *da = new char[dataLength];
			memcpy(da, packet + pointer, dataLength);
			proof = std::string(da, dataLength);
			delete da;
			pointer += dataLength;
		}
		std::cerr << "proof: " << proof << std::endl;

		this->chain->pushToEventQueue(this->chain->getEventByProof(proof));
	}
	else if (packet[0] == 0x05)
	{
		std::cerr << "received consensus packet phase 5" << std::endl;

		std::string signature = "";
		std::string vote = "";

		char *tempBytes = new char[4];
		memcpy(tempBytes, packet + pointer, 4);
		pointer += 4;
		uint32_t signLength = Utils::getInstance().parseDataAsInt(tempBytes);
		delete tempBytes;
		std::cerr << "signature length: " << signLength << std::endl;
		if (signLength > 0)
		{
			char *da = new char[signLength];
			memcpy(da, packet + pointer, signLength);
			signature = std::string(da, signLength);
			delete da;
			pointer += signLength;
		}
		std::cerr << "signature: " << signature << std::endl;

		char *tempBytes2 = new char[4];
		memcpy(tempBytes2, packet + pointer, 4);
		pointer += 4;
		uint32_t dataLength = Utils::getInstance().parseDataAsInt(tempBytes2);
		delete tempBytes2;
		std::cerr << "data length: " << dataLength << std::endl;
		if (dataLength > 0)
		{
			char *da = new char[dataLength];
			memcpy(da, packet + pointer, dataLength);
			vote = std::string(da, dataLength);
			delete da;
			pointer += dataLength;
		}
		std::cerr << "vote: " << vote << std::endl;

		this->chain->voteForNextEvent(origin, vote);
	}
}

ChainSocketItem::ChainSocketItem(IChain *chain, int conn, ICore *core)
{
	this->chain = chain;
	this->conn = conn;
	this->core = core;
	this->ack = true;
}

Chain::Chain(ICore *core)
{
	this->core = core;
	this->blocks = {};
	this->shardPeers = {};
	this->pendingEvents = {};
}

void Chain::pushNewElection()
{
	std::lock_guard<std::mutex> lock(this->lock);
	this->pendingBlockElections++;
}

void Chain::voteForNextEvent(std::string origin, std::string eventProof)
{
	Event *choosenEvent = NULL;
	bool done = false;
	{
		std::lock_guard<std::mutex> lock(this->lock);
		this->nextEventVotes[origin] = eventProof;
		if (this->nextEventVotes.size() == this->shardPeers.size())
		{
			std::unordered_map<std::string, int> votes{};
			for (auto vote : this->nextEventVotes)
			{
				if (votes.find(vote.second) == votes.end())
				{
					votes[vote.second] = 1;
				}
				else
				{
					votes[vote.second] = votes[vote.second] + 1;
				}
			}
			std::vector<std::pair<std::string, int>> sortedArr{};
			for (auto item : votes)
			{
				sortedArr.push_back({item.first, item.second});
			}
			std::sort(sortedArr.begin(), sortedArr.end(), [](const std::pair<std::string, int> &a, const std::pair<std::string, int> &b)
					  { return a.second > b.second; });
			this->nextEventVotes.clear();

			choosenEvent = this->getEventByProof(sortedArr[0].first);
			done = true;
		}
	}
	if (done)
	{
		{
			std::lock_guard<std::mutex> lock(this->lock);
			this->nextBlockQueue.push(new Block{choosenEvent->trxs});
		}
		{
			std::lock_guard<std::mutex> lock(mtx);
			ready = true;
		}
		this->cond_var_.notify_one();
	}
}

uint64_t Chain::getOrderIndexOfEvent(std::string proof)
{
	std::lock_guard<std::mutex> lock(this->lock);
	uint64_t index = 0;
	uint64_t count = this->pendingEvents.size();
	for (auto i = this->pendingEvents.rbegin(); i != this->pendingEvents.rend(); ++i)
	{
		if ((*i)->proof == proof)
		{
			return count - index - 1;
		}
		index++;
	}
	return std::numeric_limits<uint64_t>::max();
}

Event *Chain::getEventByProof(std::string proof)
{
	return this->proofEvents[proof];
}

void Chain::pushToEventQueue(Event *e)
{
	return this->nextEventsQueue.push(e);
}

void Chain::run(int port)
{
	std::cerr << "starting chain server on port " << port << "..." << std::endl;
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

				std::thread t([this, clientSocket, origin]{
					this->handleConnection(origin, clientSocket);
					this->shardPeers.erase(origin);
					close(clientSocket);
				});
				t.detach();
	 		} });
	t.detach();
	std::thread t2([port, this]
				   {
					while (true)
					{
						if (this->pendingTrxs.size() > 0) {
							auto now = std::chrono::system_clock::now();
   							auto duration = now.time_since_epoch();
    						auto microseconds = std::chrono::duration_cast<std::chrono::microseconds>(duration).count();
							std::string proof = std::to_string(microseconds);
							std::lock_guard<std::mutex> lock(this->lock);
							Event *e = new Event{this->core->getIp(), this->pendingTrxs, proof, {}};
							this->pendingTrxs.clear();
							this->pendingEvents.push_back(e);
							this->proofEvents[e->proof] = e;
							json trxsJson;
							for (auto trx : e->trxs) {
								json trxJson;
								trxJson["type"] = trx.first;
								trxJson["data"] = trx.second;
								trxsJson.push_back(trxJson);
							}
							json eventJson;
							eventJson["trxs"] = trxsJson;
							eventJson["proof"] = e->proof;
							std::string dataStr = eventJson.dump();
							std::string signature = this->core->signPacket(dataStr);
							const char* dataBytes = dataStr.c_str();
							size_t dataLen = dataStr.size();
							char* dataLenBytes = Utils::getInstance().convertIntToData(dataLen);
							const char* signBytes = signature.c_str();
							size_t signLen = signature.size();
							char* signLenBytes = Utils::getInstance().convertIntToData(signLen);
							size_t payloadLen = 1 + 4 + signLen + 4 + dataLen;
							char* payload = new char[payloadLen];
							uint32_t pointer = 1;
							payload[0] = 0x01;
							memcpy(payload + pointer, signLenBytes, 4);
							pointer += 4;
							memcpy(payload + pointer, signBytes, signLen);
							pointer += signLen;
							memcpy(payload + pointer, dataLenBytes, 4);
							pointer += 4;
							memcpy(payload + pointer, dataBytes, dataLen);
							pointer += dataLen;
							char* payloadLenBytes = Utils::getInstance().convertIntToData(payloadLen);
							for (auto shardPeer : this->shardPeers) {
								send(shardPeer.second->conn, payloadLenBytes, 4, 0);
								send(shardPeer.second->conn, payload, payloadLen, 0);
							}
							delete payloadLenBytes;
							delete payload;
							delete signLenBytes;
							delete dataLenBytes;
						}
						std::this_thread::sleep_for(std::chrono::milliseconds(100));
					} });
	t2.detach();

	std::thread t3([this]
				   {
		 while (true)
		 {
			Event *e = this->nextEventsQueue.wait_and_pop();

			std::string signature = this->core->signPacket(e->proof);
			const char *proofBytes = e->proof.c_str();
			char *proofLenBytes = Utils::getInstance().convertIntToData(e->proof.size());
			const char *signBytes = signature.c_str();
			char *signLenBytes = Utils::getInstance().convertIntToData(signature.size());
			uint32_t updateLen = 1 + 4 + e->proof.size() + 4 + signature.size();
			char *orderUpdate = new char[updateLen];
			uint32_t pointer = 1;
			orderUpdate[0] = 0x05;
			memcpy(orderUpdate + pointer, proofLenBytes, 4);
			pointer += 4;
			memcpy(orderUpdate + pointer, proofBytes, e->proof.size());
			pointer += e->proof.size();
			memcpy(orderUpdate + pointer, signLenBytes, 4);
			pointer += 4;
			memcpy(orderUpdate + pointer, signBytes, signature.size());
			pointer += signature.size();

			this->broadcastInShard(orderUpdate, updateLen);

			delete orderUpdate;
			delete signLenBytes;
			delete proofLenBytes;
			
			std::unique_lock<std::mutex> lock(mtx);
			this->cond_var_.wait(lock, [this]{ return this->ready; });
			{
				std::lock_guard<std::mutex> lock(mtx);
				ready = false;
			}
		 } });
	t3.detach();

	std::thread t4([this]
				   {
			Block *block = this->nextBlockQueue.wait_and_pop();
			{
				std::lock_guard<std::mutex> lock(this->lock);
				this->blocks.push_back(block);
			}
			for (auto trx : block->trxs) {
				std::cerr << "received transaction: " << trx.first << " " << trx.second << std::endl;
			} });
	t4.detach();

	std::thread t5([this]
				   {
				while (true)
				{
					{
						std::lock_guard<std::mutex> lock(this->lock);
						this->pendingBlockElections--;
					}
					
					auto e = this.events
					{
						std::lock_guard<std::mutex> lock(this->lock);
						this->voteForNextEvent(this->core->getIp(), proof);
					}
					this->broadcastInShard((char *)e->myUpdate.c_str(), e->myUpdate.size());

					std::unique_lock<std::mutex> lock(mtx);
					this->cond_var_.wait(lock, [this]{ return this->ready; });
					{
						std::lock_guard<std::mutex> lock(mtx);
						ready = false;
					}
				}
   				   });
	t5.detach();
}

void Chain::submitTrx(std::string t, std::string data)
{
	std::lock_guard<std::mutex> lock(this->lock);
	this->pendingTrxs.push_back({t, data});
}

void Chain::addPendingEvent(Event *e)
{
	std::lock_guard<std::mutex> lock(this->lock);
	this->pendingEvents.push_back(e);
}

void Chain::broadcastInShard(char *payload, uint32_t len)
{
	char *payloadLenBytes = Utils::getInstance().convertIntToData(len);
	std::lock_guard<std::mutex> lock(this->lock);
	for (auto s : this->shardPeers)
	{
		send(s.second->conn, payloadLenBytes, 4, 0);
		send(s.second->conn, payload, len, 0);
	}
	delete payloadLenBytes;
}

void Chain::sendToShardMember(std::string origin, char *payload, uint32_t len)
{
	char *payloadLenBytes = Utils::getInstance().convertIntToData(len);
	std::lock_guard<std::mutex> lock(this->lock);
	auto s = this->shardPeers[origin];
	send(s->conn, payloadLenBytes, 4, 0);
	send(s->conn, payload, len, 0);
	delete payloadLenBytes;
}

bool Chain::memorizeResponseBacked(std::string proof, std::string origin)
{
	std::lock_guard<std::mutex> lock(this->lock);
	if (auto e = this->proofEvents.find(proof); e != this->proofEvents.end())
	{
		e->second->backedResponses.insert(origin);
		if (e->second->backedResponses.size() == (this->shardPeers.size() - 1))
		{
			return true;
		}
	}
	return false;
}

bool Chain::addBackedProof(std::string proof, std::string origin, std::string signedProof)
{
	EVP_PKEY *pkey;
	{
		std::lock_guard<std::mutex> lock(this->lock);
		pkey = this->shardPeers[origin]->pkey;
	}
	if (Utils::getInstance().verify_signature_rsa(pkey, proof, signedProof))
	{
		std::lock_guard<std::mutex> lock(this->lock);
		if (auto e = this->proofEvents.find(proof); e != this->proofEvents.end())
		{
			e->second->backedProofs[origin] = signedProof;
			if ((e->second->backedProofs.size() == (this->shardPeers.size() - 2)))
			{
				if (auto b = e->second->backedProofs.find(e->second->origin); b == e->second->backedProofs.end())
				{
					return true;
				}
			}
		}
	}
	return false;
}

void Chain::handleConnection(std::string origin, int conn)
{
	auto socket = new ChainSocketItem(this, conn, this->core);
	this->shardPeers.insert({origin, socket});
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
