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

#include "nlohmann/json.hpp"
using json = nlohmann::json;

using namespace std;

function<char *(char *)> wasmSend;

void log(std::string text)
{
    json j;
    j["key"] = "log";
    json j2;
    j2["text"] = text;
    j["input"] = j2;
    std::string packet = j.dump();
    wasmSend(&packet[0]);
}

Options options;
TransactionDBOptions txn_db_options;
TransactionDB *txn_db;

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
             std::string callbackId)
    {
        this->MachineId = machineId;
        this->UserId = userId;
        this->Key = key;
        this->CallbackId = callbackId;
        this->Input = input;
    }
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
    WasmMac(std::string machineId, std::string modPath, function<char *(char *)> cb);
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

// Utils: -------------------------------------------------

class WasmThreadPool
{
public:
    WasmThreadPool(size_t num_threads = thread::hardware_concurrency())
    {
        for (size_t i = 0; i < num_threads; ++i)
        {
            threads_.emplace_back([this]
                                  {
                  while (true) {
                      function<void()> task;
                      {
                          unique_lock<mutex> lock(
                              queue_mutex_);
                          cv_.wait(lock, [this] {
                              return !tasks_.empty() || stop_;
                          });
                          if (stop_ && tasks_.empty()) {
                              return;
                          }
                          task = move(tasks_.front());
                          tasks_.pop();
                      }
  
                      task();
                  } });
        }
    }
    void stick()
    {
        for (auto &thread : threads_)
        {
            thread.join();
        }
    }

    void stopPool()
    {
        {
            unique_lock<mutex> lock(queue_mutex_);
            stop_ = true;
        }

        cv_.notify_all();
    }

    void enqueue(function<void()> task)
    {
        {
            unique_lock<std::mutex> lock(queue_mutex_);
            tasks_.emplace(move(task));
        }
        cv_.notify_one();
    }

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
    static int parseDataAsInt(vector<char> buffer)
    {
        return uint32_t((unsigned char)(buffer[0]) << 24 |
                        (unsigned char)(buffer[1]) << 16 |
                        (unsigned char)(buffer[2]) << 8 |
                        (unsigned char)(buffer[3]));
    }
    static vector<char> pickSubarray(vector<char> A, int i, int j)
    {
        vector<char> sub = vector<char>(j, 0);
        for (int x = i; x < i + j; x++)
        {
            sub[x - i] = A[x];
        }
        return sub;
    }
    static bool startswith(const std::string &str, const std::string &cmp)
    {
        return str.compare(0, cmp.length(), cmp) == 0;
    }
    static std::string pickString(vector<char> A, int i, int j)
    {
        vector<char> da = WasmUtils::pickSubarray(A, i, j);
        std::string str(da.data(), da.size());
        return str;
    }
};

vector<char> wasmGetByteArrayOfChars(const char *c, int length)
{
    std::vector<char> bytes(c, c + length);
    return bytes;
}

vector<char> wasmGetBytesOfInt(int n)
{
    vector<char> bytes = vector<char>(4, 0);
    bytes[0] = (n >> 24) & 0xFF;
    bytes[1] = (n >> 16) & 0xFF;
    bytes[2] = (n >> 8) & 0xFF;
    bytes[3] = n & 0xFF;
    return bytes;
}

vector<char> int64ToBytes(int64_t value)
{
    std::vector<char> result;
    result.push_back(value >> 56);
    result.push_back(value >> 48);
    result.push_back(value >> 40);
    result.push_back(value >> 32);
    result.push_back(value >> 24);
    result.push_back(value >> 16);
    result.push_back(value >> 8);
    result.push_back(value);
    return result;
}

// Trx Implementation: -------------------------------------------------

Trx::Trx()
{
    this->trx = txn_db->BeginTransaction(write_options);
}

vector<char> Trx::getBytesOfStr(std::string str)
{
    std::vector<char> bytes(str.begin(), str.end());
    bytes.push_back('\0');
    return bytes;
}

void Trx::put(std::string key, std::string val)
{
    this->ops.push_back(WasmDbOp{type : "put", key : key, val : val});
    this->store[key] = val;
    this->newlyCreated[key] = true;
    this->newlyDeleted.erase(key);
}

vector<std::string> Trx::getByPrefix(std::string prefix)
{
    vector<std::string> result{};

    for (auto item : this->store)
    {
        if (WasmUtils::startswith(item.first, prefix))
        {
            result.push_back(std::string(item.second.data(), item.second.size()));
        }
    }

    ReadOptions options = ReadOptions();
    auto itr = this->trx->GetIterator(options);
    itr->Seek(prefix);
    while (itr->Valid())
    {
        std::string key = itr->key().ToString();
        std::string value = itr->value().ToString();
        if (!WasmUtils::startswith(key, prefix))
            break;
        if ((this->newlyCreated.find(key) == this->newlyCreated.end()) && (this->newlyDeleted.find(key) == this->newlyDeleted.end()))
        {
            result.push_back(value);
        }
        itr->Next();
    }

    return result;
}

