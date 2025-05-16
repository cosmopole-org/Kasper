#include "../file/datapack.h"
#include "signverifyres.h"
#include <string>
#include <map>
#include <vector>

class ISecurity
{
public:
    virtual ~ISecurity() = default;
    virtual void loadKeys() = 0;
    virtual void generateSecureKeyPair(std::string tag) = 0;
    virtual std::vector<DataPack> fetchKeyPair(std::string tag) = 0;
    virtual SignVerifyRes authWithSignature(std::string userId, char *packet, char *signatureBase64) = 0;
    virtual bool hasAccessToPoint(std::string userId, std::string pointId) = 0;
};
