#include <string>

bool startswith(const std::string &str, const std::string &cmp)
{
    return str.compare(0, cmp.length(), cmp) == 0;
}