std::string Trx::get(std::string key)
{
    if (this->store.find(key) != this->store.end())
    {
        return this->store[key];
    }
    else
    {
        std::string value;

        Status s = this->trx->Get(this->read_options, key, &value);
        if (s.IsNotFound())
        {
        }
        this->store[key] = value;
        return value;
    }
}

void Trx::del(std::string key)
{
    this->ops.push_back(WasmDbOp{type : "del", key : key});
    this->store.erase(key);
    this->newlyCreated.erase(key);
    this->newlyDeleted[key] = true;
}

void Trx::commitAsOffchain()
{
    for (auto op : this->ops)
    {
        if (op.type == "put")
        {
            this->trx->Put(op.key, op.val);
        }
        else if (op.type == "del")
        {
            this->trx->Delete(op.key);
        }
    }
    Status s = this->trx->Commit();
    if (s.ok())
    {
        log("committed transaction successfully.");
    }
    else
    {
        log("committing transaction failed.");
        log(s.ToString());
    }
}

void Trx::dummyCommit()
{
    this->trx->Commit();
}

// WasmMac Implementation: -------------------------------------------------

void WasmMac::prepareLooper()
{
    this->looper = thread([this]
                          {
      while (!stop_) {
        function<void()> task;
        {
          unique_lock<mutex> lock(
            this->queue_mutex_);
          cv_.wait(lock, [this] {
            return !this->tasks.empty() || this->stop_;
          });
          if (this->stop_ && this->tasks.empty()) {
            return;
          }
          task = this->tasks.front();
          this->tasks.pop();
        }
        printf("executing...\n");
        task();
        printf("done!\n");
      }
      printf("ended!\n"); });
}

WasmMac::WasmMac(std::string machineId, std::string modPath, function<char *(char *)> cb)
{
    this->onchain = false;
    this->callback = cb;
    this->machineId = machineId;
    this->id = "";
    this->index = 0;
    this->trx = new Trx();
    if (this->onchain)
    {
        this->prepareLooper();
    }
    this->registerHost(modPath);
}

WasmMac::WasmMac(std::string machineId, std::string vmId, int index, std::string modPath, function<char *(char *)> cb)
{
    this->onchain = true;
    this->callback = cb;
    this->machineId = machineId;
    this->id = vmId;
    this->index = index;
    this->trx = new Trx();
    if (this->onchain)
    {
        this->prepareLooper();
    }
    this->registerHost(modPath);
}

void WasmMac::registerHost(std::string modPath)
{
    WasmEdge_ConfigureContext *ConfCxt = WasmEdge_ConfigureCreate();
    WasmEdge_ConfigureAddHostRegistration(ConfCxt, WasmEdge_HostRegistration_Wasi);
    WasmEdge_ConfigureStatisticsSetInstructionCounting(ConfCxt, true);
    WasmEdge_VMContext *VMCxt = WasmEdge_VMCreate(ConfCxt, NULL);

    WasmEdge_String HostName = WasmEdge_StringCreateByCString("env");
    WasmEdge_ModuleInstanceContext *HostMod = WasmEdge_ModuleInstanceCreate(HostName);
    WasmEdge_ValType Params1[4] = {WasmEdge_ValTypeGenI32(), WasmEdge_ValTypeGenI32(), WasmEdge_ValTypeGenI32(), WasmEdge_ValTypeGenI32()};
    WasmEdge_ValType Returns1[1] = {WasmEdge_ValTypeGenI32()};
    this->registerFunction(HostMod, "put", trx_put, Params1, 4, Returns1);
    WasmEdge_ValType Params2[2] = {WasmEdge_ValTypeGenI32(), WasmEdge_ValTypeGenI32()};
    WasmEdge_ValType Returns2[1] = {WasmEdge_ValTypeGenI32()};
    this->registerFunction(HostMod, "del", trx_del, Params2, 2, Returns2);
    WasmEdge_ValType Params3[2] = {WasmEdge_ValTypeGenI32(), WasmEdge_ValTypeGenI32()};
    WasmEdge_ValType Returns3[1] = {WasmEdge_ValTypeGenI64()};
    this->registerFunction(HostMod, "get", trx_get, Params3, 2, Returns3);
    WasmEdge_ValType Params4[2] = {WasmEdge_ValTypeGenI32(), WasmEdge_ValTypeGenI32()};
    WasmEdge_ValType Returns4[1] = {WasmEdge_ValTypeGenI64()};
    this->registerFunction(HostMod, "getByPrefix", trx_get_by_prefix, Params4, 2, Returns4);
    WasmEdge_ValType Params5[2] = {WasmEdge_ValTypeGenI32(), WasmEdge_ValTypeGenI32()};
    WasmEdge_ValType Returns5[1] = {WasmEdge_ValTypeGenI32()};
    this->registerFunction(HostMod, "consoleLog", consoleLog, Params5, 2, Returns5);
    WasmEdge_ValType Params6[2] = {WasmEdge_ValTypeGenI32(), WasmEdge_ValTypeGenI32()};
    WasmEdge_ValType Returns6[1] = {WasmEdge_ValTypeGenI32()};
    this->registerFunction(HostMod, "output", output, Params6, 2, Returns6);
    WasmEdge_ValType Params7[6] = {WasmEdge_ValTypeGenI32(), WasmEdge_ValTypeGenI32(), WasmEdge_ValTypeGenI32(), WasmEdge_ValTypeGenI32(), WasmEdge_ValTypeGenI32(), WasmEdge_ValTypeGenI32()};
    WasmEdge_ValType Returns7[1] = {WasmEdge_ValTypeGenI64()};
    this->registerFunction(HostMod, "submitOnchainTrx", submitOnchainTrx, Params7, 6, Returns7);
    WasmEdge_ValType Params8[2] = {WasmEdge_ValTypeGenI32(), WasmEdge_ValTypeGenI32()};
    WasmEdge_ValType Returns8[1] = {WasmEdge_ValTypeGenI32()};
    this->registerFunction(HostMod, "newSyncTask", newSyncTask, Params8, 2, Returns8);
    auto Res = WasmEdge_VMRegisterModuleFromImport(VMCxt, HostMod);
    if (!WasmEdge_ResultOK(Res))
    {
        printf("Host module registration failed: %s\n", WasmEdge_ResultGetMessage(Res));
    }
    WasmEdge_StringDelete(HostName);

    Res = WasmEdge_VMLoadWasmFromFile(VMCxt, &modPath[0]);
    if (!WasmEdge_ResultOK(Res))
    {
        printf("Loading phase failed: %s\n", WasmEdge_ResultGetMessage(Res));
    }
    Res = WasmEdge_VMValidate(VMCxt);
    if (!WasmEdge_ResultOK(Res))
    {
        printf("Validation phase failed: %s\n", WasmEdge_ResultGetMessage(Res));
    }
    Res = WasmEdge_VMInstantiate(VMCxt);
    if (!WasmEdge_ResultOK(Res))
    {
        printf("Instantiation phase failed: %s\n", WasmEdge_ResultGetMessage(Res));
    }
    this->vm = VMCxt;
}

