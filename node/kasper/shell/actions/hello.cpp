#include "hello.h"

namespace service_hello
{
    class HelloIntel : public Intelligence
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
        core->getActor()->insertAction("/api/hello", new HelloIntel(), [](StateHolder state, ActionInput input)
                                       {
        ActionOutput output;
        std::string username = input.data["username"].template get<std::string>();
        output.data["message"] = "hello " + username + " !";
        state.trx->putString("admin", "keyhan mohammadi");
        return output; });

        core->getActor()->insertAction("/api/adminName", new HelloIntel(), [](StateHolder state, ActionInput input)
                                       {
        ActionOutput output;
        output.data["message"] = "admin is " + state.trx->getString("admin");
        return output; });
    }
}
