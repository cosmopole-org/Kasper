#pragma once

#include "../../drivers/security/isecurity.h"
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
    std::function<ActionOutput(StateHolder, ActionInput)> fn;

public:
    SecAction(ISecurity *security, Intelligence *intel, std::function<ActionOutput(StateHolder, ActionInput)> fn);
    ActionOutput run(std::string myOrigin, std::function<void(std::function<void(StateTrx *)>)> stateModifier, ITools *tools, std::string userId, DataPack payload, std::string signature);
};
