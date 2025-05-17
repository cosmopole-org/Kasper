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

    Storage(std::string basedbPath)
    {
        this->basedb = nullptr;
        Options options;
        TransactionDBOptions txn_db_options;
        options.create_if_missing = true;
        Status s = TransactionDB::Open(options, txn_db_options, basedbPath, &this->basedb);
        assert(s.ok());
        if (!this->basedb)
        {
            throw std::runtime_error("Failed to create database");
        }
    }

    void logPacket(std::string pointId, std::string userId, std::string data) override
    {
    }

    std::vector<Packet> getPacketLogs(std::string pointId, std::string userId) override
    {
        return {};
    }

    std::string generateId(StateTrx *trx, std::string origin)
    {
        std::lock_guard<std::mutex> lock(this->lock);
        if (origin == "global")
        {
            std::cerr << "hellooo" << std::endl;
            auto old = trx->getBytes("maxGlobalId");
            std::cerr << "hellooo 2" << std::endl;
            int counter = 0;
            if (old.len > 0)
            {
                std::cerr << "hellooo 3" << std::endl;
                counter = Utils::getInstance().parseDataAsInt(old.data);
            }
            std::cerr << "hellooo 4" << std::endl;
            counter++;
            trx->putBytes("maxGlobalId", Utils::getInstance().convertIntToData(counter), 4);
            std::cerr << "hellooo 5" << std::endl;
            return std::to_string(counter) + "::glboal";
        }
        else
        {
            auto old = trx->getBytes("maxLocalId");
            int counter = 0;
            if (old.len > 0)
            {
                counter = Utils::getInstance().parseDataAsInt(old.data);
            }
            counter++;
            trx->putBytes("maxLocalId", Utils::getInstance().convertIntToData(counter), 4);
            return std::to_string(counter) + "::local";
        }
    }

    TransactionDB *getBasedb() override
    {
        return this->basedb;
    }
};
