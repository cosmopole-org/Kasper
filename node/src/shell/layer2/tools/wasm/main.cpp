#include <stdio.h>
#include <unordered_map>
#include <string>
#include <list>
#include <unistd.h>
#include <mutex>
#include <string.h>
#include <any>
#include <unordered_set>
#include <condition_variable>
#include <queue>
#include <thread>
#include <functional>
#include <iostream>
#include <fstream>

#include "main.h"

#include "lib/runtime.h"

using namespace std;

void init(char* kvDbPath)
{
  wasmSend = wasmCallback;
  options.create_if_missing = true;
  Status s = TransactionDB::Open(options, txn_db_options, kvDbPath, &txn_db);
  assert(s.ok());
}

void wasmRunVm(
    char *astPath,
    char *input,
    char *machineId)
{
  auto rt = new WasmMac(machineId, astPath, wasmCallback);
  rt->executeOnUpdate(input);
  rt->finalize();
}

void wasmRunEffects(char *effectsStr)
{
  json j = json::parse(effectsStr);
  WriteOptions write_options;
  Transaction *trx = txn_db->BeginTransaction(write_options);
  for (json::iterator item = j.begin(); item != j.end(); ++item)
  {
    if (item.value()["opType"].template get<std::string>() == "put")
    {
      trx->Put(item.value()["key"].template get<std::string>(), item.value()["val"].template get<std::string>());
    }
    else if (item.value()["opType"].template get<std::string>() == "del")
    {
      trx->Delete(item.value()["key"].template get<std::string>());
    }
  }
  Status s = trx->Commit();
  if (s.ok())
  {
    log("committed transaction group effects successfully.");
  }
  else
  {
    log("committing transaction group effects failed.");
  }
}

void wasmRunTrxs(
    char *astStorePath,
    char *input)
{
  json j = json::parse(input);
  vector<ChainTrx *> trxs{};
  for (json::iterator item = j.begin(); item != j.end(); ++item)
  {
    trxs.push_back(new ChainTrx(
        item.value()["machineId"].template get<std::string>(),
        item.value()["key"].template get<std::string>(),
        item.value()["payload"].template get<std::string>(),
        item.value()["userId"].template get<std::string>(),
        item.value()["callbackId"].template get<std::string>()));
  }
  ConcurrentRunner* cr = new ConcurrentRunner(astStorePath, trxs);
  cr->run();
}