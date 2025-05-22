#pragma once

#include <cstdint>
#include <future>
#include <string>
#include "../../utils/nlohmann/json.hpp"

using json = nlohmann::json;

class ISocketItem
{
public:
    virtual void writeRawUpdate(std::string key, char *updatePack, uint32_t len) = 0;
    virtual void writeObjUpdate(std::string key, json updatePack) = 0;
    virtual void writeRawResponse(std::string requestId, int resCode, char *response, uint32_t len) = 0;
    virtual void writeObjResponse(std::string requestId, int resCode, json response) = 0;
};

class ITcp
{
public:
    virtual ~ITcp() = default;
    virtual void run(int port) = 0;
};
