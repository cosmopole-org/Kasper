#pragma once

#include "../../core/core/actionio.h"

struct Request
{
public:
	std::string userId;
	std::string requestId;
	std::string key;
	ActionInput input;
	std::function<void(int resCode, std::string response)> callback;
};