void WasmMac::registerFunction(WasmEdge_ModuleInstanceContext *HostMod, char *name, WasmEdge_HostFunc_t fn, WasmEdge_ValType *ParamList, int paramsLength, WasmEdge_ValType *ReturnList)
{
    WasmEdge_FunctionTypeContext *HostFType = WasmEdge_FunctionTypeCreate(ParamList, paramsLength, ReturnList, 1);
    WasmEdge_FunctionInstanceContext *HostFunc = WasmEdge_FunctionInstanceCreate(HostFType, fn, this, 0);
    WasmEdge_String HostName = WasmEdge_StringCreateByCString(name);
    WasmEdge_ModuleInstanceAddFunction(HostMod, HostName, HostFunc);
    // WasmEdge_StringDelete(HostName);
    // WasmEdge_FunctionTypeDelete(HostFType);
}

vector<WasmDbOp> WasmMac::finalize()
{
    if (this->onchain)
    {
        this->trx->dummyCommit();
    }
    else
    {
        this->trx->commitAsOffchain();
    }
    return this->trx->ops;
}

void WasmMac::enqueue(function<void()> task)
{
    {
        unique_lock<std::mutex> lock(queue_mutex_);
        if (this->tasks.empty())
        {
            // printf("enqueing...\n");
            this->tasks.emplace(task);
        }
    }
    cv_.notify_one();
}

void WasmMac::executeOnUpdate(std::string input)
{
    auto memName = WasmEdge_StringCreateByCString("memory");
    auto mallocName = WasmEdge_StringCreateByCString("malloc");

    auto mod = WasmEdge_VMGetActiveModule(this->vm);
    auto mem = WasmEdge_ModuleInstanceFindMemory(mod, memName);

    int valL = input.size();

    WasmEdge_Value Params[1] = {WasmEdge_ValueGenI32(valL)};
    WasmEdge_Value Returns[1] = {WasmEdge_ValueGenI32(0)};
    WasmEdge_VMExecute(this->vm, mallocName, Params, 1, Returns, 1);
    int valOffset = WasmEdge_ValueGetI32(Returns[0]);
    char *rawArr = &input[0];
    unsigned char *arr = new unsigned char[valL];
    for (int i = 0; i < valL; i++)
    {
        arr[i] = (unsigned char)rawArr[i];
    }
    WasmEdge_MemoryInstanceSetData(mem, arr, valOffset, valL);
    int64_t c = ((ino64_t)valOffset << 32) | valL;

    WasmEdge_String FuncName = WasmEdge_StringCreateByCString("run");
    WasmEdge_Value Params2[1] = {WasmEdge_ValueGenI64(c)};
    WasmEdge_Value Returns2[1] = {WasmEdge_ValueGenI64(0)};
    auto Res2 = WasmEdge_VMExecute(this->vm, FuncName, Params2, 1, Returns2, 0);
    if (!WasmEdge_ResultOK(Res2))
    {
        printf("Execution phase failed: %s\n", WasmEdge_ResultGetMessage(Res2));
    }
    // WasmEdge_StringDelete(FuncName);
    // WasmEdge_StringDelete(memName);
    // WasmEdge_StringDelete(mallocName);
}

