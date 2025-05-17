#pragma once

#include <string>
#include <map>
#include <vector>
#include "../../utils/utils.h"
#include "../../drivers/storage/storage.cpp"

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
    std::string putObj(std::string objType, std::string key, std::map<std::string, std::string> obj);
    std::map<std::string, std::string> getObj(std::string objType, std::string key);
    ValPack getColumn(std::string objType, std::string key, std::string columnKey);
    std::vector<std::string> getLinksList(std::string prefix);
    RSA* getPubKey(std::string userId);
    void commit();
    void discard();
    void dispose();
};