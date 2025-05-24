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
	std::unordered_map<std::string, uint64_t> votes;
};

class Block
{
public:
	std::vector<std::pair<std::string, std::string>> trxs;
};