void WasmMac::runTask(std::string taskId)
{
    auto memName = WasmEdge_StringCreateByCString("memory");
    auto mallocName = WasmEdge_StringCreateByCString("malloc");

    auto mod = WasmEdge_VMGetActiveModule(this->vm);
    auto mem = WasmEdge_ModuleInstanceFindMemory(mod, memName);

    int valL = taskId.size();

    WasmEdge_Value Params[1] = {WasmEdge_ValueGenI32(valL)};
    WasmEdge_Value Returns[1] = {WasmEdge_ValueGenI32(0)};
    WasmEdge_VMExecute(this->vm, mallocName, Params, 1, Returns, 1);
    int valOffset = WasmEdge_ValueGetI32(Returns[0]);
    char *rawArr = &taskId[0];
    unsigned char *arr = new unsigned char[valL];
    for (int i = 0; i < valL; i++)
    {
        arr[i] = (unsigned char)rawArr[i];
    }
    WasmEdge_MemoryInstanceSetData(mem, arr, valOffset, valL);
    int64_t c = ((ino64_t)valOffset << 32) | valL;

    WasmEdge_String FuncName = WasmEdge_StringCreateByCString("runTask");
    WasmEdge_Value Params2[1] = {WasmEdge_ValueGenI64(c)};
    WasmEdge_Value Returns2[1] = {WasmEdge_ValueGenI32(0)};
    auto Res = WasmEdge_VMExecute(this->vm, FuncName, Params2, 1, Returns2, 1);
    if (!WasmEdge_ResultOK(Res))
    {
        printf("Execution phase failed: %s\n", WasmEdge_ResultGetMessage(Res));
    }
    // WasmEdge_StringDelete(FuncName);
}

void WasmMac::executeOnChain(std::string input, void *crRaw)
{
    ConcurrentRunner *cr = (ConcurrentRunner *)crRaw;
    auto memName = WasmEdge_StringCreateByCString("memory");
    auto mallocName = WasmEdge_StringCreateByCString("malloc");

    auto mod = WasmEdge_VMGetActiveModule(this->vm);
    auto mem = WasmEdge_ModuleInstanceFindMemory(mod, memName);

    int valL = input.size();

    WasmEdge_Value Params[1] = {WasmEdge_ValueGenI32(valL)};
    WasmEdge_Value Returns[1] = {WasmEdge_ValueGenI32(0)};
    WasmEdge_VMExecute(this->vm, mallocName, Params, 1, Returns, 1);
    int valOffset = WasmEdge_ValueGetI32(Returns[0]);
    char *rawArr = &input[0];
    unsigned char *arr = new unsigned char[valL];
    for (int i = 0; i < valL; i++)
    {
        arr[i] = (unsigned char)rawArr[i];
    }
    WasmEdge_MemoryInstanceSetData(mem, arr, valOffset, valL);
    int64_t c = ((ino64_t)valOffset << 32) | valL;

    WasmEdge_String FuncName = WasmEdge_StringCreateByCString("run");
    WasmEdge_Value Params2[1] = {WasmEdge_ValueGenI64(c)};
    WasmEdge_Value Returns2[1] = {WasmEdge_ValueGenI64(0)};
    auto Res = WasmEdge_VMExecute(this->vm, FuncName, Params2, 1, Returns2, 0);
    if (!WasmEdge_ResultOK(Res))
    {
        printf("Execution phase failed: %s\n", WasmEdge_ResultGetMessage(Res));
    }
    WasmEdge_StringDelete(FuncName);

    if (this->onchain)
    {
        cr->wasmGlobalLock.lock();
        cr->wasmDoneTasks++;
        if (cr->wasmDoneTasks == cr->WASM_COUNT)
        {
            if (step == 0)
            {
                cr->wasmDoneTasks = 0;
                step++;
                cr->wasmGlobalLock.unlock();
                cr->wasmDoCritical();
            }
        }
    }
}

void WasmMac::stick()
{
    this->looper.join();
}

// Wasm Host Functions: -------------------------------------------------

WasmEdge_Result newSyncTask(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out)
{
    auto memName = WasmEdge_StringCreateByCString("memory");

    WasmMac *rt = (WasmMac *)data;
    auto mod = WasmEdge_VMGetActiveModule(rt->vm);
    uint32_t keyOffset = WasmEdge_ValueGetI32(In[0]);
    int keyL = WasmEdge_ValueGetI32(In[1]);

    auto mem = WasmEdge_ModuleInstanceFindMemory(mod, memName);

    unsigned char *rawKey = new unsigned char[keyL];
    vector<char> rawKeyC{};
    WasmEdge_MemoryInstanceGetData(mem, rawKey, keyOffset, keyL);
    for (int i = 0; i < keyL; i++)
    {
        if (rawKey[i] == '\00')
            break;
        rawKeyC.push_back((char)rawKey[i]);
    }
    auto text = std::string(rawKeyC.begin(), rawKeyC.end());

    log(text);

    json j = json::parse(text);

    std::string name = j["name"].template get<std::string>();

    vector<std::string> deps{};
    for (json::iterator item = j["deps"].begin(); item != j["deps"].end(); ++item)
    {
        deps.push_back(item.value().template get<std::string>());
    }

    rt->syncTasks.push_back({deps, name});

    // WasmEdge_StringDelete(memName);

    return WasmEdge_Result_Success;
}

