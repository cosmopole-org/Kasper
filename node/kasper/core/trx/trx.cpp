#pragma once

#include "trx.h"
#include <vector>

using json = nlohmann::json;

StateTrx::StateTrx(TransactionDB *db)
{
    WriteOptions write_options;
    this->trx = db->BeginTransaction(write_options);
}

void StateTrx::delKey(std::string key)
{
    this->trx->Delete(key);
}

void StateTrx::putString(std::string key, std::string val)
{
    this->trx->Put(key, val);
}

ReadOptions read_options;

std::string StateTrx::getString(std::string key)
{
    std::string value;
    Status s = this->trx->Get(read_options, key, &value);
    if (s.IsNotFound())
    {
        value = "";
    }
    return value;
}

void StateTrx::putBytes(std::string key, char *val, int len)
{
    std::string valStr = std::string(val, len);
    this->trx->Put(key, valStr);
}

ValPack StateTrx::getBytes(std::string key)
{
    auto val = this->getString(key);
    if (val.length() == 0)
    {
        char data[0];
        return ValPack{
            data,
            0};
    }
    return ValPack{
        (char*)val.c_str(),
        val.length()};
}

void StateTrx::delLink(std::string key)
{
    this->delKey("link::" + key);
}

void StateTrx::putLink(std::string key, std::string val)
{
    this->putString("link::" + key, val);
}

std::string StateTrx::getLink(std::string key)
{
    return this->getString("link::" + key);
}

void StateTrx::delIndex(std::string key)
{
    this->delKey("index::" + key);
}

void StateTrx::putIndex(std::string key, std::string val)
{
    this->putString("index::" + key, val);
}

std::string StateTrx::getIndex(std::string key)
{
    return this->getString("index::" + key);
}

void StateTrx::putJsonObj(std::string objType, std::string key, json obj)
{
    std::vector<std::pair<std::string, std::string>> data{};
    for (auto &[k, val] : obj.items())
    {
        std::string v = val.template get<std::string>();
        data.push_back({k, v});
    }

    for (auto p : data)
    {
        this->putString("obj::" + objType + "::" + key + "::" + p.first, p.second);
        std::cerr << std::endl << p.first << " " << p.second << std::endl;
    }
}

void StateTrx::putObj(std::string objType, std::string key, std::map<std::string, std::string> obj)
{
    for (auto item : obj)
    {
        this->putString("obj::" + objType + "::" + key + "::" + item.first, item.second);
    }
}

std::map<std::string, std::string> StateTrx::getObjAsMap(std::string objType, std::string key)
{
    std::map<std::string, std::string> obj;
    ReadOptions options = ReadOptions();
    auto itr = this->trx->GetIterator(options);
    std::string prefix = "obj::" + objType + "::" + key + "::";
    itr->Seek(prefix);
    while (itr->Valid())
    {
        std::string key = itr->key().ToString();
        std::string value = itr->value().ToString();
        if (!Utils::getInstance().stringStartsWith(key, prefix))
            break;
        std::string k = key.substr(prefix.length(), key.length() - prefix.length());
        if (k != "|")
        {
            obj[k] = value;
        }
        itr->Next();
    }
    return obj;
}

json StateTrx::getObjAsJson(std::string objType, std::string key)
{
    json obj;
    ReadOptions options = ReadOptions();
    auto itr = this->trx->GetIterator(options);
    std::string prefix = "obj::" + objType + "::" + key + "::";
    itr->Seek(prefix);
    while (itr->Valid())
    {
        std::string key = itr->key().ToString();
        std::string value = itr->value().ToString();
        if (!Utils::getInstance().stringStartsWith(key, prefix))
            break;
        std::string k = key.substr(prefix.length(), key.length() - prefix.length());
        if (k != "|")
        {
            obj[k] = value;
        }
        itr->Next();
    }
    return obj;
}

ValPack StateTrx::getColumn(std::string objType, std::string key, std::string columnKey)
{
    return this->getBytes("obj::" + objType + "::" + key + "::" + columnKey);
}

std::vector<std::string> StateTrx::getLinksList(std::string p)
{
    std::vector<std::string> links{};
    ReadOptions options = ReadOptions();
    auto itr = this->trx->GetIterator(options);
    std::string prefix = "index::" + p;
    itr->Seek(prefix);
    while (itr->Valid())
    {
        std::string key = itr->key().ToString();
        std::string value = itr->value().ToString();
        if (!Utils::getInstance().stringStartsWith(key, prefix))
            break;
        links.push_back(key.substr(prefix.length(), key.length() - prefix.length()));
        itr->Next();
    }
    return links;
}

EVP_PKEY *StateTrx::getPubKey(std::string userId)
{
    auto data = this->getString("obj::User::" + userId + "::publicKey");
    std::cerr << "public key data: [" << data << "]" << std::endl;
    return Utils::getInstance().load_public_key_from_string(data);
}

void StateTrx::commit()
{
    this->trx->Commit();
}

void StateTrx::discard()
{
    this->trx->Rollback();
}

void StateTrx::dispose()
{
    delete this->trx;
    delete this;
}
