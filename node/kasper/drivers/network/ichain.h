#pragma once

#include <cstdint>
#include <future>
#include "block.h"

class IChain
{
public:
	virtual ~IChain() = default;
	virtual void run(int port) = 0;
	virtual void submitTrx(std::string t, std::string data) = 0;
	virtual void addPendingEvent(Event *e) = 0;
	virtual bool addBackedProof(std::string proof, std::string origin, std::string signedProof) = 0;
	virtual void broadcastInShard(char *payload, uint32_t len) = 0;
	virtual void sendToShardMember(std::string origin, char *payload, uint32_t len) = 0;
	virtual bool memorizeResponseBacked(std::string proof, std::string origin) = 0;
	virtual Event *getEventByProof(std::string proof) = 0;
	virtual uint64_t getOrderIndexOfEvent(std::string proof) = 0;
	virtual void voteForNextEvent(std::string origin, std::string eventProof) = 0;
	virtual void pushNewElection() = 0;
	virtual void notifyElectorReady(std::string origin) = 0;
	virtual void handleConnection(std::string origin, int conn) = 0;
	virtual void removeConnection(std::string origin) = 0;
};