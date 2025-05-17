#pragma once

#include <any>
#include "../../core/trx/trx.h"
#include "../../core/core/icore.h"
#include <unordered_map>
#include <functional>
#include <string>
#include <vector>
#include <unordered_set>
#include "isignaler.h"

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

    std::function<void(std::string, std::any, size_t)> findListener(std::string userId) {
        if (auto lis = this->listeners.find(userId); lis != this->listeners.end()) {
            return lis->second;
        }
        return NULL;
    }

    void listenToSingle(std::string userId, std::function<void(std::string, std::any, size_t)> listener) override
    {
        this->listeners.insert({userId, listener});
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