WasmEdge_Result output(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out)
{
    auto memName = WasmEdge_StringCreateByCString("memory");

    WasmMac *rt = (WasmMac *)data;
    auto mod = WasmEdge_VMGetActiveModule(rt->vm);
    uint32_t keyOffset = WasmEdge_ValueGetI32(In[0]);
    int keyL = WasmEdge_ValueGetI32(In[1]);

    auto mem = WasmEdge_ModuleInstanceFindMemory(mod, memName);

    unsigned char *rawKey = new unsigned char[keyL];
    vector<char> rawKeyC{};
    WasmEdge_MemoryInstanceGetData(mem, rawKey, keyOffset, keyL);
    for (int i = 0; i < keyL; i++)
    {
        if (rawKey[i] == '\00')
            break;
        rawKeyC.push_back((char)rawKey[i]);
    }
    auto text = std::string(rawKeyC.begin(), rawKeyC.end());

    log(text);

    rt->executionResult = text;

    // WasmEdge_StringDelete(memName);

    return WasmEdge_Result_Success;
}

WasmEdge_Result consoleLog(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out)
{
    auto memName = WasmEdge_StringCreateByCString("memory");

    WasmMac *rt = (WasmMac *)data;
    auto mod = WasmEdge_VMGetActiveModule(rt->vm);
    uint32_t keyOffset = WasmEdge_ValueGetI32(In[0]);
    int keyL = WasmEdge_ValueGetI32(In[1]);

    auto mem = WasmEdge_ModuleInstanceFindMemory(mod, memName);

    unsigned char *rawKey = new unsigned char[keyL];
    vector<char> rawKeyC{};
    WasmEdge_MemoryInstanceGetData(mem, rawKey, keyOffset, keyL);
    for (int i = 0; i < keyL; i++)
    {
        if (rawKey[i] == '\00')
            break;
        rawKeyC.push_back((char)rawKey[i]);
    }
    auto text = std::string(rawKeyC.begin(), rawKeyC.end());

    log(text);

    // WasmEdge_StringDelete(memName);

    return WasmEdge_Result_Success;
}

WasmEdge_Result submitOnchainTrx(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out)
{
    auto memName = WasmEdge_StringCreateByCString("memory");
    auto mallocName = WasmEdge_StringCreateByCString("malloc");

    WasmMac *rt = (WasmMac *)data;
    auto mod = WasmEdge_VMGetActiveModule(rt->vm);
    uint32_t tmOffset = WasmEdge_ValueGetI32(In[0]);
    int tmL = WasmEdge_ValueGetI32(In[1]);
    uint32_t keyOffset = WasmEdge_ValueGetI32(In[2]);
    int keyL = WasmEdge_ValueGetI32(In[3]);
    uint32_t inputOffset = WasmEdge_ValueGetI32(In[4]);
    int inputL = WasmEdge_ValueGetI32(In[5]);

    auto mem = WasmEdge_ModuleInstanceFindMemory(mod, memName);

    unsigned char *rawTm = new unsigned char[tmL];
    vector<char> rawTmC{};
    WasmEdge_MemoryInstanceGetData(mem, rawTm, tmOffset, tmL);
    for (int i = 0; i < tmL; i++)
    {
        if (rawTm[i] == '\00')
            break;
        rawTmC.push_back((char)rawTm[i]);
    }
    auto targetMachineId = std::string(rawTmC.begin(), rawTmC.end());

    unsigned char *rawKey = new unsigned char[keyL];
    vector<char> rawKeyC{};
    WasmEdge_MemoryInstanceGetData(mem, rawKey, keyOffset, keyL);
    for (int i = 0; i < keyL; i++)
    {
        if (rawKey[i] == '\00')
            break;
        rawKeyC.push_back((char)rawKey[i]);
    }
    auto key = std::string(rawKeyC.begin(), rawKeyC.end());

    unsigned char *rawInput = new unsigned char[inputL];
    vector<char> rawInputC{};
    WasmEdge_MemoryInstanceGetData(mem, rawInput, inputOffset, inputL);
    for (int i = 0; i < inputL; i++)
    {
        if (rawInput[i] == '\00')
            break;
        rawInputC.push_back((char)rawInput[i]);
    }
    auto input = std::string(rawInputC.begin(), rawInputC.end());

    log(targetMachineId + " || " + key + " || " + input);

    json j;
    j["key"] = "submitOnchainTrx";
    json j2;
    j2["machineId"] = rt->machineId;
    j2["targetMachineId"] = targetMachineId;
    j2["key"] = key;
    j2["packet"] = input;
    j["input"] = j2;
    std::string packet = j.dump();

    std::string val = rt->callback(&packet[0]);
    auto valL = val.size();

    WasmEdge_Value Params[1] = {WasmEdge_ValueGenI32(valL)};
    WasmEdge_Value Returns[1] = {WasmEdge_ValueGenI32(0)};

    WasmEdge_VMExecute(rt->vm, mallocName, Params, 1, Returns, 1);
    int valOffset = WasmEdge_ValueGetI32(Returns[0]);

    char *rawArr = &val[0];
    unsigned char *arr = new unsigned char[valL];
    for (int i = 0; i < valL; i++)
    {
        arr[i] = (unsigned char)rawArr[i];
    }

    WasmEdge_MemoryInstanceSetData(mem, arr, valOffset, valL);
    int64_t c = ((ino64_t)valOffset << 32) | valL;

    Out[0] = WasmEdge_ValueGenI64(c);

    // WasmEdge_StringDelete(memName);

    return WasmEdge_Result_Success;
}

