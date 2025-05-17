#pragma once

#include "../../drivers/security/isecurity.h"
#include <functional>
#include <string>
#include <map>
#include <mutex>
#include <unordered_map>
#include "../trx/trx.h"
#include "../tools/itools.h"
#include "../../utils/nlohmann/json.hpp"
#include "../../drivers/file/datapack.h"

using json = nlohmann::json;

struct ActionOutput
{
    int resCode;
    json data;
    std::string err;
};

struct ActionInput
{
    json data;
};

class Meta
{
public:
    std::string pointId;
    std::string origin;
};

class Intelligence
{
public:
    virtual bool mustBeUser() = 0;
    virtual bool mustBeMember() = 0;
    virtual Meta extractMeta(json data) = 0;
};

struct StateHolder
{
public:
    std::string userId;
    std::string pointId;
    std::string origin;
    StateTrx *trx;
    ITools *tools;
};

class ISecAction
{
public:
    virtual ActionOutput run(std::string myOrigin, std::function<void(std::function<void(StateTrx *)>)> stateModifier, ITools *tools, std::string userId, DataPack payload, std::string signature) = 0;
};
