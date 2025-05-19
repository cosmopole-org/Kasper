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

    class JoinPointIntel : public Intelligence
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
    ActionOutput joinPoint(StateHolder state, ActionInput input)
    {
        ActionOutput output;
        std::string pointId = input.data["pointId"].template get<std::string>();
        auto v = state.trx->getColumn("Point", pointId, "isPublic");
        if (v.len == 0)
        {
            output.resCode = 1;
            output.err = "point not found";
            return output;
        }
        if (v.data[0] != 0x01)
        {
            output.resCode = 1;
            output.err = "point is not public";
            return output;
        }
        state.tools->getSignaler()->join(state.userId, pointId);
        return output;
    }

    class LeavePointIntel : public Intelligence
    {
    public:
        bool mustBeUser() override
        {
            return true;
        }

        bool mustBeMember() override
        {
            return true;
        }

        Meta extractMeta(json data) override
        {
            return Meta{};
        }
    };
    ActionOutput leavePoint(StateHolder state, ActionInput input)
    {
        ActionOutput output;
        std::string pointId = input.data["pointId"].template get<std::string>();
        state.tools->getSignaler()->leave(state.userId, pointId);
        return output;
    }

    class AddMemberIntel : public Intelligence
    {
    public:
        bool mustBeUser() override
        {
            return true;
        }

        bool mustBeMember() override
        {
            return true;
        }

        Meta extractMeta(json data) override
        {
            return Meta{};
        }
    };
    ActionOutput addMember(StateHolder state, ActionInput input)
    {
        ActionOutput output;
        std::string pointId = input.data["pointId"].template get<std::string>();
        std::string userId = input.data["userId"].template get<std::string>();
        if (state.trx->getLink("adminof::" + state.userId + "::" + pointId) != "true")
        {
            output.resCode = 1;
            output.err = "you are not admin";
            return output;
        }
        if (state.trx->getLink("memberof::" + userId + "::" + pointId) == "true")
        {
            output.resCode = 1;
            output.err = "user is already a member";
            return output;
        }
        state.tools->getSignaler()->join(userId, pointId);
        return output;
    }

    class RemoveMemberIntel : public Intelligence
    {
    public:
        bool mustBeUser() override
        {
            return true;
        }

        bool mustBeMember() override
        {
            return true;
        }

        Meta extractMeta(json data) override
        {
            return Meta{};
        }
    };
    ActionOutput removeMember(StateHolder state, ActionInput input)
    {
        ActionOutput output;
        std::string pointId = input.data["pointId"].template get<std::string>();
        std::string userId = input.data["userId"].template get<std::string>();
        if (state.trx->getLink("adminof::" + state.userId + "::" + pointId) != "true")
        {
            output.resCode = 1;
            output.err = "you are not admin";
            return output;
        }
        if (state.trx->getLink("memberof::" + userId + "::" + pointId) != "true")
        {
            output.resCode = 1;
            output.err = "user is not a member";
            return output;
        }
        state.tools->getSignaler()->leave(userId, pointId);
        return output;
    }

    class InviteUserIntel : public Intelligence
    {
    public:
        bool mustBeUser() override
        {
            return true;
        }

        bool mustBeMember() override
        {
            return true;
        }

        Meta extractMeta(json data) override
        {
            return Meta{};
        }
    };
    ActionOutput inviteUser(StateHolder state, ActionInput input)
    {
        ActionOutput output;
        std::string pointId = input.data["pointId"].template get<std::string>();
        std::string userId = input.data["userId"].template get<std::string>();
        if (state.trx->getLink("adminof::" + state.userId + "::" + pointId) != "true")
        {
            output.resCode = 1;
            output.err = "you are not admin";
            return output;
        }
        if (state.trx->getLink("invitedto::" + userId + "::" + pointId) == "true")
        {
            output.resCode = 1;
            output.err = "user is already invited";
            return output;
        }
        state.trx->putLink("invitedto::" + userId + "::" + pointId, "true");
        return output;
    }

    class DeInviteUserIntel : public Intelligence
    {
    public:
        bool mustBeUser() override
        {
            return true;
        }

        bool mustBeMember() override
        {
            return true;
        }

        Meta extractMeta(json data) override
        {
            return Meta{};
        }
    };
    ActionOutput deinviteUser(StateHolder state, ActionInput input)
    {
        ActionOutput output;
        std::string pointId = input.data["pointId"].template get<std::string>();
        std::string userId = input.data["userId"].template get<std::string>();
        if (state.trx->getLink("adminof::" + state.userId + "::" + pointId) != "true")
        {
            output.resCode = 1;
            output.err = "you are not admin";
            return output;
        }
        if (state.trx->getLink("invitedto::" + userId + "::" + pointId) != "true")
        {
            output.resCode = 1;
            output.err = "user is not invited";
            return output;
        }
        state.trx->delLink("invitedto::" + userId + "::" + pointId);
        return output;
    }

    class AcceptInviteIntel : public Intelligence
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
    ActionOutput acceptInvite(StateHolder state, ActionInput input)
    {
        ActionOutput output;
        std::string pointId = input.data["pointId"].template get<std::string>();
        if (state.trx->getLink("invitedto::" + state.userId + "::" + pointId) != "true")
        {
            output.resCode = 1;
            output.err = "you are not invited";
            return output;
        }
        state.trx->delLink("invitedto::" + state.userId + "::" + pointId);
        state.tools->getSignaler()->join(state.userId, pointId);
        return output;
    }

    class DeclineInviteIntel : public Intelligence
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
    ActionOutput declineInvite(StateHolder state, ActionInput input)
    {
        ActionOutput output;
        std::string pointId = input.data["pointId"].template get<std::string>();
        if (state.trx->getLink("invitedto::" + state.userId + "::" + pointId) != "true")
        {
            output.resCode = 1;
            output.err = "you are not invited";
            return output;
        }
        state.trx->delLink("invitedto::" + state.userId + "::" + pointId);
        return output;
    }

    class ConnectPointsIntel : public Intelligence
    {
    public:
        bool mustBeUser() override
        {
            return true;
        }

        bool mustBeMember() override
        {
            return true;
        }

        Meta extractMeta(json data) override
        {
            return Meta{};
        }
    };
    ActionOutput connectPoints(StateHolder state, ActionInput input)
    {
        ActionOutput output;
        std::string parentPointId = input.data["parentPointId"].template get<std::string>();
        std::string childPointId = input.data["childPointId"].template get<std::string>();
        if (state.trx->getLink("parentof::" + parentPointId + "::" + childPointId) == "true")
        {
            output.resCode = 1;
            output.err = "points are already connected";
            return output;
        }
        bool isAdminOfParent = state.trx->getLink("adminof::" + state.userId + "::" + parentPointId) == "true";
        bool isAdminOfChild = state.trx->getLink("adminof::" + state.userId + "::" + childPointId) == "true";
        if (!isAdminOfParent && !isAdminOfChild)
        {
            output.resCode = 1;
            output.err = "you are not admin of any of 2 points";
            return output;
        }
        if (isAdminOfParent)
        {
            if (state.trx->getLink("reqParentof::" + parentPointId + "::" + childPointId + "::" + parentPointId) == "true")
            {
                output.resCode = 1;
                output.err = "connection request is already submitted";
                return output;
            }
            state.trx->putLink("reqParentof::" + parentPointId + "::" + childPointId + "::" + parentPointId, "true");
        }
        else if (isAdminOfChild)
        {
            if (state.trx->getLink("reqParentof::" + parentPointId + "::" + childPointId + "::" + childPointId) == "true")
            {
                output.resCode = 1;
                output.err = "connection request is already submitted";
                return output;
            }
            state.trx->putLink("reqParentof::" + parentPointId + "::" + childPointId + "::" + childPointId, "true");
        }
        bool reqParent = state.trx->getLink("reqParentof::" + parentPointId + "::" + childPointId + "::" + childPointId) == "true";
        bool reqChild = state.trx->getLink("reqParentof::" + parentPointId + "::" + childPointId + "::" + childPointId) == "true";
        if (reqParent && reqChild) {
            state.trx->putLink("parentof::" + parentPointId + "::" + childPointId, "true");
            state.trx->delLink("reqParentof::" + parentPointId + "::" + childPointId + "::" + childPointId);
            state.trx->delLink("reqParentof::" + parentPointId + "::" + childPointId + "::" + childPointId);
        }
        return output;
    }

    void installService(ICore *core)
    {
        core->getActor()->insertAction("/points/create", new CreatePointIntel(), createPoint);
        core->getActor()->insertAction("/points/get", new GetPointIntel(), getPoint);
        core->getActor()->insertAction("/points/joinPoint", new JoinPointIntel(), joinPoint);
        core->getActor()->insertAction("/points/leavePoint", new LeavePointIntel(), leavePoint);
        core->getActor()->insertAction("/points/addMember", new AddMemberIntel(), addMember);
        core->getActor()->insertAction("/points/removeMember", new RemoveMemberIntel(), removeMember);
        core->getActor()->insertAction("/points/inviteUser", new InviteUserIntel(), inviteUser);
        core->getActor()->insertAction("/points/deinviteUser", new DeInviteUserIntel(), deinviteUser);
        core->getActor()->insertAction("/points/acceptInvite", new AcceptInviteIntel(), acceptInvite);
        core->getActor()->insertAction("/points/declineInvite", new DeclineInviteIntel(), declineInvite);
        core->getActor()->insertAction("/points/connectPoints", new ConnectPointsIntel(), connectPoints);
    }
}
