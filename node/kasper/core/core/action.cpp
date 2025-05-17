#pragma once

#include "../../drivers/security/isecurity.h"
#include <functional>
#include <string>
#include <map>
#include <mutex>
#include <unordered_map>
#include "../tools/itools.h"
#include "iaction.h"
#include "icore.h"

class SecAction : public ISecAction
{
    Intelligence *intel;
    ISecurity *security;
    std::function<ActionOutput(StateHolder, ActionInput)> fn;

public:
    SecAction(ISecurity *security, Intelligence *intel, std::function<ActionOutput(StateHolder, ActionInput)> fn)
    {
        this->intel = intel;
        this->security = security;
        this->fn = fn;
    }

    ActionOutput run(std::string myOrigin, std::function<void(std::function<void(StateTrx *)>)> stateModifier, ITools *tools, std::string userId, DataPack payload, std::string signature)
    {
        json input = json::parse(std::string(payload.data, payload.len));
        auto meta = this->intel->extractMeta(input);

        if (meta.origin == "global")
        {
            return {};
        }
        else if ((meta.origin != "") && (meta.origin != myOrigin))
        {
            return {};
        }

        if (this->intel->mustBeUser())
        {
            auto checkres = this->security->authWithSignature(userId, std::string(payload.data, payload.len), signature);
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
                if (!accessVerified)
                {
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
        stateModifier([&output, this, &input, userId, meta, &tools](StateTrx *trx)
                      {
            auto state = StateHolder{userId, meta.pointId, meta.origin, trx, tools};
            output = this->fn(state, ActionInput{input}); });
        return output;
    }
};
