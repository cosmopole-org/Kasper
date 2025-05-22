#pragma once

#include "../tools/tools.h"
#include "../trx/trx.h"
#include "actor.h"
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
    EVP_PKEY *pkey;
    Core();
    void modifyState(std::function<void(StateTrx *)> fn) override;
    ITools *getTools() override;
    IActor *getActor() override;
    std::string getIp() override;
    std::string signPacket(std::string data) override;
    void run() override;
};
