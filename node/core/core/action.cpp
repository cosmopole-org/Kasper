#include "../../drivers/security/isecurity.h"
#include <functional>
#include <string>
#include <map>
#include <mutex>
#include <unordered_map>
#include "iaction.h"
#include "icore.h"

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

class SecAction : public ISecAction
{
    Intelligence *intel;
    ISecurity *security;
    std::function<ActionOutput(State, ActionInput)> fn;

public:
    SecAction(ISecurity *security, Intelligence *intel, std::function<ActionOutput(State, ActionInput)> fn)
    {
        this->intel = intel;
        this->security = security;
        this->fn = fn;
    }

    ActionOutput run(ICore *core, std::string userId, char *payload, char *signature)
    {
        json input = json::parse(payload);
        auto meta = this->intel->extractMeta(input);

        if (meta.origin == "global") {

            return;

        } else if (meta.origin != core->getIp()) {

            return;
        }

        auto checkres = this->security->authWithSignature(userId, payload, signature);
        if (this->intel->mustBeUser())
        {
            if (!checkres.verified)
            {
                ActionOutput response;
                json data;
                response.data = data;
                response.resCode = 1;
                response.err = "authentication failed";
                return response;
            }
            if (this->intel->mustBeMember())
            {
                auto accessVerified = this->security->hasAccessToPoint(userId, meta.pointId);
                if (!accessVerified) {
                    ActionOutput response;
                    json data;
                    response.data = data;
                    response.resCode = 2;
                    response.err = "authorization failed";
                    return response;
                }
            }
        }
        ActionOutput output;
        core->modifyState([&output, this, &input, userId, meta](StateTrx* trx) {
            auto state = State{userId, meta.pointId, meta.origin, trx};
            output = this->fn(state, ActionInput{input});
        });
        return output;
    }
};
