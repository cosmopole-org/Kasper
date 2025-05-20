#pragma once

#include <stdio.h>
#include <unordered_map>
#include <string>
#include <future>
#include <list>
#include <unistd.h>
#include <mutex>
#include <string.h>
#include <any>
#include <unordered_set>
#include <iostream>
#include <vector>
#include <queue>
#include <atomic>
#include <wasmedge/wasmedge.h>
#include <sstream>
#include <map>

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

#include "../../../utils/nlohmann/json.hpp"
using json = nlohmann::json;

using namespace std;

char *wasmSend(char *);

void log(std::string text);

TransactionDB *get_txn_db();
void set_txn_db(TransactionDB *);

struct wasmLock
{
public:
    std::mutex mut;
};

struct WasmTask
{
public:
    int id;
    std::string name;
    unordered_map<int, pair<bool, WasmTask *>> inputs;
    unordered_map<int, WasmTask *> outputs;
    int vmIndex;
    bool started = false;
};

class ChainTrx
{
public:
    std::string MachineId;
    std::string Key;
    std::string Input;
    std::string UserId;
    std::string CallbackId;
    ChainTrx(std::string machineId,
             std::string key,
             std::string input,
             std::string userId,
             std::string callbackId);
};

class WasmDbOp
{
public:
    std::string type;
    std::string key;
    std::string val;
};

class Trx
{
public:
    Transaction *trx;
    WriteOptions write_options;
    ReadOptions read_options;
    TransactionOptions txn_options;
    map<std::string, std::string> store{};
    map<std::string, bool> newlyCreated{};
    map<std::string, bool> newlyDeleted{};
    vector<WasmDbOp> ops{};

    Trx();
    vector<char> getBytesOfStr(std::string str);
    void put(std::string key, std::string val);
    vector<std::string> getByPrefix(std::string prefix);
    std::string get(std::string key);
    void del(std::string key);
    void commitAsOffchain();
    void dummyCommit();
};

class WasmMac
{
public:
    std::string executionResult;
    bool onchain;
    function<char *(char *)> callback;
    std::string id;
    std::string machineId;
    std::string pointId;
    int index;
    Trx *trx;
    uint64_t instCounter{1};
    std::priority_queue<uint64_t, std::vector<uint64_t>, std::greater<uint64_t>> triggerQueue{};
    unordered_map<uint64_t, unordered_map<char *, uint32_t>> triggerListeners{};
    bool newTiggersPendingToAdd{false};
    bool paused{false};
    mutex tirggerwasmLock;
    thread looper;
    queue<function<void()>> tasks{};
    mutex queue_mutex_;
    condition_variable cv_;
    WasmEdge_VMContext *vm;
    bool stop_ = false;
    atomic<uint64_t> triggerFront = 0;
    bool (*areAllPassed)(uint64_t instCounter, std::unordered_map<char *, uint32_t> waiters, char *myId);
    void (*tryToCheckTrigger)(char *vmId, uint32_t resNum, uint64_t instNum, char *myId);
    int (*lock)(char *vmId, uint32_t resNum, bool shouldwasmLock);
    void (*unlock)(char *vmId, uint32_t resNum);
    vector<tuple<vector<std::string>, std::string>> syncTasks{};
    atomic<int> step = 0;

    void prepareLooper();
    WasmMac(std::string machineId, std::string pointId, std::string modPath, function<char *(char *)> cb);
    WasmMac(std::string machineId, std::string vmId, int index, std::string modPath, function<char *(char *)> cb);
    void registerHost(std::string modPath);
    void registerFunction(WasmEdge_ModuleInstanceContext *HostMod, char *name, WasmEdge_HostFunc_t fn, WasmEdge_ValType *ParamList, int paramsLength, WasmEdge_ValType *ReturnList);
    vector<WasmDbOp> finalize();
    void enqueue(function<void()> task);
    void executeOnUpdate(std::string input);
    void runTask(std::string taskId);
    void executeOnChain(std::string input, void *cr);
    void stick();
};

class ConcurrentRunner
{
public:
    int WASM_COUNT = 0;
    atomic<int> wasmDoneTasks = 0;
    std::mutex wasmGlobalLock;
    vector<WasmMac *> wasmVms{};
    vector<std::mutex *> execwasmLocks{};
    std::mutex mainwasmLock;
    function<void(WasmTask *)> execWasmTask;
    vector<ChainTrx *> trxs{};
    std::string astStorePath;
    std::unordered_map<std::string, std::pair<int, WasmMac *>> wasmVmMap = {};

    ConcurrentRunner(std::string astStorePath, vector<ChainTrx *> trxs);
    void run();
    void prepareContext(int vmCount);
    void registerWasmMac(WasmMac *rt);
    void wasmRunTask(function<void(void *)> task, int index);
    void wasmDoCritical();
};

void wasmDoCritical();
void wasmRunTask(function<void(void *)>, int index);

WasmEdge_Result newSyncTask(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out);
WasmEdge_Result output(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out);
WasmEdge_Result consoleLog(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out);
WasmEdge_Result trx_put(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out);
WasmEdge_Result trx_del(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out);
WasmEdge_Result trx_get(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out);
WasmEdge_Result trx_get_by_prefix(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out);
WasmEdge_Result submitOnchainTrx(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out);
WasmEdge_Result runDocker(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out);
WasmEdge_Result execDocker(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out);
WasmEdge_Result plantTrigger(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out);
WasmEdge_Result signalPoint(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out);

// Utils: -------------------------------------------------

class WasmThreadPool
{
public:
    WasmThreadPool(size_t num_threads = thread::hardware_concurrency());
    void stick();
    void stopPool();
    void enqueue(function<void()> task);

private:
    vector<thread> threads_;
    queue<function<void()>> tasks_;
    mutex queue_mutex_;
    condition_variable cv_;
    bool stop_ = false;
};

class WasmUtils
{
public:
    static int parseDataAsInt(vector<char> buffer);
    static vector<char> pickSubarray(vector<char> A, int i, int j);
    static bool startswith(const std::string &str, const std::string &cmp);
    static std::string pickString(vector<char> A, int i, int j);
};

vector<char> wasmGetByteArrayOfChars(const char *c, int length);
vector<char> wasmGetBytesOfInt(int n);
vector<char> int64ToBytes(int64_t value);
