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
    Signaler(ICore *core);
    std::function<void(std::string, std::any, size_t)> findListener(std::string userId) override;
    void listenToSingle(std::string userId, std::function<void(std::string, std::any, size_t)> listener) override;
    json createPoint(std::map<std::string, std::any> point) override;
    json getPoint(std::string pointId) override;
    void join(std::string userId, std::string pointId) override;
    void leave(std::string userId, std::string pointId) override;
    void signalUserAsObj(std::string userId, std::string key, std::any data) override;
    void signalUserAsBytes(std::string userId, std::string key, char *data, size_t len) override;
    void signalPointAsObj(std::string pointId, std::string key, std::any data, std::vector<std::string> exceptions) override;
    void signalPointAsBytes(std::string pointId, std::string key, char *data, size_t len, std::vector<std::string> exceptions) override;
};
