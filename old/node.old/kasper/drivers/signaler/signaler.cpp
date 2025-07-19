#include "signaler.h"

using json = nlohmann::json;

Signaler::Signaler(ICore *core)
{
    this->core = core;
    this->listeners = {};
}

std::function<void(std::string, std::any, size_t)> Signaler::findListener(std::string userId)
{
    if (auto lis = this->listeners.find(userId); lis != this->listeners.end())
    {
        return lis->second;
    }
    return NULL;
}

void Signaler::listenToSingle(std::string userId, std::function<void(std::string, std::any, size_t)> listener)
{
    this->listeners.insert({userId, listener});
}

json Signaler::createPoint(std::map<std::string, std::any> point)
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

json Signaler::getPoint(std::string pointId)
{
    json j;
    this->core->modifyState([pointId, &j](StateTrx *trx)
                            { j = trx->getObjAsJson("Point", pointId); });
    j["persHist"] = j["persHist"].template get<std::string>().c_str()[0] == 0x01;
    j["isPublic"] = j["isPublic"].template get<std::string>().c_str()[0] == 0x01;
    return j;
}

void Signaler::join(std::string userId, std::string pointId)
{
    this->core->modifyState([userId, pointId](StateTrx *trx)
                            {
            trx->putLink("ismember::" + userId + "::" + pointId, "true");
            trx->putLink("hasmember::" + pointId + "::" + userId, "true"); });
}

void Signaler::leave(std::string userId, std::string pointId)
{
    this->core->modifyState([userId, pointId](StateTrx *trx)
                            {
            trx->delLink("ismember::" + userId + "::" + pointId);
            trx->delLink("hasmember::" + pointId + "::" + userId); });
}

void Signaler::signalUserAsObj(std::string userId, std::string key, std::any data)
{
    if (auto listener = this->listeners.find(userId); listener != this->listeners.end())
    {
        listener->second(key, data, 0);
    }
}

void Signaler::signalUserAsBytes(std::string userId, std::string key, char *data, size_t len)
{
    if (auto listener = this->listeners.find(userId); listener != this->listeners.end())
    {
        listener->second(key, data, len);
    }
}

void Signaler::signalPointAsObj(std::string pointId, std::string key, std::any data, std::vector<std::string> exceptions)
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

void Signaler::signalPointAsBytes(std::string pointId, std::string key, char *data, size_t len, std::vector<std::string> exceptions)
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
