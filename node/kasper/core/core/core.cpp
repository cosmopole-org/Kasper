#include "core.h"

Core::Core()
{
    this->ip = Utils::getInstance().getKasperNodeIPAddress();
    this->tools = new Tools(this,
                            {
                                {"BASE_DB_PATH", "/app/basedb"},
                                {"STORAGE_ROOT", "/app/storage"},
                            });
    this->actor = new Actor(this->tools->getSecurity(), this->getTools()->getFederation());
    auto priKeyData = this->tools->getSecurity()->fetchKeyPair("server_key")[0];
    this->pkey = Utils::getInstance().load_private_key_from_string(std::string(priKeyData.data, priKeyData.len));
}

void Core::modifyState(std::function<void(StateTrx *)> fn)
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

ITools *Core::getTools()
{
    return this->tools;
}

IActor *Core::getActor()
{
    return this->actor;
}

std::string Core::getIp()
{
    return this->ip;
}

std::string Core::signPacket(std::string data) {
    return Utils::getInstance().sign_payload_with_rsa(this->pkey, data);
}

void Core::run()
{
    this->tools->getNetwork()->run(8080);
    this->tools->getFederation()->run(8081);
}
