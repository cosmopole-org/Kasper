#include "../../drivers/security/isecurity.h"
#include <functional>
#include <string>
#include <map>
#include <mutex>
#include <unordered_map>
#include "../../utils/nlohmann/json.hpp"

using json = nlohmann::json;

struct ActionOutput
{
    int resCode;
    json data;
    std::string err;
};

struct ActionInput
{
    json data;
};

class Meta
{
public:
    std::string pointId;
    std::string origin;
};

class Intelligence
{
public:
    virtual bool mustBeUser() = 0;
    virtual bool mustBeMember() = 0;
    virtual Meta extractMeta(json data) = 0;
};

struct State
{
public:
    std::string userId;
    std::string pointId;
    std::string origin;
    StateTrx *trx;
};

class ISecAction
{
public:
    virtual ActionOutput run(ICore *core, std::string userId, char *payload, char *signature) = 0;
};
