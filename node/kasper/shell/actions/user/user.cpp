#include "user.h"
#include <vector>

using json = nlohmann::json;

namespace service_user
{
    class CreateUserIntel : public Intelligence
    {
    public:
        bool mustBeUser() override
        {
            return false;
        }

        bool mustBeMember() override
        {
            return false;
        }

        Meta extractMeta(json data) override
        {
            return Meta{};
        }
    };
    ActionOutput createUser(StateHolder state, ActionInput input)
    {
        ActionOutput output;
        std::string username = input.data["username"].template get<std::string>();
        if (state.trx->getIndex("User::username::id::" + username) != "") {
            output.resCode = 1;
            output.err = "username already exist";
            return output;
        }
        std::string userId = state.tools->getStorage()->generateId(state.trx, "global");
        std::vector<DataPack> keys = state.tools->getSecurity()->generateSecureKeyPair(userId);
        json user = {
            {"id", userId},
            {"type", "human"},
            {"username", username},
            {"publicKey", std::string(keys[1].data, keys[1].len)},
            {"privateKey", std::string(keys[0].data, keys[0].len)}};
        state.trx->putJsonObj("User", userId, user);
        state.trx->putIndex("User::username::id::" + username, userId);
        output.data["user"] = user;
        return output;
    }

    class GetUserIntel : public Intelligence
    {
    public:
        bool mustBeUser() override
        {
            return false;
        }

        bool mustBeMember() override
        {
            return false;
        }

        Meta extractMeta(json data) override
        {
            return Meta{};
        }
    };
    ActionOutput getUser(StateHolder state, ActionInput input)
    {
        ActionOutput output;
        std::string userId = input.data["userId"].template get<std::string>();
        if (state.trx->getColumn("User", userId, "id").len == 0) {
            output.resCode = 1;
            output.err = "user not found";
            return output;
        }
        json user = state.trx->getObjAsJson("User", userId);
        output.data["user"] = user;
        return output;
    }

    void installService(ICore *core)
    {
        core->getActor()->insertAction("/users/create", new CreateUserIntel(), createUser);
        core->getActor()->insertAction("/users/get", new GetUserIntel(), getUser);
    }
}
