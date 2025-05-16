#include "../drivers/storage/istorage.h"
#include "../drivers/signaler/isignaler.h"
#include "../drivers/security/isecurity.h"
#include <map>
#include <string>

class ITools
{
public:
    virtual ~ITools() = default;
    virtual IStorage *getStorage() = 0;
    virtual ISecurity *getSecurity() = 0;
    virtual ISignaler *getSignaler() = 0;
};
