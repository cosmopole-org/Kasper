#pragma once

#include <string>

class IChain
{
public:
	virtual ~IChain() = default;
	virtual void submitTrx(std::string typ, std::string payload) = 0;
};
