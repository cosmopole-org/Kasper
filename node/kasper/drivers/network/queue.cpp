#include "queue.h"

template <typename T>
void SafeQueue<T>::push(const T &item)
{
	{
		std::lock_guard<std::mutex> lock(mutex_);
		queue_.push(item);
	}
	cond_var_.notify_one();
}

template <typename T>
void SafeQueue<T>::push(T &&item)
{
	{
		std::lock_guard<std::mutex> lock(mutex_);
		queue_.push(std::move(item));
	}
	cond_var_.notify_one();
}

template <typename T>
T SafeQueue<T>::wait_and_pop()
{
	std::unique_lock<std::mutex> lock(mutex_);
	cond_var_.wait(lock, [this]()
				   { return !queue_.empty(); });

	T item = std::move(queue_.front());
	queue_.pop();
	return item;
}

template <typename T>
std::optional<T> SafeQueue<T>::try_pop()
{
	std::lock_guard<std::mutex> lock(mutex_);
	if (queue_.empty())
	{
		return std::nullopt;
	}

	T item = std::move(queue_.front());
	queue_.pop();
	return item;
}

template <typename T>
bool SafeQueue<T>::empty() const
{
	std::lock_guard<std::mutex> lock(mutex_);
	return queue_.empty();
}

template <typename T>
size_t SafeQueue<T>::size() const
{
	std::lock_guard<std::mutex> lock(mutex_);
	return queue_.size();
}

template <typename T>
ValPack SafeQueue<T>::peek() const
{
	std::lock_guard<std::mutex> lock(mutex_);
	return queue_.front();
}
