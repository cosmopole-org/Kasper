
#include <queue>
#include <mutex>
#include <condition_variable>
#include <string>
#include <future>
#include <unordered_map>
#include <cstring>
#include <iostream>
#include <netinet/in.h>
#include <sys/socket.h>
#include <unistd.h>
#include "../../utils/nlohmann/json.hpp"
#include <any>
#include <optional>
#include "itcp.h"
#include "../../utils/utils.h"
#include "../../core/core/icore.h"
#include "../file/datapack.h"
#include <thread>

template <typename T>
class SafeQueue
{
public:
	SafeQueue() = default;
	~SafeQueue() = default;
	SafeQueue(const SafeQueue &) = delete;
	SafeQueue &operator=(const SafeQueue &) = delete;
	void push(const T &item);
	void push(T &&item);
	T wait_and_pop();
	std::optional<T> try_pop();
	bool empty() const;
	size_t size() const;
	ValPack peek() const;

private:
	mutable std::mutex mutex_;
	std::condition_variable cond_var_;
	std::queue<T> queue_;
};
