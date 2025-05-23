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
	virtual bool memorizeResponseBacked(std::string proof, std::string origin) = 0;
	virtual Event *getEventByProof(std::string proof) = 0;
	virtual int getOrderIndexOfEvent(std::string proof) = 0;
};
