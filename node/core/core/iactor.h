#include "../../utils/ip.cpp"
#include "../trx/trx.h"
#include "../tools/itools.h"
#include "../tools/tools.cpp"
#include "icore.h"
#include "iaction.h"
#include <functional>
#include <string>
#include <map>

class IActor
{
public:
    virtual ~IActor() = default;
    virtual std::function<ActionOutput(State, ActionInput)> findAction(std::string path) = 0;
    virtual ISecAction* findActionAsSecure(std::string path) = 0;
    virtual void insertAction(std::string path, Intelligence* intel, std::function<ActionOutput(State, ActionInput)> fn) = 0;
};
