#pragma once

#include <functional>
#include "../trx/trx.h"
#include "../tools/itools.h"
#include "iactor.h"

class ICore
{
public:
    virtual ~ICore() = default;
    virtual void modifyState(std::function<void(StateTrx *)>) = 0;
    virtual ITools *getTools() = 0;
    virtual IActor *getActor() = 0;
    virtual std::string getIp() = 0;
    virtual void run() = 0;
};
