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

    void installService(ICore *core)
    {
        core->getActor()->insertAction("/users/create", new CreateUserIntel(), [](StateHolder state, ActionInput input)
                                       {
        ActionOutput output;
        std::string username = input.data["username"].template get<std::string>();
        std::string userId = state.tools->getStorage()->generateId(state.trx, "global");
        state.tools->getSecurity()->generateSecureKeyPair(userId);
        std::vector<DataPack> keys = state.tools->getSecurity()->fetchKeyPair(userId);
        json user = {
            {"type", "human"},
            {"username", username},
            {"publicKey", std::string(keys[1].data, keys[1].len)},
            {"privateKey", std::string(keys[0].data, keys[0].len)}
        };
        std::cerr << std::endl << "step 1" << std::endl << std::endl;
        state.trx->putJsonObj("User", userId, user);
        std::cerr << std::endl << "step 2" << std::endl << std::endl;
        output.data["user"] = user;
        std::cerr << std::endl << "step 3" << std::endl << std::endl;
        return output; });
    }
}