WasmEdge_Result trx_put(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out)
{
    auto memName = WasmEdge_StringCreateByCString("memory");

    WasmMac *rt = (WasmMac *)data;
    auto mod = WasmEdge_VMGetActiveModule(rt->vm);
    uint32_t keyOffset = WasmEdge_ValueGetI32(In[0]);
    int keyL = WasmEdge_ValueGetI32(In[1]);
    uint32_t valOffset = WasmEdge_ValueGetI32(In[2]);
    int valL = WasmEdge_ValueGetI32(In[3]);

    auto mem = WasmEdge_ModuleInstanceFindMemory(mod, memName);

    unsigned char *rawKey = new unsigned char[keyL];
    vector<char> rawKeyC{};
    WasmEdge_MemoryInstanceGetData(mem, rawKey, keyOffset, keyL);
    for (int i = 0; i < keyL; i++)
    {
        if (rawKey[i] == '\00')
            break;
        rawKeyC.push_back((char)rawKey[i]);
    }
    auto key = std::string(rawKeyC.begin(), rawKeyC.end());

    unsigned char *rawVal = new unsigned char[valL];
    vector<char> rawValC{};
    WasmEdge_MemoryInstanceGetData(mem, rawVal, valOffset, valL);
    for (int i = 0; i < valL; i++)
    {
        rawValC.push_back((char)rawVal[i]);
    }
    auto val = std::string(rawValC.begin(), rawValC.end());

    // WasmEdge_StringDelete(memName);

    rt->trx->put(key, val);

    return WasmEdge_Result_Success;
}

WasmEdge_Result trx_del(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out)
{
    auto memName = WasmEdge_StringCreateByCString("memory");

    WasmMac *rt = (WasmMac *)data;
    auto mod = WasmEdge_VMGetActiveModule(rt->vm);
    uint32_t keyOffset = WasmEdge_ValueGetI32(In[0]);
    int keyL = WasmEdge_ValueGetI32(In[1]);

    auto mem = WasmEdge_ModuleInstanceFindMemory(mod, memName);

    unsigned char *rawKey = new unsigned char[keyL];
    vector<char> rawKeyC{};
    WasmEdge_MemoryInstanceGetData(mem, rawKey, keyOffset, keyL);
    for (int i = 0; i < keyL; i++)
    {
        if (rawKey[i] == '\00')
            break;
        rawKeyC.push_back((char)rawKey[i]);
    }
    auto key = std::string(rawKeyC.begin(), rawKeyC.end());

    // WasmEdge_StringDelete(memName);

    rt->trx->del(key);

    return WasmEdge_Result_Success;
}

WasmEdge_Result trx_get(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out)
{
    auto memName = WasmEdge_StringCreateByCString("memory");
    auto mallocName = WasmEdge_StringCreateByCString("malloc");

    WasmMac *rt = (WasmMac *)data;
    auto mod = WasmEdge_VMGetActiveModule(rt->vm);
    uint32_t keyOffset = WasmEdge_ValueGetI32(In[0]);
    int keyL = WasmEdge_ValueGetI32(In[1]);

    auto mem = WasmEdge_ModuleInstanceFindMemory(mod, memName);

    unsigned char *rawKey = new unsigned char[keyL];
    vector<char> rawKeyC{};
    WasmEdge_MemoryInstanceGetData(mem, rawKey, keyOffset, keyL);
    for (int i = 0; i < keyL; i++)
    {
        if (rawKey[i] == '\00')
            break;
        rawKeyC.push_back((char)rawKey[i]);
    }
    auto key = std::string(rawKeyC.begin(), rawKeyC.end());

    std::string val = rt->trx->get(key);
    int valL = val.size();

    WasmEdge_Value Params[1] = {WasmEdge_ValueGenI32(valL)};
    WasmEdge_Value Returns[1] = {WasmEdge_ValueGenI32(0)};

    WasmEdge_VMExecute(rt->vm, mallocName, Params, 1, Returns, 1);
    int valOffset = WasmEdge_ValueGetI32(Returns[0]);

    char *rawArr = &val[0];
    unsigned char *arr = new unsigned char[valL];
    for (int i = 0; i < valL; i++)
    {
        arr[i] = (unsigned char)rawArr[i];
    }

    WasmEdge_MemoryInstanceSetData(mem, arr, valOffset, valL);
    int64_t c = ((ino64_t)valOffset << 32) | valL;

    Out[0] = WasmEdge_ValueGenI64(c);

    // WasmEdge_StringDelete(memName);
    // WasmEdge_StringDelete(mallocName);

    return WasmEdge_Result_Success;
}

