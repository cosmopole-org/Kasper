#pragma once

#include "../tools/tools.cpp"
#include "../trx/trx.h"
#include "actor.cpp"
#include "icore.h"
#include "../../utils/utils.h"
#include <functional>
#include <string>
#include <map>

class ITools; // Forward declaration
class IActor; // Forward declaration

class Core : public ICore
{
public:
    std::string ip;
    ITools *tools;
    IActor *actor;
    Core();
    void modifyState(std::function<void(StateTrx *)> fn) override;
    ITools *getTools() override;
    IActor *getActor() override;
    std::string getIp() override;
    void run() override;
};
