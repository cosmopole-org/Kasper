#pragma once

#include <cstdint>
#include <future>
#include "../../core/core/actionio.h"
#include "request.h"

class IFed
{
public:
    virtual ~IFed() = default;
    virtual void run(int port) = 0;
    virtual Request* findRequest(std::string requestId) = 0;
    virtual void clearRequest(std::string requestId) = 0;
	virtual void request(std::string origin, std::string userId, std::string key, std::string payload, std::string signature, ActionInput input, std::function<void(int, std::string)> callback) = 0;
};
