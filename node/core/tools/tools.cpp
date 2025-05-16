#include "../drivers/storage/istorage.h"
#include "../drivers/signaler/isignaler.h"
#include "../drivers/security/isecurity.h"
#include "../drivers/storage/storage.cpp"
#include "../drivers/signaler/signaler.cpp"
#include "../drivers/security/security.cpp"
#include "../core/icore.h"
#include <map>
#include <string>

class Tools : public ITools
{
    IStorage *storage;
    ISecurity *security;
    ISignaler *signaler;

public:
    Tools(ICore *core, std::map<std::string, std::string> env)
    {
        this->storage = new Storage(env["BASE_DB_PATH"]);
        this->signaler = new Signaler(core);
        this->security = new Security(core, env["STORAGE_ROOT"], this->storage, this->signaler);
    }
    IStorage *getStorage() override
    {
        return this->storage;
    }

    ISecurity *getSecurity() override
    {
        return this->security;
    }

    ISignaler *getSignaler() override
    {
        return this->signaler;
    }
};
