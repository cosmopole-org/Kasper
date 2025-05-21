#pragma once

#include "../../drivers/security/isecurity.h"
#include "../../drivers/network/ifed.h"
#include "../../drivers/network/itcp.h"
#include <functional>
#include <string>
#include <map>
#include <mutex>
#include <unordered_map>
#include "../tools/itools.h"
#include "iaction.h"
#include "icore.h"

class SecAction : public ISecAction
{
    Intelligence *intel;
    ISecurity *security;
    std::string key;
    IFed *federation;
    std::function<ActionOutput(StateHolder, ActionInput)> fn;

public:
    SecAction(ISecurity *security, IFed *federation, std::string key, Intelligence *intel, std::function<ActionOutput(StateHolder, ActionInput)> fn);
    Intelligence *getIntel() override;
    ActionOutput run(ISocketItem *socket, std::string myOrigin, std::function<void(std::function<void(StateTrx *)>)> stateModifier, ITools *tools, std::string userId, DataPack payload, std::string signature, std::string requestId) override;
};
