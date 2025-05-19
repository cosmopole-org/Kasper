#include "point.h"
#include <vector>
#include <map>
#include <any>

using json = nlohmann::json;

namespace service_point
{
    class CreatePointIntel : public Intelligence
    {
    public:
        bool mustBeUser() override
        {
            return true;
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
    ActionOutput createPoint(StateHolder state, ActionInput input)
    {
        ActionOutput output;
        auto pointId = state.tools->getStorage()->generateId(state.trx, "global");
        std::map<std::string, std::any> pointMap = {
            {"id", pointId},
            {"owner", state.userId},
            {"persHist", input.data["persHist"].template get<bool>()},
            {"isPublic", input.data["isPublic"].template get<bool>()}};
        json point = state.tools->getSignaler()->createPoint(pointMap);
        state.trx->putLink("adminof::" + state.userId + "::" + pointId, "true");
        state.tools->getSignaler()->join(state.userId, pointId);
        output.data["point"] = point;
        return output;
    }

    class GetPointIntel : public Intelligence
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
    ActionOutput getPoint(StateHolder state, ActionInput input)
    {
        ActionOutput output;
        std::string pointId = input.data["pointId"].template get<std::string>();
        if (state.trx->getColumn("Point", pointId, "id").len == 0)
        {
            output.resCode = 1;
            output.err = "point not found";
            return output;
        }
        json point = state.tools->getSignaler()->getPoint(pointId);
        output.data["point"] = point;
        return output;
    }

    void installService(ICore *core)
    {
        core->getActor()->insertAction("/points/create", new CreatePointIntel(), createPoint);
        core->getActor()->insertAction("/points/get", new GetPointIntel(), getPoint);
    }
}
