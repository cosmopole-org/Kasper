#pragma once

#include "../../drivers/file/ifile.h"
#include "../../drivers/storage/istorage.h"
#include "../../drivers/signaler/isignaler.h"
#include "../../drivers/security/isecurity.h"
#include "../../drivers/network/itcp.h"
#include "../../drivers/wasm/iwasm.h"
#include "../../drivers/network/ifed.h"

#include "../../drivers/file/file.h"
#include "../../drivers/storage/storage.h"
#include "../../drivers/signaler/signaler.h"
#include "../../drivers/security/security.h"
#include "../../drivers/network/tcp.h"
#include "../../drivers/wasm/wasm.h"
#include "../../drivers/network/fed.h"

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
    IWasm *wasm;
    IFed *federation;

public:
    Tools(ICore *core, std::map<std::string, std::string> env);
    IStorage *getStorage() override;
    ISecurity *getSecurity() override;
    ISignaler *getSignaler() override;
    IFile *getFile() override;
    ITcp *getNetwork() override;
    IWasm *getWasm() override;
    IFed *getFederation() override;
};
