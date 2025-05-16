#include "../../utils/ip.cpp"
#include "../trx/trx.h"
#include "../tools/itools.h"
#include "../tools/tools.cpp"
#include "iactor.h"
#include "actor.cpp"
#include "icore.h"
#include <functional>
#include <string>
#include <map>

class Core : public ICore
{
public:
    std::string ip;
    ITools *tools;
    IActor *actor;
    Core()
    {
        this->ip = getMachineIPAddress();
        this->tools = new Tools(this,
                                {
                                    {"BASE_DB_PATH", "/app/basedb"},
                                    {"STORAGE_ROOT", "/app/storage"},
                                });
        this->actor = new Actor(this->tools->getSecurity());
    }

    void modifyState(std::function<void(StateTrx *)> fn) override
    {
        auto trx = new StateTrx(this->tools->getStorage()->getBasedb());
        try
        {
            fn(trx);
            trx->commit();
        }
        catch (const std::exception &e)
        {
            std::cerr << "Standard exception caught: " << e.what() << std::endl;
            trx->discard();
        }
        catch (...)
        {
            std::cerr << "Unknown exception caught" << std::endl;
            trx->discard();
        }
        trx->dispose();
    }

    ITools *getTools() override
    {
        return this->tools;
    };

    IActor *getActor() override
    {
        return this->actor;
    };

    std::string getIp() override
    {
        return this->ip;
    };
};
