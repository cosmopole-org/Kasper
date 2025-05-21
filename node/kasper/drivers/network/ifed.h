#pragma once

#include <cstdint>
#include <future>

class IFed
{
public:
    virtual ~IFed() = default;
    virtual std::shared_future<void> run(int port) = 0;

	virtual void request(std::string origin, std::string userId, std::string key, std::string payload, std::string signature, ActionInput input, std::function<void(int, std::string)> callback) = 0;
};