WasmEdge_Result trx_get_by_prefix(void *data, const WasmEdge_CallingFrameContext *, const WasmEdge_Value *In, WasmEdge_Value *Out)
{
    auto memName = WasmEdge_StringCreateByCString("memory");
    auto mallocName = WasmEdge_StringCreateByCString("malloc");

    WasmMac *rt = (WasmMac *)data;
    auto mod = WasmEdge_VMGetActiveModule(rt->vm);
    uint32_t keyOffset = WasmEdge_ValueGetI32(In[0]);
    int keyL = WasmEdge_ValueGetI32(In[1]);

    auto mem = WasmEdge_ModuleInstanceFindMemory(mod, memName);

    unsigned char *rawKey = new unsigned char[keyL];
    vector<char> rawKeyC{};
    WasmEdge_MemoryInstanceGetData(mem, rawKey, keyOffset, keyL);
    for (int i = 0; i < keyL; i++)
    {
        if (rawKey[i] == '\00')
            break;
        rawKeyC.push_back((char)rawKey[i]);
    }
    auto prefix = std::string(rawKeyC.begin(), rawKeyC.end());

    vector<std::string> vals = rt->trx->getByPrefix(prefix);

    json arrOfS;
    for (int i = 0; i < vals.size(); i++)
    {
        arrOfS.push_back(vals[i]);
    }
    json j;
    j["data"] = arrOfS;

    std::string val = j.dump();
    int valL = val.size();

    WasmEdge_Value Params[1] = {WasmEdge_ValueGenI32(valL)};
    WasmEdge_Value Returns[1] = {WasmEdge_ValueGenI32(0)};

    WasmEdge_VMExecute(rt->vm, mallocName, Params, 1, Returns, 1);
    int valOffset = WasmEdge_ValueGetI32(Returns[0]);

    char *rawArr = &val[0];
    unsigned char *arr = new unsigned char[valL];
    for (int i = 0; i < valL; i++)
    {
        arr[i] = (unsigned char)rawArr[i];
    }

    WasmEdge_MemoryInstanceSetData(mem, arr, valOffset, valL);
    int64_t c = ((ino64_t)valOffset << 32) | valL;

    Out[0] = WasmEdge_ValueGenI64(c);

    // WasmEdge_StringDelete(memName);
    // WasmEdge_StringDelete(mallocName);

    return WasmEdge_Result_Success;
}

// ConcurrentRunner Implementation: -------------------------------------------------

ConcurrentRunner::ConcurrentRunner(std::string astStorePath, vector<ChainTrx *> trxs)
{
    this->trxs = trxs;
    this->astStorePath = astStorePath;
}

void ConcurrentRunner::run()
{
    auto startTime = std::chrono::high_resolution_clock::now();
    WasmThreadPool tp(8);
    this->prepareContext(trxs.size());
    for (int i = 0; i < trxs.size(); i++)
    {
        tp.enqueue([this, i]
                   {
        auto rt = new WasmMac(this->trxs[i]->MachineId, to_string(i), i, this->astStorePath + "/" + this->trxs[i]->MachineId + "/module", wasmSend);
        this->registerWasmMac(rt);
        rt->executeOnChain(this->trxs[i]->Input, this); });
    }
    tp.stopPool();
    tp.stick();
    for (auto rt : this->wasmVms)
    {
        auto changes = rt->finalize();
        auto output = rt->executionResult;
        json arr;
        for (auto op : changes)
        {
            json item;
            item["opType"] = op.type;
            item["key"] = op.key;
            item["val"] = op.val;
            arr.push_back(item);
        }
        std::string opsStr = arr.dump();

        json j;
        j["key"] = "submitOnchainResponse";
        json j2;
        j2["callbackId"] = trxs[rt->index]->CallbackId;
        j2["packet"] = output;
        j2["resCode"] = 1;
        j2["error"] = "";
        j2["changes"] = opsStr;
        j["input"] = j2;
        std::string packet = j.dump();

        wasmSend(&packet[0]);
    }
    auto stopTime = std::chrono::high_resolution_clock::now();
    auto passedTime = std::chrono::duration_cast<std::chrono::microseconds>(stopTime - startTime).count();
    log("executed chain applet transactions in " + to_string(passedTime) + " microseconds.");
}

