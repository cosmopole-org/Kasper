#pragma once

#include "iactor.h"
#include "action.h"
#include "../../drivers/security/isecurity.h"
#include <functional>
#include <string>
#include <map>
#include <mutex>
#include <unordered_map>

class Actor : public IActor
{
    std::mutex lock;
    ISecurity *security;
    std::unordered_map<std::string, std::function<ActionOutput(StateHolder, ActionInput)>> store;
    std::unordered_map<std::string, ISecAction *> secStore;

public:
    Actor(ISecurity *security);
    std::function<ActionOutput(StateHolder, ActionInput)> findAction(std::string path) override;
    ISecAction *findActionAsSecure(std::string path) override;
    void insertAction(std::string path, Intelligence *intel, std::function<ActionOutput(StateHolder, ActionInput)> fn) override;
};
