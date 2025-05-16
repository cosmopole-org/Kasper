
#include "packet.h"
#include <string>
#include <vector>
#include "istorage.h"

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

void createDb(TransactionDB *txn_db, std::string path)
{
    Options options;
    TransactionDBOptions txn_db_options;
    options.create_if_missing = true;
    Status s = TransactionDB::Open(options, txn_db_options, path, &txn_db);
    assert(s.ok());
}

class Storage : public IStorage
{
public:
    TransactionDB *basedb;
    
    Storage(std::string basedbPath)
    {
        createDb(this->basedb, basedbPath);
    }

    void logPacket(std::string pointId, std::string userId, std::string data) override
    {
    
    }

    std::vector<Packet> getPacketLogs(std::string pointId, std::string userId) override
    {
        return {};
    }

    TransactionDB *getBasedb() override {
        return this->basedb;
    }
};
