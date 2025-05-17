#pragma once

#include <string>

struct SignVerifyRes
{
public:
    bool verified;
    std::string typ;
    bool isGod;
};