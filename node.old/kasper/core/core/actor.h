#pragma once

#include "iactor.h"
#include "action.h"
#include "actionio.h"
#include "../../drivers/security/isecurity.h"
#include "../../drivers/network/ifed.h"
#include <functional>
#include <string>
#include <map>
#include <mutex>
#include <unordered_map>

class Actor : public IActor
{
    std::mutex lock;
    ISecurity *security;
    IFed *federation;
    std::unordered_map<std::string, std::function<ActionOutput(StateHolder, ActionInput)>> store;
    std::unordered_map<std::string, ISecAction *> secStore;

public:
    Actor(ISecurity *security, IFed *federation);
    std::function<ActionOutput(StateHolder, ActionInput)> findAction(std::string path) override;
    ISecAction *findActionAsSecure(std::string path) override;
    void insertAction(std::string path, Intelligence *intel, std::function<ActionOutput(StateHolder, ActionInput)> fn) override;
};
