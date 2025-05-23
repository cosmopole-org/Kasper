#pragma once

#include <string>
#include <vector>
#include <cstdint>
#include <unordered_map>
#include <unordered_set>

class Event
{
public:
    std::string origin;
	std::vector<std::pair<std::string, std::string>> trxs;
	std::string proof;
	std::string myUpdate;
	std::unordered_map<std::string, std::string> backedProofs;
	std::unordered_set<std::string> backedResponses;
};

class Block
{
public:
	uint64_t index;
	std::vector<std::pair<std::string, std::string>> trxs;
};
