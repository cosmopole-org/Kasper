#pragma once

#include <cstddef>

struct DataPack
{
    char *data;
    size_t len;
};

struct KeyPack
{
    DataPack priKey;
    DataPack pubKey;
};
