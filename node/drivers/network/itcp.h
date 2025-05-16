#include <cstdint>

class ITcp
{
public:
    virtual ~ITcp() = default;
    virtual void run(int port) = 0;
    virtual void handleConnection(uint64_t id, int conn) = 0;
};
