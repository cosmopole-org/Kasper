#pragma once

#include "../../drivers/file/ifile.h"
#include "../../drivers/storage/istorage.h"
#include "../../drivers/signaler/isignaler.h"
#include "../../drivers/security/isecurity.h"
#include "../../drivers/network/itcp.h"
#include "../../drivers/file/file.h"
#include "../../drivers/storage/storage.cpp"
#include "../../drivers/signaler/signaler.cpp"
#include "../../drivers/security/security.cpp"
#include "../../drivers/network/tcp.cpp"
#include "../core/icore.h"
#include <map>
#include <string>

class Tools : public ITools
{
    IStorage *storage;
    ISecurity *security;
    ISignaler *signaler;
    IFile *file;
    ITcp *network;

public:
    Tools(ICore *core, std::map<std::string, std::string> env)
    {
        this->file = new File(env["STORAGE_ROOT"]);
        this->storage = new Storage(env["BASE_DB_PATH"]);
        this->signaler = new Signaler(core);
        this->security = new Security(core, env["STORAGE_ROOT"], this->file, this->storage, this->signaler);
        this->network = new Tcp(core);
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

    IFile *getFile() override
    {
        return this->file;
    }

    ITcp *getNetwork() override
    {
        return this->network;
    }
};
