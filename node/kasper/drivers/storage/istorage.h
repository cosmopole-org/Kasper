#pragma once

#include "packet.h"
#include <string>
#include <vector>

#include <rocksdb/db.h>
#include <rocksdb/options.h>
#include <rocksdb/slice.h>
#include <rocksdb/utilities/transaction.h>
#include <rocksdb/utilities/transaction_db.h>

using ROCKSDB_NAMESPACE::Options;
using ROCKSDB_NAMESPACE::ReadOptions;
using ROCKSDB_NAMESPACE::Snapshot;
using ROCKSDB_NAMESPACE::Status;
using ROCKSDB_NAMESPACE::Transaction;
using ROCKSDB_NAMESPACE::TransactionDB;
using ROCKSDB_NAMESPACE::TransactionDBOptions;
using ROCKSDB_NAMESPACE::TransactionOptions;
using ROCKSDB_NAMESPACE::WriteOptions;

class IStorage
{
public:
    virtual ~IStorage() = default;
    virtual void logPacket(std::string pointId, std::string userId, std::string data) = 0;
    virtual std::vector<Packet> getPacketLogs(std::string pointId, std::string userId) = 0;
    virtual TransactionDB *getBasedb() = 0;
};
