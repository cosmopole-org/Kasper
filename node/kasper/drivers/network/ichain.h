#pragma once

#include <cstdint>
#include <future>

class IChain
{
public:
    virtual ~IChain() = default;
	virtual void run(int port) = 0;
	virtual void submitTrx(std::string data) = 0;
};
