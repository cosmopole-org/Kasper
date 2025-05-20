#pragma once

#include "packet.h"
#include "../../core/trx/trx.h"
#include <string>
#include <vector>
#include "istorage.h"

#include <rocksdb/db.h>
#include <rocksdb/options.h>
#include <rocksdb/slice.h>
#include <rocksdb/utilities/transaction.h>
#include <rocksdb/utilities/transaction_db.h>
#include <mutex>

using ROCKSDB_NAMESPACE::Options;
using ROCKSDB_NAMESPACE::ReadOptions;
using ROCKSDB_NAMESPACE::Snapshot;
using ROCKSDB_NAMESPACE::Status;
using ROCKSDB_NAMESPACE::Transaction;
using ROCKSDB_NAMESPACE::TransactionDB;
using ROCKSDB_NAMESPACE::TransactionDBOptions;
using ROCKSDB_NAMESPACE::TransactionOptions;
using ROCKSDB_NAMESPACE::WriteOptions;

class Storage : public IStorage
{
public:
    TransactionDB *basedb;
    std::mutex lock;

    Storage(std::string basedbPath);
    void logPacket(std::string pointId, std::string userId, std::string data) override;
    std::vector<Packet> getPacketLogs(std::string pointId, std::string userId) override;
    std::string generateId(StateTrx *trx, std::string origin) override;
    TransactionDB *getBasedb() override;
};
