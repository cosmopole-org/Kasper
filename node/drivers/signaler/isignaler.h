#include <any>
#include <functional>
#include <string>
#include <vector>

class ISignaler
{
public:
    virtual ~ISignaler() = default;
    virtual std::function<void(std::string, std::any, size_t)> findListener(std::string userId) = 0;
    virtual void listenToSingle(std::string userId, std::function<void(std::string, std::any, size_t)> listener) = 0;
    virtual void join(std::string userId, std::string pointId) = 0;
    virtual void leave(std::string userId, std::string pointId) = 0;
    virtual void signalUserAsObj(std::string userId, std::string key, std::any data) = 0;
    virtual void signalUserAsBytes(std::string userId, std::string key, char *data, size_t len) = 0;
    virtual void signalPointAsObj(std::string pointId, std::string key, std::any data, std::vector<std::string> exceptions) = 0;
    virtual void signalPointAsBytes(std::string pointId, std::string key, char *data, size_t len, std::vector<std::string> exceptions) = 0;
};
