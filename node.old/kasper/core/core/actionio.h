#pragma once

#include "../../utils/nlohmann/json.hpp"
#include <string>

using json = nlohmann::json;

struct ActionOutput
{
    int resCode;
    json data;
    std::string err;
};

struct ActionInput
{
    json data;
};
