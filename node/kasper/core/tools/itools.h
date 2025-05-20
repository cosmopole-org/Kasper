#pragma once

#include "../../drivers/storage/istorage.h"
#include "../../drivers/signaler/isignaler.h"
#include "../../drivers/security/isecurity.h"
#include "../../drivers/network/itcp.h"
#include "../../drivers/file/ifile.h"
#include "../../drivers/wasm/iwasm.h"
#include "../../drivers/network/ifed.h"
#include <map>
#include <string>

class ITools
{
public:
    virtual ~ITools() = default;
    virtual IStorage *getStorage() = 0;
    virtual ISecurity *getSecurity() = 0;
    virtual ISignaler *getSignaler() = 0;
    virtual IFile *getFile() = 0;
    virtual ITcp *getNetwork() = 0;
    virtual IWasm *getWasm() = 0;
    virtual IFed *getFederation() = 0;
};