void ConcurrentRunner::prepareContext(int vmCount)
{
    WASM_COUNT = vmCount;
    wasmVms = vector<WasmMac *>(vmCount, NULL);
    execwasmLocks = vector<std::mutex *>(vmCount, new std::mutex());
}

void ConcurrentRunner::registerWasmMac(WasmMac *rt)
{
    wasmVmMap.insert({rt->id, {rt->index, rt}});
    wasmVms[rt->index] = rt;
}

void ConcurrentRunner::wasmRunTask(function<void(void *)> task, int index)
{
    task(wasmVms[index]);
}

void ConcurrentRunner::wasmDoCritical()
{
    int keyCounter = 1;
    map<int, std::tuple<vector<std::string>, int, std::string>> allWasmTasks{};
    map<std::string, WasmTask *> taskRefs{};
    unordered_map<std::string, pair<WasmTask *, wasmLock *>> reswasmLocks{};
    unordered_map<std::string, WasmTask *> startPoints{};
    auto resCounter = 0;
    auto c = 0;

    for (int index = 0; index < WASM_COUNT; index++)
    {
        WasmMac *vm = wasmVms[index];
        for (auto t : vm->syncTasks)
        {
            vector<std::string> resNums = std::get<0>(t);
            std::string name = std::get<1>(t);
            std::stringstream mySS;
            int resCount = 0;
            for (auto r : resNums)
            {
                if (reswasmLocks.find(r) == reswasmLocks.end())
                {
                    reswasmLocks[r] = {NULL, new wasmLock()};
                    resCounter++;
                }
                resCount++;
            }
            allWasmTasks[keyCounter] = {resNums, index, name};
            keyCounter++;
            c++;
        }
    }

    for (auto i = 1; i <= keyCounter; i++)
    {
        vector<std::string> resNums = std::get<0>(allWasmTasks[i]);
        unordered_map<int, pair<bool, WasmTask *>> inputs = {};
        unordered_map<int, WasmTask *> outputs = {};
        int vmIndex = std::get<1>(allWasmTasks[i]);
        std::string name = std::get<2>(allWasmTasks[i]);
        WasmTask *task = new WasmTask{i, name, inputs, outputs, vmIndex};
        std::string vmIndexStr = std::to_string(vmIndex);
        taskRefs[vmIndexStr + ":" + name] = task;
        for (auto r : resNums)
        {
            if (WasmUtils::startswith(r, "lock_"))
            {
                WasmTask *t = taskRefs[vmIndexStr + ":" + r];
                task->inputs[t->id] = {false, t};
                t->outputs[task->id] = task;
            }
            else
            {
                if (reswasmLocks[r].first == NULL)
                {
                    reswasmLocks[r].first = task;
                    startPoints[r] = task;
                }
                else
                {
                    task->inputs[reswasmLocks[r].first->id] = {false, reswasmLocks[r].first};
                    reswasmLocks[r].first->outputs[task->id] = task;
                    reswasmLocks[r].first = task;
                }
            }
        }
    }

    std::mutex threadwasmLocks[c];
    vector<thread> ts{};

    WasmThreadPool pool(resCounter);
    atomic<int> wasmDoneWasmTasksCount = 1;

    execWasmTask = [&pool, this, &wasmDoneWasmTasksCount, keyCounter](WasmTask *task)
    {
        bool readyToExec = false;
        this->mainwasmLock.lock();
        if (task->started == false)
        {
            bool passed = true;
            for (auto t : task->inputs)
            {
                if (!t.first)
                {
                    passed = false;
                    break;
                }
            }
            if (passed)
            {
                task->started = true;
                readyToExec = true;
            }
        }
        this->mainwasmLock.unlock();
        if (readyToExec)
        {
            pool.enqueue([task, this, &wasmDoneWasmTasksCount, keyCounter, &pool]
                         {
                  log("task "+ to_string(task->id));
                  this->execwasmLocks[task->vmIndex]->lock();
                  wasmRunTask([task](void *vmRaw)
                  {
                    void *res;
                    WasmMac* vm = (WasmMac*) vmRaw;
                    vm->runTask(task->name);
                  },
                  task->vmIndex);
                  this->execwasmLocks[task->vmIndex]->unlock();
                  this->mainwasmLock.lock();
                  wasmDoneWasmTasksCount++;
                  if (wasmDoneWasmTasksCount == keyCounter) {
                    pool.stopPool();
                  }
                  vector<WasmTask*> nextWasmTasks{};
                  for (auto t : task->outputs) {
                    if (!t.second->started) {
                      t.second->inputs[task->id].first = true;
                      nextWasmTasks.push_back(t.second);
                    }
                  }
                  this->mainwasmLock.unlock();
                  for (auto t : nextWasmTasks) {
                    this->execWasmTask(t);
                  } });
        }
    };
    for (auto q : startPoints)
    {
        auto task = q.second;
        this->execWasmTask(task);
    }
    pool.stick();
}
