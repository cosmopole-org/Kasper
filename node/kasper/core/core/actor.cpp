#include "actor.h"

Actor::Actor(ISecurity *security)
{
    this->security = security;
    this->store = {};
    this->secStore = {};
}

std::function<ActionOutput(StateHolder, ActionInput)> Actor::findAction(std::string path)
{
    std::lock_guard<std::mutex> lock(this->lock);
    if (auto fn = this->store.find(path); fn != this->store.end())
    {
        return fn->second;
    }
    return NULL;
}

ISecAction *Actor::findActionAsSecure(std::string path)
{
    std::lock_guard<std::mutex> lock(this->lock);
    if (auto fn = this->secStore.find(path); fn != this->secStore.end())
    {
        return fn->second;
    }
    return NULL;
}

void Actor::insertAction(std::string path, Intelligence *intel, std::function<ActionOutput(StateHolder, ActionInput)> fn)
{
    std::lock_guard<std::mutex> lock(this->lock);
    this->store[path] = fn;
    this->secStore[path] = new SecAction(this->security, intel, fn);
}
