#include "iactor.h"
#include "action.cpp"
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
    std::unordered_map<std::string, std::function<ActionOutput(State, ActionInput)>> store;
    std::unordered_map<std::string, ISecAction *> secStore;

public:
    Actor(ISecurity *security)
    {
        this->security = security;
        this->store = {};
        this->secStore = {};
    }

    std::function<ActionOutput(State, ActionInput)> findAction(std::string path) override
    {
        std::lock_guard<std::mutex> lock(this->lock);
        if (auto fn = this->store.find(path); fn != this->store.end())
        {
            return fn->second;
        }
        return NULL;
    }

    ISecAction *findActionAsSecure(std::string path) override
    {
        std::lock_guard<std::mutex> lock(this->lock);
        if (auto fn = this->secStore.find(path); fn != this->secStore.end())
        {
            return fn->second;
        }
        return NULL;
    }

    void insertAction(std::string path, Intelligence *intel, std::function<ActionOutput(State, ActionInput)> fn) override
    {
        std::lock_guard<std::mutex> lock(this->lock);
        this->store.insert({path, fn});
        this->secStore.insert({path, new SecAction(this->security, intel, fn)});
    }
};
