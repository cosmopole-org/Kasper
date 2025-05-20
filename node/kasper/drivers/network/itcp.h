#pragma once

#include <cstdint>
#include <future>

class ITcp
{
public:
    virtual ~ITcp() = default;
    virtual std::shared_future<void> run(int port) = 0;
};
