#pragma once

#include <string>
#include <map>
#include <vector>
#include "../../utils/nlohmann/json.hpp"
#include "../../utils/utils.h"

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

using json = nlohmann::json;

struct ValPack
{
    char *data;
    uint64_t len;
};

class StateTrx
{
    Transaction *trx;

public:
    StateTrx(TransactionDB *db);
    void delKey(std::string key);
    void putString(std::string key, std::string val);
    std::string getString(std::string key);
    void putBytes(std::string key, char *val, int len);
    ValPack getBytes(std::string key);
    void delLink(std::string key);
    void putLink(std::string key, std::string val);
    std::string getLink(std::string key);
    void delIndex(std::string key);
    void putIndex(std::string key, std::string val);
    std::string getIndex(std::string key);
    void putJsonObj(std::string objType, std::string key, json obj);
    void putObj(std::string objType, std::string key, std::map<std::string, std::string> obj);
    json getObjAsJson(std::string objType, std::string key);
    std::map<std::string, std::string> getObjAsMap(std::string objType, std::string key);
    ValPack getColumn(std::string objType, std::string key, std::string columnKey);
    std::vector<std::string> getLinksList(std::string prefix);
    EVP_PKEY* getPubKey(std::string userId);
    void commit();
    void discard();
    void dispose();
};