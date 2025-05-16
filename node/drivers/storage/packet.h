#include <string>

struct Packet
{
public:
    std::string pointId;
    std::string author;
    uint64_t time;
    std::string data;
};
