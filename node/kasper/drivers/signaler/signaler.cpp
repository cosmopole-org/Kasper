#pragma once

#include <any>
#include "../../core/trx/trx.h"
#include "../../core/core/icore.h"
#include <unordered_map>
#include <functional>
#include <string>
#include <vector>
#include <map>
#include <unordered_set>
#include "isignaler.h"
#include "../../utils/nlohmann/json.hpp"

using json = nlohmann::json;

class Signaler : public ISignaler
{
    ICore *core;
    std::unordered_map<std::string, std::function<void(std::string, std::any, size_t)>> listeners;

public:
    Signaler(ICore *core)
    {
        this->core = core;
        this->listeners = {};
    }

    std::function<void(std::string, std::any, size_t)> findListener(std::string userId)
    {
        if (auto lis = this->listeners.find(userId); lis != this->listeners.end())
        {
            return lis->second;
        }
        return NULL;
    }

    void listenToSingle(std::string userId, std::function<void(std::string, std::any, size_t)> listener) override
    {
        this->listeners.insert({userId, listener});
    }

    json createPoint(std::map<std::string, std::any> point) override
    {
        std::string pointId = std::any_cast<std::string>(point["id"]);
        std::string owner = std::any_cast<std::string>(point["owner"]);
        std::string persHist = std::string(1, std::any_cast<bool>(point["persHist"]) ? 0x01 : 0x02);
        std::string isPublic = std::string(1, std::any_cast<bool>(point["isPublic"]) ? 0x01 : 0x02);
        json j = {
            {"id", pointId},
            {"owner", owner},
            {"persHist", persHist},
            {"isPublic", isPublic},
        };
        this->core->modifyState([pointId, j](StateTrx *trx)
                                { trx->putObj("Point", pointId, j); });
        j["persHist"] = std::any_cast<bool>(point["persHist"]);
        j["isPublic"] = std::any_cast<bool>(point["isPublic"]);
        return j;
    }

    json getPoint(std::string pointId)
    {
        json j;
        this->core->modifyState([pointId, &j](StateTrx *trx)
                                { j = trx->getObjAsJson("Point", pointId); });
        j["persHist"] = j["persHist"].template get<std::string>().c_str()[0] == 0x01;
        j["isPublic"] = j["isPublic"].template get<std::string>().c_str()[0] == 0x01;
        return j;
    }

    void join(std::string userId, std::string pointId) override
    {
        this->core->modifyState([userId, pointId](StateTrx *trx)
                                {
            trx->putLink("ismember::" + userId + "::" + pointId, "true");
            trx->putLink("hasmember::" + pointId + "::" + userId, "true"); });
    }

    void leave(std::string userId, std::string pointId) override
    {
        this->core->modifyState([userId, pointId](StateTrx *trx)
                                {
            trx->delLink("ismember::" + userId + "::" + pointId);
            trx->delLink("hasmember::" + pointId + "::" + userId); });
    }

    void signalUserAsObj(std::string userId, std::string key, std::any data) override
    {
        if (auto listener = this->listeners.find(userId); listener != this->listeners.end())
        {
            listener->second(key, data, 0);
        }
    }

    void signalUserAsBytes(std::string userId, std::string key, char *data, size_t len) override
    {
        if (auto listener = this->listeners.find(userId); listener != this->listeners.end())
        {
            listener->second(key, data, len);
        }
    }

    void signalPointAsObj(std::string pointId, std::string key, std::any data, std::vector<std::string> exceptions) override
    {
        std::vector<std::string> userIds{};
        this->core->modifyState([pointId, &userIds](StateTrx *trx)
                                { userIds = trx->getLinksList("hasmember::" + pointId + "::"); });
        std::unordered_set<std::string> excepSet{};
        for (auto eId : exceptions)
        {
            excepSet.insert(eId);
        }
        for (auto userId : userIds)
        {
            if (excepSet.find(userId) == excepSet.end())
            {
                this->signalUserAsObj(userId, key, data);
            }
        }
    }

    void signalPointAsBytes(std::string pointId, std::string key, char *data, size_t len, std::vector<std::string> exceptions) override
    {
        std::vector<std::string> userIds{};
        this->core->modifyState([pointId, &userIds](StateTrx *trx)
                                { userIds = trx->getLinksList("hasmember::" + pointId + "::"); });
        std::unordered_set<std::string> excepSet{};
        for (auto eId : exceptions)
        {
            excepSet.insert(eId);
        }
        for (auto userId : userIds)
        {
            if (excepSet.find(userId) == excepSet.end())
            {
                this->signalUserAsBytes(userId, key, data, len);
            }
        }
    }
};
