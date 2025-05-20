#pragma once

#include <cstdint>
#include <future>

class IFed
{
public:
    virtual ~IFed() = default;
    virtual std::shared_future<void> run(int port) = 0;
    virtual std::shared_future<void> request(std::string userId, std::string key, std::string payload, std::function<void(int, std::string, std::string)> callback) = 0;
    virtual std::shared_future<void> resonse(std::string key, std::string payload, std::string packetId) = 0;
    virtual std::shared_future<void> update(std::string targetId, std::string key, std::string payload) = 0;
};
