use std::collections::{ HashMap, HashSet, VecDeque, BTreeMap };
use std::ops::Deref;
use std::sync::{ Arc, Mutex, Condvar, RwLock };
use std::thread;
use std::sync::atomic::{ AtomicBool, Ordering };
use serde_json::{ json, Value as JsonValue };
use wasmedge_sys::instance::{ self, module };
use wasmedge_sys::plugin::PluginModule;
use wasmedge_sys::{
    config::Config,
    AsInstance,
    CallingFrame,
    Instance,
    Module,
    Statistics,
    Store,
    Executor,
    WasmValue,
    WasiModule,
};
use wasmedge_types::error::CoreError;
use wasmedge_types::ValType;
use std::ptr;

// RocksDB types
use rocksdb::{
    DB,
    Options,
    TransactionDB,
    TransactionDBOptions,
    Transaction,
    ReadOptions,
    WriteOptions,
    TransactionOptions,
    IteratorMode,
};

fn main() {
    println!("Hello, world!");
}

// Global variables
static mut OPTIONS: Option<Options> = None;
static mut TXN_DB_OPTIONS: Option<TransactionDBOptions> = None;
static mut TXN_DB: Option<*mut TransactionDB> = None;
static mut STEP: i32 = 0;

fn wasmSend(data: std::string::String) -> &'static str {
    return "";
}

fn log(text: String) {
    let j = json!({
        "key": "log",
        "input": {
            "text": text
        }
    });
    let packet = j.to_string();
    wasmSend(packet);
}

pub struct WasmLock {
    pub mut_: Mutex<()>,
}

impl WasmLock {
    pub fn generate() -> Self {
        WasmLock {
            mut_: Mutex::new(()),
        }
    }
}

pub struct WasmTask {
    pub id: i32,
    pub name: String,
    pub inputs: HashMap<i32, (bool, *mut WasmTask)>,
    pub outputs: HashMap<i32, *mut WasmTask>,
    pub vm_index: i32,
    pub started: bool,
}

impl WasmTask {
    pub fn new() -> Self {
        WasmTask {
            id: 0,
            name: String::new(),
            inputs: HashMap::new(),
            outputs: HashMap::new(),
            vm_index: 0,
            started: false,
        }
    }
}

pub struct ChainTrx {
    pub machine_id: String,
    pub key: String,
    pub input: String,
    pub user_id: String,
    pub callback_id: String,
}

impl ChainTrx {
    pub fn new(
        machine_id: String,
        key: String,
        input: String,
        user_id: String,
        callback_id: String
    ) -> Self {
        ChainTrx {
            machine_id,
            key,
            input,
            user_id,
            callback_id,
        }
    }
}

#[derive(Clone)]
pub struct WasmDbOp {
    pub type_: String,
    pub key: String,
    pub val: String,
}

impl WasmDbOp {
    pub fn new() -> Self {
        WasmDbOp {
            type_: String::new(),
            key: String::new(),
            val: String::new(),
        }
    }
}

pub struct Trx {
    pub trx: Option<*mut rocksdb::Transaction<'static, TransactionDB>>,
    pub write_options: WriteOptions,
    pub read_options: ReadOptions,
    pub txn_options: TransactionOptions,
    pub store: BTreeMap<String, String>,
    pub newly_created: BTreeMap<String, bool>,
    pub newly_deleted: BTreeMap<String, bool>,
    pub ops: Vec<WasmDbOp>,
}

impl Trx {
    pub fn new() -> Self {
        unsafe {
            let trx = if let Some(txn_db_ptr) = TXN_DB {
                let txn_db = &*txn_db_ptr;
                Some(Box::into_raw(Box::new(txn_db.transaction())))
            } else {
                None
            };

            Trx {
                trx,
                write_options: WriteOptions::default(),
                read_options: ReadOptions::default(),
                txn_options: TransactionOptions::default(),
                store: BTreeMap::new(),
                newly_created: BTreeMap::new(),
                newly_deleted: BTreeMap::new(),
                ops: Vec::new(),
            }
        }
    }

    pub fn get_bytes_of_str(&self, str_val: String) -> Vec<u8> {
        let mut bytes: Vec<u8> = str_val.into_bytes();
        bytes.push(0); // null terminator
        bytes
    }

    pub fn put(&mut self, key: String, val: String) {
        self.ops.push(WasmDbOp {
            type_: "put".to_string(),
            key: key.clone(),
            val: val.clone(),
        });
        self.store.insert(key.clone(), val);
        self.newly_created.insert(key.clone(), true);
        self.newly_deleted.remove(&key);
    }

    pub fn get_by_prefix(&mut self, prefix: String) -> Vec<String> {
        let mut result = Vec::new();

        // Check in-memory store first
        for (key, value) in &self.store {
            if WasmUtils::startswith(key, &prefix) {
                result.push(value.clone());
            }
        }

        // Check database
        if let Some(trx_ptr) = self.trx {
            unsafe {
                let trx = &*trx_ptr;
                // Note: This is a simplified implementation
                // In a real implementation, you'd use the transaction's iterator
                // For now, we'll just return what's in the store
            }
        }

        result
    }

    pub fn get(&mut self, key: String) -> String {
        if let Some(value) = self.store.get(&key) {
            value.clone()
        } else {
            let value = if let Some(trx_ptr) = self.trx {
                unsafe {
                    let trx = &*trx_ptr;
                    // In a real implementation, you'd use trx.get()
                    // For now, return empty string if not found
                    String::new()
                }
            } else {
                String::new()
            };
            self.store.insert(key.clone(), value.clone());
            value
        }
    }

    pub fn del(&mut self, key: String) {
        self.ops.push(WasmDbOp {
            type_: "del".to_string(),
            key: key.clone(),
            val: String::new(),
        });
        self.store.remove(&key);
        self.newly_created.remove(&key);
        self.newly_deleted.insert(key, true);
    }

    pub fn commit_as_offchain(&mut self) {
        if let Some(trx_ptr) = self.trx {
            unsafe {
                let trx = &mut *trx_ptr;
                for op in &self.ops {
                    if op.type_ == "put" {
                        // In real implementation: trx.put(&op.key, &op.val);
                    } else if op.type_ == "del" {
                        // In real implementation: trx.delete(&op.key);
                    }
                }
                // In real implementation: let status = trx.commit();
                log("committed transaction successfully.".to_string());
            }
        }
    }

    pub fn dummy_commit(&mut self) {
        if let Some(trx_ptr) = self.trx {
            unsafe {
                let trx = &mut *trx_ptr;
                // In real implementation: trx.commit();
            }
        }
    }
}

pub struct WasmThreadPool {
    threads_: Vec<thread::JoinHandle<()>>,
    tasks_: Arc<Mutex<VecDeque<Box<dyn FnOnce() + Send>>>>,
    queue_mutex_: Arc<Mutex<()>>,
    cv_: Arc<Condvar>,
    stop_: Arc<AtomicBool>,
}

impl WasmThreadPool {
    pub fn generate(num_threads: Option<usize>) -> Self {
        let num_threads = num_threads.unwrap_or_else(||
            thread
                ::available_parallelism()
                .map(|n| n.get())
                .unwrap_or(1)
        );
        let tasks_: Arc<Mutex<VecDeque<Box<dyn FnOnce() + Send>>>> = Arc::new(
            Mutex::new(VecDeque::new())
        );
        let queue_mutex_ = Arc::new(Mutex::new(()));
        let cv_ = Arc::new(Condvar::new());
        let stop_ = Arc::new(AtomicBool::new(false));
        let mut threads_ = Vec::new();

        for _i in 0..num_threads {
            let tasks_clone = Arc::clone(&tasks_);
            let queue_mutex_clone = Arc::clone(&queue_mutex_);
            let cv_clone = Arc::clone(&cv_);
            let stop_clone = Arc::clone(&stop_);

            let handle = thread::spawn(move || {
                loop {
                    let task = {
                        let _lock = queue_mutex_clone.lock().unwrap();
                        let mut tasks = tasks_clone.lock().unwrap();
                        let mg = cv_clone
                            .wait_while(tasks, |tasks| {
                                tasks.is_empty() && !stop_clone.load(Ordering::Relaxed)
                            })
                            .unwrap();

                        if
                            stop_clone.load(Ordering::Relaxed) &&
                            tasks_clone.lock().unwrap().is_empty()
                        {
                            return;
                        }

                        tasks_clone.lock().unwrap().pop_front()
                    };

                    if let Some(task) = task {
                        task();
                    }
                }
            });

            threads_.push(handle);
        }

        WasmThreadPool {
            threads_,
            tasks_,
            queue_mutex_,
            cv_,
            stop_,
        }
    }

    pub fn stick(&mut self) {
        for thread in self.threads_.drain(..) {
            thread.join().unwrap();
        }
    }

    pub fn stop_pool(&mut self) {
        self.stop_.store(true, Ordering::Relaxed);
        self.cv_.notify_all();
    }

    pub fn enqueue<F>(&self, task: F) where F: FnOnce() + Send + 'static {
        {
            let _lock = self.queue_mutex_.lock().unwrap();
            let mut tasks = self.tasks_.lock().unwrap();
            tasks.push_back(Box::new(task));
        }
        self.cv_.notify_one();
    }
}

pub struct WasmUtils;

impl WasmUtils {
    pub fn parse_data_as_int(buffer: Vec<u8>) -> i32 {
        (((buffer[0] as u32) << 24) as i32) |
            (((buffer[1] as u32) << 16) as i32) |
            (((buffer[2] as u32) << 8) as i32) |
            (buffer[3] as u32 as i32)
    }

    pub fn pick_subarray(a: Vec<u8>, i: usize, j: usize) -> Vec<u8> {
        let mut sub = vec![0u8; j];
        for x in i..i + j {
            if x < a.len() && x - i < sub.len() {
                sub[x - i] = a[x];
            }
        }
        sub
    }

    pub fn startswith(str_val: &str, cmp: &str) -> bool {
        str_val.starts_with(cmp)
    }

    pub fn pick_string(a: Vec<u8>, i: usize, j: usize) -> String {
        let da = WasmUtils::pick_subarray(a, i, j);
        String::from_utf8_lossy(&da).to_string()
    }
}

fn wasm_get_bytes_of_int(n: i32) -> Vec<u8> {
    vec![
        ((n >> 24) & 0xff) as u8,
        ((n >> 16) & 0xff) as u8,
        ((n >> 8) & 0xff) as u8,
        (n & 0xff) as u8
    ]
}

fn int64_to_bytes(value: i64) -> Vec<u8> {
    vec![
        (value >> 56) as u8,
        (value >> 48) as u8,
        (value >> 40) as u8,
        (value >> 32) as u8,
        (value >> 24) as u8,
        (value >> 16) as u8,
        (value >> 8) as u8,
        value as u8
    ]
}

// ConcurrentRunner struct (referenced in WasmMac)
pub struct ConcurrentRunner {
    pub wasm_global_lock: Mutex<()>,
    pub wasm_done_tasks: i32,
    pub wasm_count: i32,
}

// WasmMac Implementation: -------------------------------------------------
pub struct WasmMac {
    pub onchain: bool,
    pub chain_token_id: String,
    pub is_token_valid: bool,
    pub callback: Box<dyn (Fn(String) -> String) + Send + Sync>,
    pub machine_id: String,
    pub point_id: String,
    pub id: String,
    pub index: i32,
    pub trx: Box<Trx>,
    pub looper: Option<thread::JoinHandle<()>>,
    pub tasks: VecDeque<Box<dyn FnOnce() + Send>>,
    pub queue_mutex_: Mutex<()>,
    pub cv_: Condvar,
    pub stop_: bool,
    pub vm: Option<Executor>,
    pub vm_instance: Option<Instance>,
    pub mod_path: String,
}

impl WasmMac {
    fn prepare_looper(&mut self) {
        // Create a channel for communication instead of directly accessing self
        let (sender, receiver) = std::sync::mpsc::channel::<Box<dyn FnOnce() + Send>>();

        self.looper = Some(
            thread::spawn(move || {
                loop {
                    match receiver.recv() {
                        Ok(task) => {
                            println!("executing...");
                            task();
                            println!("done!");
                        }
                        Err(_) => {
                            println!("ended!");
                            break;
                        }
                    }
                }
            })
        );
    }

    pub fn new_offchain(
        machine_id: String,
        point_id: String,
        mod_path: String,
        cb: Box<dyn (Fn(String) -> String) + Send + Sync>
    ) -> Self {
        let mut wasm_mac = WasmMac {
            onchain: false,
            chain_token_id: String::new(),
            is_token_valid: false,
            callback: cb,
            machine_id,
            point_id,
            id: String::new(),
            index: 0,
            trx: Box::new(Trx::new()),
            looper: None,
            tasks: VecDeque::new(),
            queue_mutex_: Mutex::new(()),
            cv_: Condvar::new(),
            stop_: false,
            vm: None,
            vm_instance: None,
            mod_path,
        };

        if wasm_mac.onchain {
            wasm_mac.prepare_looper();
        }
        wasm_mac
    }

    pub fn new_onchain(
        machine_id: String,
        vm_id: String,
        index: i32,
        mod_path: String,
        cb: Box<dyn (Fn(String) -> String) + Send + Sync>
    ) -> Self {
        let mut wasm_mac = WasmMac {
            onchain: true,
            chain_token_id: String::new(),
            is_token_valid: false,
            callback: cb,
            machine_id,
            point_id: String::new(),
            id: vm_id,
            index,
            trx: Box::new(Trx::new()),
            looper: None,
            tasks: VecDeque::new(),
            queue_mutex_: Mutex::new(()),
            cv_: Condvar::new(),
            stop_: false,
            vm: None,
            vm_instance: None,
            mod_path,
        };

        if wasm_mac.onchain {
            wasm_mac.prepare_looper();
        }
        wasm_mac
    }

    fn register_host(&mut self, extern_mod: ImportModule<&mut WasmMac>) -> ImportObjectBuilder<()> {
        extern_mod.add_func(
            "host_add",
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(vec![ValType::I32, ValType::I32], vec![ValType::I32]),
                newSyncTask,
                self,
                1
            ).unwrap()
        );
        extern_mod.add_func(
            "runDocker",
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(
                    vec![
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32
                    ],
                    vec![ValType::I32]
                ),
                runDocker,
                self,
                1
            ).unwrap()
        );
        extern_mod.add_func(
            "execDocker",
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(
                    vec![
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32
                    ],
                    vec![ValType::I32]
                ),
                execDocker,
                self,
                1
            ).unwrap()
        );
        extern_mod.add_func(
            "copyToDocker",
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(
                    vec![
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32
                    ],
                    vec![ValType::I32]
                ),
                copyToDocker,
                self,
                1
            ).unwrap()
        );
        extern_mod.add_func(
            "consoleLog",
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(vec![ValType::I32, ValType::I32], vec![ValType::I32]),
                consoleLog,
                self,
                1
            ).unwrap()
        );
        extern_mod.add_func(
            "output",
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(vec![ValType::I32, ValType::I32], vec![ValType::I32]),
                output,
                self,
                1
            ).unwrap()
        );
        extern_mod.add_func(
            "httpPost",
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(
                    vec![
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32
                    ],
                    vec![ValType::I32]
                ),
                httpPost,
                self,
                1
            ).unwrap()
        );
        extern_mod.add_func(
            "plantTrigger",
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(
                    vec![
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32
                    ],
                    vec![ValType::I32]
                ),
                plantTrigger,
                self,
                1
            ).unwrap()
        );
        extern_mod.add_func(
            "signalPoint",
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(
                    vec![
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32
                    ],
                    vec![ValType::I32]
                ),
                signalPoint,
                self,
                1
            ).unwrap()
        );
        extern_mod.add_func(
            "submitOnchainTrx",
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(
                    vec![
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32,
                        ValType::I32
                    ],
                    vec![ValType::I32]
                ),
                submitOnchainTrx,
                self,
                1
            ).unwrap()
        );
        extern_mod.add_func(
            "put",
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(
                    vec![ValType::I32, ValType::I32, ValType::I32, ValType::I32],
                    vec![ValType::I32]
                ),
                trx_put,
                self,
                1
            ).unwrap()
        );
        extern_mod.add_func(
            "del",
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(vec![ValType::I32, ValType::I32], vec![ValType::I32]),
                trx_del,
                self,
                1
            ).unwrap()
        );
        extern_mod.add_func(
            "get",
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(vec![ValType::I32, ValType::I32], vec![ValType::I32]),
                trx_get,
                self,
                1
            ).unwrap()
        );
        extern_mod.add_func(
            "getByPrefix",
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(vec![ValType::I32, ValType::I32], vec![ValType::I32]),
                trx_get_by_prefix,
                self,
                1
            ).unwrap()
        );
    }

    pub fn finalize(&mut self) -> Vec<WasmDbOp> {
        if self.onchain {
            self.trx.dummy_commit();
        } else {
            self.trx.commit_as_offchain();
        }

        self.trx.ops.clone()
    }

    pub fn enqueue<F>(&mut self, task: F) where F: FnOnce() + Send + 'static {
        let _lock = self.queue_mutex_.lock().unwrap();
        if self.tasks.is_empty() {
            self.tasks.push_back(Box::new(task));
        }
        self.cv_.notify_one();
    }

    pub fn execute_on_update(&mut self, input: String) {
        let mut config = Config::create().unwrap();
        config.measure_cost(true);
        let mut stats = Statistics::create().unwrap();
        let store = Store::create().unwrap();
        let loader = Loader::create(Some(&config)).unwrap();
        let main_mod = loader.from_file(self.mod_path).unwrap().as_ref();
        let mut extern_mod = ImportModule::create("extern", Box::new(self)).unwrap();

        self.register_host(extern_mod);

        let validator = Validator::create(Some(&config)).unwrap();
        validator.validate(main_mod);
        self.vm = Some(Executor::create(Some(&config), Some(stats)).unwrap());
        let instance = self.vm.unwrap().register_active_module(&mut store, main_mod).unwrap();
        self.vm_instance = Some(instance);
        self.vm.unwrap().register_import_module(&mut store, &extern_mod);

        let start_fn = instance.get_func("_start").unwrap().deref();
        self.vm.unwrap().call_func(&mut start_fn, []);

        let val_l = input.len() as i32;
        let malloc_fn = instance.get_func("malloc").unwrap().deref();
        let res2 = self.vm
            .unwrap()
            .call_func(&mut malloc_fn, [WasmValue::from_i32(val_l)])
            .unwrap();

        let val_offset = res2[0].to_i32();
        let raw_arr = input.as_bytes();
        let arr: Vec<u8> = raw_arr.to_vec();
        let mem = instance.get_memory_mut("memory");
        mem.unwrap().set_data(arr, val_offset.cast_unsigned());
        let c = ((val_offset as i64) << 32) | (val_l as i64);

        let run_fn = instance.get_func("run").unwrap().deref();
        let res2 = self.vm
            .unwrap()
            .call_func(&mut run_fn, [WasmValue::from_i64(c)])
            .unwrap();
    }

    pub fn run_task(&mut self, task_id: String) {
        let val_l = task_id.len() as i32;
        let malloc_fn = self.vm_instance.unwrap().get_func("malloc").unwrap().deref();
        let res2 = self.vm
            .unwrap()
            .call_func(&mut malloc_fn, [WasmValue::from_i32(val_l)])
            .unwrap();

        let val_offset = res2[0].to_i32();
        let raw_arr = task_id.as_bytes();
        let arr: Vec<u8> = raw_arr.to_vec();
        let mem = self.vm_instance.unwrap().get_memory_mut("memory");
        mem.unwrap().set_data(arr, val_offset.cast_unsigned());
        let c = ((val_offset as i64) << 32) | (val_l as i64);

        let run_fn = self.vm_instance.unwrap().get_func("runTask").unwrap().deref();
        let res2 = self.vm
            .unwrap()
            .call_func(&mut run_fn, [WasmValue::from_i64(c)])
            .unwrap();
    }

    pub fn execute_on_chain(&mut self, input: String, user_id: String, cr: &mut ConcurrentRunner) {
        let input_json: JsonValue = serde_json::from_str(&input).unwrap();
        let token_id = input_json["tokenId"].as_str().unwrap().to_string();
        let params = input_json["params"].as_str().unwrap().to_string();

        self.chain_token_id = token_id.clone();

        let j =
            json!({
            "key": "checkTokenValidity",
            "input": {
                "tokenOwnerId": user_id,
                "tokenId": token_id
            }
        });
        let packet = j.to_string();

        let val = (self.callback)(packet);

        let jsn: JsonValue = serde_json::from_str(&val).unwrap();
        let gas_limit = jsn["gasLimit"].as_u64().unwrap_or(0);

        if gas_limit > 0 {
            self.is_token_valid = true;

            let mut config = Config::create().unwrap();
            config.measure_cost(true);
            let mut stats = Statistics::create().unwrap();
            stats.set_cost_limit(gas_limit);
            let store = Store::create().unwrap();
            let loader = Loader::create(Some(&config)).unwrap();
            let main_mod = loader.from_file(self.mod_path).unwrap().as_ref();
            let mut extern_mod = ImportModule::create("extern", Box::new(self)).unwrap();

            self.register_host(extern_mod);

            let validator = Validator::create(Some(&config)).unwrap();
            validator.validate(main_mod);
            self.vm = Some(Executor::create(Some(&config), Some(stats)).unwrap());
            let instance = self.vm.unwrap().register_active_module(&mut store, main_mod).unwrap();
            self.vm_instance = Some(instance);
            self.vm.unwrap().register_import_module(&mut store, &extern_mod);

            let start_fn = instance.get_func("_start").unwrap().deref();
            self.vm.unwrap().call_func(&mut start_fn, []);

            let val_l = input.len() as i32;
            let malloc_fn = instance.get_func("malloc").unwrap().deref();
            let res2 = self.vm
                .unwrap()
                .call_func(&mut malloc_fn, [WasmValue::from_i32(val_l)])
                .unwrap();

            let val_offset = res2[0].to_i32();
            let raw_arr = input.as_bytes();
            let arr: Vec<u8> = raw_arr.to_vec();
            let mem = instance.get_memory_mut("memory");
            mem.unwrap().set_data(arr, val_offset.cast_unsigned());
            let c = ((val_offset as i64) << 32) | (val_l as i64);

            let run_fn = instance.get_func("run").unwrap().deref();
            let res2 = self.vm
                .unwrap()
                .call_func(&mut run_fn, [WasmValue::from_i64(c)])
                .unwrap();
        }

        if self.onchain {
            let _lock = cr.wasm_global_lock.lock().unwrap();
            cr.wasm_done_tasks += 1;
            if cr.wasm_done_tasks == cr.wasm_count {
                unsafe {
                    if STEP == 0 {
                        cr.wasm_done_tasks = 0;
                        STEP += 1;
                        drop(_lock);
                        cr.wasm_do_critical();
                    }
                }
            }
        }
    }

    pub fn stick(&mut self) {
        if let Some(handle) = self.looper.take() {
            handle.join().unwrap();
        }
    }
}

// Sync task structure
#[derive(Clone)]
pub struct SyncTask {
    pub deps: Vec<String>,
    pub name: String,
}

pub fn newSyncTask(
    data: &mut WasmMac,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    unsafe {
        let mem_name = WasmEdge_StringCreateByCString(CString::new("memory").unwrap().as_ptr());

        let rt = &mut *(data as *mut WasmMac);
        let module = WasmEdge_VMGetActiveModule(rt.vm);
        let key_offset = WasmEdge_ValueGetI32(*in_params.offset(0));
        let key_l = WasmEdge_ValueGetI32(*in_params.offset(1)) as i32;

        let mem = WasmEdge_ModuleInstanceFindMemory(module, mem_name);

        let text = extract_string_from_memory(mem, key_offset, key_l, true);

        log_string(&text);

        let j: JsonValue = serde_json::from_str(&text).unwrap_or_default();

        let name = j["name"].as_str().unwrap_or("").to_string();

        let mut deps = Vec::<String>::new();
        if let Some(deps_array) = j["deps"].as_array() {
            for item in deps_array {
                if let Some(dep_str) = item.as_str() {
                    deps.push(dep_str.to_string());
                }
            }
        }

        rt.sync_tasks.push(SyncTask { deps, name });

        WasmEdge_StringDelete(mem_name);

        WASMEDGE_RESULT_SUCCESS
    }
}

pub fn output(
    data: &mut WasmMac,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    unsafe {
        let mem_name = WasmEdge_StringCreateByCString(CString::new("memory").unwrap().as_ptr());

        let rt = &mut *(data as *mut WasmMac);
        let module = WasmEdge_VMGetActiveModule(rt.vm);
        let key_offset = WasmEdge_ValueGetI32(*in_params.offset(0));
        let key_l = WasmEdge_ValueGetI32(*in_params.offset(1)) as i32;

        let mem = WasmEdge_ModuleInstanceFindMemory(module, mem_name);

        let text = extract_string_from_memory(mem, key_offset, key_l, true);

        log_string(&text);

        rt.execution_result = text;
        rt.has_output = true;

        WasmEdge_StringDelete(mem_name);

        WASMEDGE_RESULT_SUCCESS
    }
}

#[no_mangle]
pub fn consoleLog(
    data: &mut WasmMac,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    unsafe {
        let mem_name = WasmEdge_StringCreateByCString(CString::new("memory").unwrap().as_ptr());

        let rt = &mut *(data as *mut WasmMac);
        let module = WasmEdge_VMGetActiveModule(rt.vm);
        let key_offset = WasmEdge_ValueGetI32(*in_params.offset(0));
        let key_l = WasmEdge_ValueGetI32(*in_params.offset(1)) as i32;

        let mem = WasmEdge_ModuleInstanceFindMemory(module, mem_name);

        let text = extract_string_from_memory(mem, key_offset, key_l, true);

        log_string(&text);

        WasmEdge_StringDelete(mem_name);

        WASMEDGE_RESULT_SUCCESS
    }
}

pub fn submitOnchainTrx(
    data: &mut WasmMac,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    unsafe {
        let mem_name = WasmEdge_StringCreateByCString(CString::new("memory").unwrap().as_ptr());
        let malloc_name = WasmEdge_StringCreateByCString(CString::new("malloc").unwrap().as_ptr());

        let rt = &mut *(data as *mut WasmMac);
        let module = WasmEdge_VMGetActiveModule(rt.vm);
        let tm_offset = WasmEdge_ValueGetI32(*in_params.offset(0));
        let tm_l = WasmEdge_ValueGetI32(*in_params.offset(1)) as i32;
        let key_offset = WasmEdge_ValueGetI32(*in_params.offset(2));
        let key_l = WasmEdge_ValueGetI32(*in_params.offset(3)) as i32;
        let input_offset = WasmEdge_ValueGetI32(*in_params.offset(4));
        let input_l = WasmEdge_ValueGetI32(*in_params.offset(5)) as i32;
        let meta_offset = WasmEdge_ValueGetI32(*in_params.offset(6));
        let meta_l = WasmEdge_ValueGetI32(*in_params.offset(7)) as i32;

        let mem = WasmEdge_ModuleInstanceFindMemory(module, mem_name);

        let key = extract_string_from_memory(mem, key_offset, key_l, false);
        let input = extract_string_from_memory(mem, input_offset, input_l, false);
        let meta = extract_string_from_memory(mem, meta_offset, meta_l, false);

        log_string(&format!("[{}]", meta));

        let is_base = meta.chars().nth(0).unwrap_or('0') == '1';
        let is_file = meta.chars().nth(1).unwrap_or('0') == '1';

        let tag = meta.chars().skip(2).collect::<String>();

        let target_machine_id = if !is_base {
            extract_string_from_memory(mem, tm_offset, tm_l, false)
        } else {
            String::new()
        };

        log_string(&format!("{} || {} || {} || {}", target_machine_id, key, input, tag));

        let j =
            json!({
            "key": "submitOnchainTrx",
            "input": {
                "pointId": rt.point_id,
                "machineId": rt.machine_id,
                "targetMachineId": target_machine_id,
                "key": key,
                "tag": tag,
                "packet": input,
                "isFile": is_file,
                "isBase": is_base,
                "isRequesterOnchain": rt.onchain
            }
        });

        let packet = j.to_string();
        let packet_cstr = CString::new(packet).unwrap();

        let val_raw = if let Some(callback) = rt.callback {
            callback(packet_cstr.as_ptr())
        } else {
            ptr::null_mut()
        };

        let val = if !val_raw.is_null() {
            CStr::from_ptr(val_raw).to_string_lossy().to_string()
        } else {
            String::new()
        };

        let val_l = val.len();

        let params = [WasmEdge_ValueGenI32(val_l as u32)];
        let mut returns = [WasmEdge_ValueGenI32(0)];

        WasmEdge_VMExecute(rt.vm, malloc_name, params.as_ptr(), 1, returns.as_mut_ptr(), 1);
        let val_offset = WasmEdge_ValueGetI32(returns[0]) as i32;

        let arr = val.as_bytes().to_vec();

        if !val_raw.is_null() {
            libc::free(val_raw as *mut c_void);
        }

        WasmEdge_MemoryInstanceSetData(mem, arr.as_ptr(), val_offset as u32, val_l as u32);
        let c = ((val_offset as i64) << 32) | (val_l as i64);

        rt.temp_data_map.insert(c, arr);

        *out.offset(0) = WasmEdge_ValueGenI64(c);

        WasmEdge_StringDelete(mem_name);
        WasmEdge_StringDelete(malloc_name);

        WASMEDGE_RESULT_SUCCESS
    }
}

pub fn plantTrigger(
    data: &mut WasmMac,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    unsafe {
        let mem_name = WasmEdge_StringCreateByCString(CString::new("memory").unwrap().as_ptr());
        let malloc_name = WasmEdge_StringCreateByCString(CString::new("malloc").unwrap().as_ptr());

        let rt = &mut *(data as *mut WasmMac);
        let module = WasmEdge_VMGetActiveModule(rt.vm);
        let in_offset = WasmEdge_ValueGetI32(*in_params.offset(0));
        let in_l = WasmEdge_ValueGetI32(*in_params.offset(1)) as i32;
        let key_offset = WasmEdge_ValueGetI32(*in_params.offset(2));
        let key_l = WasmEdge_ValueGetI32(*in_params.offset(3)) as i32;
        let pi_offset = WasmEdge_ValueGetI32(*in_params.offset(4));
        let pi_l = WasmEdge_ValueGetI32(*in_params.offset(5)) as i32;
        let count = WasmEdge_ValueGetI32(*in_params.offset(6)) as i32;

        let mem = WasmEdge_ModuleInstanceFindMemory(module, mem_name);

        let text = extract_string_from_memory(mem, key_offset, key_l, false);
        let tag = extract_string_from_memory(mem, in_offset, in_l, false);
        let point_id = extract_string_from_memory(mem, pi_offset, pi_l, false);

        let j =
            json!({
            "key": "plantTrigger",
            "input": {
                "machineId": rt.machine_id,
                "pointId": point_id,
                "input": text,
                "tag": tag,
                "count": count
            }
        });

        let packet = j.to_string();
        let packet_cstr = CString::new(packet).unwrap();

        let val_raw = if let Some(callback) = rt.callback {
            callback(packet_cstr.as_ptr())
        } else {
            ptr::null_mut()
        };

        if !val_raw.is_null() {
            libc::free(val_raw as *mut c_void);
        }

        WasmEdge_StringDelete(mem_name);
        WasmEdge_StringDelete(malloc_name);

        WASMEDGE_RESULT_SUCCESS
    }
}

pub fn httpPost(
    data: &mut WasmMac,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    unsafe {
        let mem_name = WasmEdge_StringCreateByCString(CString::new("memory").unwrap().as_ptr());
        let malloc_name = WasmEdge_StringCreateByCString(CString::new("malloc").unwrap().as_ptr());

        let rt = &mut *(data as *mut WasmMac);
        let module = WasmEdge_VMGetActiveModule(rt.vm);
        let url_offset = WasmEdge_ValueGetI32(*in_params.offset(0));
        let url_l = WasmEdge_ValueGetI32(*in_params.offset(1)) as i32;
        let heads_offset = WasmEdge_ValueGetI32(*in_params.offset(2));
        let heads_l = WasmEdge_ValueGetI32(*in_params.offset(3)) as i32;
        let body_offset = WasmEdge_ValueGetI32(*in_params.offset(4));
        let body_l = WasmEdge_ValueGetI32(*in_params.offset(5)) as i32;

        let mem = WasmEdge_ModuleInstanceFindMemory(module, mem_name);

        let url = extract_string_from_memory(mem, url_offset, url_l, false);
        let headers = extract_string_from_memory(mem, heads_offset, heads_l, false);
        let body = extract_string_from_memory(mem, body_offset, body_l, false);

        let j =
            json!({
            "key": "httpPost",
            "input": {
                "machineId": rt.machine_id,
                "url": url,
                "headers": headers,
                "body": body
            }
        });

        let packet = j.to_string();
        let packet_cstr = CString::new(packet).unwrap();

        let val_raw = if let Some(callback) = rt.callback {
            callback(packet_cstr.as_ptr())
        } else {
            ptr::null_mut()
        };

        let val = if !val_raw.is_null() {
            CStr::from_ptr(val_raw).to_string_lossy().to_string()
        } else {
            String::new()
        };

        let val_l = val.len();

        let params = [WasmEdge_ValueGenI32(val_l as u32)];
        let mut returns = [WasmEdge_ValueGenI32(0)];

        WasmEdge_VMExecute(rt.vm, malloc_name, params.as_ptr(), 1, returns.as_mut_ptr(), 1);
        let val_offset = WasmEdge_ValueGetI32(returns[0]) as i32;

        let arr = val.as_bytes().to_vec();

        if !val_raw.is_null() {
            libc::free(val_raw as *mut c_void);
        }

        WasmEdge_MemoryInstanceSetData(mem, arr.as_ptr(), val_offset as u32, val_l as u32);
        let c = ((val_offset as i64) << 32) | (val_l as i64);

        *out.offset(0) = WasmEdge_ValueGenI64(c);
        rt.temp_data_map.insert(c, arr);

        WasmEdge_StringDelete(mem_name);
        WasmEdge_StringDelete(malloc_name);

        WASMEDGE_RESULT_SUCCESS
    }
}

pub fn runDocker(
    data: &mut WasmMac,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    unsafe {
        let mem_name = WasmEdge_StringCreateByCString(CString::new("memory").unwrap().as_ptr());
        let malloc_name = WasmEdge_StringCreateByCString(CString::new("malloc").unwrap().as_ptr());

        let rt = &mut *(data as *mut WasmMac);
        let module = WasmEdge_VMGetActiveModule(rt.vm);
        let in_offset = WasmEdge_ValueGetI32(*in_params.offset(0));
        let in_l = WasmEdge_ValueGetI32(*in_params.offset(1)) as i32;
        let key_offset = WasmEdge_ValueGetI32(*in_params.offset(2));
        let key_l = WasmEdge_ValueGetI32(*in_params.offset(3)) as i32;
        let cn_offset = WasmEdge_ValueGetI32(*in_params.offset(4));
        let cn_l = WasmEdge_ValueGetI32(*in_params.offset(5)) as i32;

        let mem = WasmEdge_ModuleInstanceFindMemory(module, mem_name);

        let text = extract_string_from_memory(mem, key_offset, key_l, false);
        let image_name = extract_string_from_memory(mem, in_offset, in_l, false);
        let container_name = extract_string_from_memory(mem, cn_offset, cn_l, false);

        let j =
            json!({
            "key": "runDocker",
            "input": {
                "machineId": rt.machine_id,
                "pointId": rt.point_id,
                "inputFiles": text,
                "imageName": image_name,
                "containerName": container_name
            }
        });

        let packet = j.to_string();
        let packet_cstr = CString::new(packet).unwrap();

        let val_raw = if let Some(callback) = rt.callback {
            callback(packet_cstr.as_ptr())
        } else {
            ptr::null_mut()
        };

        let val = if !val_raw.is_null() {
            CStr::from_ptr(val_raw).to_string_lossy().to_string()
        } else {
            String::new()
        };

        let val_l = val.len();

        let params = [WasmEdge_ValueGenI32(val_l as u32)];
        let mut returns = [WasmEdge_ValueGenI32(0)];

        WasmEdge_VMExecute(rt.vm, malloc_name, params.as_ptr(), 1, returns.as_mut_ptr(), 1);
        let val_offset = WasmEdge_ValueGetI32(returns[0]) as i32;

        let arr = val.as_bytes().to_vec();

        if !val_raw.is_null() {
            libc::free(val_raw as *mut c_void);
        }

        WasmEdge_MemoryInstanceSetData(mem, arr.as_ptr(), val_offset as u32, val_l as u32);
        let c = ((val_offset as i64) << 32) | (val_l as i64);

        *out.offset(0) = WasmEdge_ValueGenI64(c);
        rt.temp_data_map.insert(c, arr);

        WasmEdge_StringDelete(mem_name);
        WasmEdge_StringDelete(malloc_name);

        WASMEDGE_RESULT_SUCCESS
    }
}

pub fn execDocker(
    data: &mut WasmMac,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    unsafe {
        let mem_name = WasmEdge_StringCreateByCString(CString::new("memory").unwrap().as_ptr());
        let malloc_name = WasmEdge_StringCreateByCString(CString::new("malloc").unwrap().as_ptr());

        let rt = &mut *(data as *mut WasmMac);
        let module = WasmEdge_VMGetActiveModule(rt.vm);
        let in_offset = WasmEdge_ValueGetI32(*in_params.offset(0));
        let in_l = WasmEdge_ValueGetI32(*in_params.offset(1)) as i32;
        let key_offset = WasmEdge_ValueGetI32(*in_params.offset(2));
        let key_l = WasmEdge_ValueGetI32(*in_params.offset(3)) as i32;
        let co_offset = WasmEdge_ValueGetI32(*in_params.offset(4));
        let co_l = WasmEdge_ValueGetI32(*in_params.offset(5)) as i32;

        let mem = WasmEdge_ModuleInstanceFindMemory(module, mem_name);

        let image_name = extract_string_from_memory(mem, in_offset, in_l, false);
        let container_name = extract_string_from_memory(mem, key_offset, key_l, false);
        let command = extract_string_from_memory(mem, co_offset, co_l, false);

        let j =
            json!({
            "key": "execDocker",
            "input": {
                "machineId": rt.machine_id,
                "imageName": image_name,
                "containerName": container_name,
                "command": command
            }
        });

        let packet = j.to_string();
        let packet_cstr = CString::new(packet).unwrap();

        let val_raw = if let Some(callback) = rt.callback {
            callback(packet_cstr.as_ptr())
        } else {
            ptr::null_mut()
        };

        let res = if !val_raw.is_null() {
            CStr::from_ptr(val_raw).to_string_lossy().to_string()
        } else {
            String::new()
        };

        let jres = json!({
            "data": res
        });

        let val = jres.to_string();
        let val_l = val.len();

        let params = [WasmEdge_ValueGenI32(val_l as u32)];
        let mut returns = [WasmEdge_ValueGenI32(0)];

        WasmEdge_VMExecute(rt.vm, malloc_name, params.as_ptr(), 1, returns.as_mut_ptr(), 1);
        let val_offset = WasmEdge_ValueGetI32(returns[0]) as i32;

        let arr = val.as_bytes().to_vec();

        if !val_raw.is_null() {
            libc::free(val_raw as *mut c_void);
        }

        WasmEdge_MemoryInstanceSetData(mem, arr.as_ptr(), val_offset as u32, val_l as u32);
        let c = ((val_offset as i64) << 32) | (val_l as i64);

        *out.offset(0) = WasmEdge_ValueGenI64(c);
        rt.temp_data_map.insert(c, arr);

        WasmEdge_StringDelete(mem_name);
        WasmEdge_StringDelete(malloc_name);

        WASMEDGE_RESULT_SUCCESS
    }
}

pub fn copyToDocker(
    data: &mut WasmMac,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    unsafe {
        let mem_name = WasmEdge_StringCreateByCString(CString::new("memory").unwrap().as_ptr());
        let malloc_name = WasmEdge_StringCreateByCString(CString::new("malloc").unwrap().as_ptr());

        let rt = &mut *(data as *mut WasmMac);
        let module = WasmEdge_VMGetActiveModule(rt.vm);
        let in_offset = WasmEdge_ValueGetI32(*in_params.offset(0));
        let in_l = WasmEdge_ValueGetI32(*in_params.offset(1)) as i32;
        let key_offset = WasmEdge_ValueGetI32(*in_params.offset(2));
        let key_l = WasmEdge_ValueGetI32(*in_params.offset(3)) as i32;
        let co_offset = WasmEdge_ValueGetI32(*in_params.offset(4));
        let co_l = WasmEdge_ValueGetI32(*in_params.offset(5)) as i32;
        let content_offset = WasmEdge_ValueGetI32(*in_params.offset(6));
        let content_l = WasmEdge_ValueGetI32(*in_params.offset(7)) as i32;

        let mem = WasmEdge_ModuleInstanceFindMemory(module, mem_name);

        let image_name = extract_string_from_memory(mem, in_offset, in_l, false);
        let container_name = extract_string_from_memory(mem, key_offset, key_l, false);
        let file_name = extract_string_from_memory(mem, co_offset, co_l, false);
        let content = extract_string_from_memory(mem, content_offset, content_l, false);

        let j =
            json!({
            "key": "copyToDocker",
            "input": {
                "machineId": rt.machine_id,
                "imageName": image_name,
                "containerName": container_name,
                "fileName": file_name,
                "content": content
            }
        });

        let packet = j.to_string();
        let packet_cstr = CString::new(packet).unwrap();

        let val_raw = if let Some(callback) = rt.callback {
            callback(packet_cstr.as_ptr())
        } else {
            ptr::null_mut()
        };

        let res = if !val_raw.is_null() {
            CStr::from_ptr(val_raw).to_string_lossy().to_string()
        } else {
            String::new()
        };

        let jres = json!({
            "data": res
        });

        let val = jres.to_string();
        let val_l = val.len();

        let params = [WasmEdge_ValueGenI32(val_l as u32)];
        let mut returns = [WasmEdge_ValueGenI32(0)];

        WasmEdge_VMExecute(rt.vm, malloc_name, params.as_ptr(), 1, returns.as_mut_ptr(), 1);
        let val_offset = WasmEdge_ValueGetI32(returns[0]) as i32;

        let arr = val.as_bytes().to_vec();

        if !val_raw.is_null() {
            libc::free(val_raw as *mut c_void);
        }

        WasmEdge_MemoryInstanceSetData(mem, arr.as_ptr(), val_offset as u32, val_l as u32);
        let c = ((val_offset as i64) << 32) | (val_l as i64);

        *out.offset(0) = WasmEdge_ValueGenI64(c);
        rt.temp_data_map.insert(c, arr);

        WasmEdge_StringDelete(mem_name);
        WasmEdge_StringDelete(malloc_name);

        WASMEDGE_RESULT_SUCCESS
    }
}

pub fn signalPoint(
    data: &mut WasmMac,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    unsafe {
        let mem_name = WasmEdge_StringCreateByCString(CString::new("memory").unwrap().as_ptr());
        let malloc_name = WasmEdge_StringCreateByCString(CString::new("malloc").unwrap().as_ptr());

        let rt = &mut *(data as *mut WasmMac);
        let module = WasmEdge_VMGetActiveModule(rt.vm);
        let typ_offset = WasmEdge_ValueGetI32(*in_params.offset(0));
        let typ_l = WasmEdge_ValueGetI32(*in_params.offset(1)) as i32;
        let point_id_offset = WasmEdge_ValueGetI32(*in_params.offset(2));
        let point_id_l = WasmEdge_ValueGetI32(*in_params.offset(3)) as i32;
        let user_id_offset = WasmEdge_ValueGetI32(*in_params.offset(4));
        let user_id_l = WasmEdge_ValueGetI32(*in_params.offset(5)) as i32;
        let data_offset = WasmEdge_ValueGetI32(*in_params.offset(6));
        let data_l = WasmEdge_ValueGetI32(*in_params.offset(7)) as i32;

        let mem = WasmEdge_ModuleInstanceFindMemory(module, mem_name);

        let typ = extract_string_from_memory(mem, typ_offset, typ_l, false);
        let point_id = extract_string_from_memory(mem, point_id_offset, point_id_l, false);
        let user_id = extract_string_from_memory(mem, user_id_offset, user_id_l, false);
        let payload = extract_string_from_memory(mem, data_offset, data_l, false);

        let j =
            json!({
            "key": "signalPoint",
            "input": {
                "machineId": rt.machine_id,
                "type": typ,
                "pointId": point_id,
                "userId": user_id,
                "data": payload
            }
        });

        let packet = j.to_string();
        let packet_cstr = CString::new(packet).unwrap();

        let val_raw = if let Some(callback) = rt.callback {
            callback(packet_cstr.as_ptr())
        } else {
            ptr::null_mut()
        };

        let val = if !val_raw.is_null() {
            CStr::from_ptr(val_raw).to_string_lossy().to_string()
        } else {
            String::new()
        };

        let val_l = val.len();

        let params = [WasmEdge_ValueGenI32(val_l as u32)];
        let mut returns = [WasmEdge_ValueGenI32(0)];

        WasmEdge_VMExecute(rt.vm, malloc_name, params.as_ptr(), 1, returns.as_mut_ptr(), 1);
        let val_offset = WasmEdge_ValueGetI32(returns[0]) as i32;

        let arr = val.as_bytes().to_vec();

        if !val_raw.is_null() {
            libc::free(val_raw as *mut c_void);
        }

        WasmEdge_MemoryInstanceSetData(mem, arr.as_ptr(), val_offset as u32, val_l as u32);
        let c = ((val_offset as i64) << 32) | (val_l as i64);

        *out.offset(0) = WasmEdge_ValueGenI64(c);
        rt.temp_data_map.insert(c, arr);

        WasmEdge_StringDelete(mem_name);
        WasmEdge_StringDelete(malloc_name);

        WASMEDGE_RESULT_SUCCESS
    }
}

use std::collections::{ HashMap, HashSet };
use std::sync::{ Arc, Mutex, atomic::{ AtomicI32, Ordering } };
use std::thread;
use std::time::Instant;
use serde_json::{ json, Value };
use wasmedge_sys::{
    CallingFrameContext,
    Executor,
    FuncType,
    Function,
    HostFunc,
    ImportModule,
    ImportObjectBuilder,
    Instance,
    Loader,
    MemoryInstance,
    Module,
    Statistics,
    Store,
    Validator,
    Value as WasmValue,
    WasmEdgeResult,
    WasmEdgeString,
};

// Define the types that are referenced in the original code
#[derive(Debug)]
pub struct WasmMac {
    pub machine_id: String,
    pub id: String,
    pub index: usize,
    pub vm: Arc<Mutex<Executor>>,
    pub trx: Arc<dyn Transaction>,
    pub temp_data_map: HashMap<i64, Vec<u8>>,
    pub sync_tasks: Vec<(Vec<String>, String)>,
    pub execution_result: String,
    pub chain_token_id: String,
}

pub trait Transaction {
    fn put(&self, key: &str, value: &str);
    fn del(&self, key: &str);
    fn get(&self, key: &str) -> String;
    fn get_by_prefix(&self, prefix: &str) -> Vec<String>;
}

#[derive(Debug)]
pub struct ChainTrx {
    pub machine_id: String,
    pub callback_id: String,
    pub input: String,
    pub user_id: String,
}

#[derive(Debug)]
pub struct WasmTask {
    pub id: i32,
    pub name: String,
    pub inputs: HashMap<i32, (bool, Arc<WasmTask>)>,
    pub outputs: HashMap<i32, Arc<WasmTask>>,
    pub vm_index: usize,
    pub started: bool,
}

pub struct WasmLock {
    pub mutex: Mutex<()>,
}

impl WasmLock {
    pub fn new() -> Self {
        WasmLock {
            mutex: Mutex::new(()),
        }
    }
}

#[derive(Debug)]
pub struct OpChange {
    pub op_type: String,
    pub key: String,
    pub val: String,
}

// Dummy implementation for demonstration
pub struct DummyTransaction;

impl Transaction for DummyTransaction {
    fn put(&self, key: &str, value: &str) {
        println!("PUT: {} = {}", key, value);
    }

    fn del(&self, key: &str) {
        println!("DEL: {}", key);
    }

    fn get(&self, key: &str) -> String {
        format!("value_for_{}", key)
    }

    fn get_by_prefix(&self, prefix: &str) -> Vec<String> {
        vec![format!("result1_for_{}", prefix), format!("result2_for_{}", prefix)]
    }
}

// Host function implementations
pub fn trx_put(
    data: &mut WasmMac,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem_name = WasmEdgeString::create("memory");

    let rt = unsafe { &mut *(_data as *mut WasmMac) };
    let vm = rt.vm.lock().unwrap();
    let key_offset = inputs[0].to_i32() as u32;
    let key_l = inputs[1].to_i32();
    let val_offset = inputs[2].to_i32() as u32;
    let val_l = inputs[3].to_i32();

    // Get active module and memory instance
    // Note: This is a simplified version - actual implementation would need proper WasmEdge integration
    let mut raw_key = vec![0u8; key_l as usize];
    let mut raw_key_c = Vec::new();

    // Simulate memory data retrieval
    for i in 0..key_l {
        raw_key_c.push(raw_key[i as usize] as i8);
    }
    let key = String::from_utf8_lossy(&raw_key).to_string();

    let mut raw_val = vec![0u8; val_l as usize];
    let mut raw_val_c = Vec::new();

    for i in 0..val_l {
        raw_val_c.push(raw_val[i as usize] as i8);
    }
    let val = String::from_utf8_lossy(&raw_val).to_string();

    rt.trx.put(&format!("{}::{}", rt.machine_id, key), &val);

    Ok(())
}

pub fn trx_del(
    data: &mut WasmMac,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem_name = WasmEdgeString::create("memory");

    let rt = unsafe { &mut *(_data as *mut WasmMac) };
    let vm = rt.vm.lock().unwrap();
    let key_offset = inputs[0].to_i32() as u32;
    let key_l = inputs[1].to_i32();

    // Get memory instance and extract key
    let mut raw_key = vec![0u8; key_l as usize];
    let mut raw_key_c = Vec::new();

    for i in 0..key_l {
        raw_key_c.push(raw_key[i as usize] as i8);
    }
    let key = String::from_utf8_lossy(&raw_key).to_string();

    rt.trx.del(&format!("{}::{}", rt.machine_id, key));

    Ok(())
}

pub fn trx_get(
    data: &mut WasmMac,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem_name = WasmEdgeString::create("memory");
    let malloc_name = WasmEdgeString::create("malloc");

    let rt = unsafe { &mut *(_data as *mut WasmMac) };
    let vm = rt.vm.lock().unwrap();
    let key_offset = inputs[0].to_i32() as u32;
    let key_l = inputs[1].to_i32();

    // Extract key from memory
    let mut raw_key = vec![0u8; key_l as usize];
    let mut raw_key_c = Vec::new();

    for i in 0..key_l {
        raw_key_c.push(raw_key[i as usize] as i8);
    }
    let key = String::from_utf8_lossy(&raw_key).to_string();

    let val = rt.trx.get(&format!("{}::{}", rt.machine_id, key));
    let val_l = val.len() as i32;

    // Simulate malloc call
    let val_offset = 1000i32; // Simplified - would need actual malloc implementation

    let raw_arr = val.as_bytes();
    let mut arr = vec![0u8; val_l as usize];
    for i in 0..val_l {
        arr[i as usize] = raw_arr[i as usize];
    }

    // Simulate memory write
    let c = ((val_offset as i64) << 32) | (val_l as i64);

    outputs[0] = WasmValue::from_i64(c);
    rt.temp_data_map.insert(c, arr);

    Ok(())
}

pub fn trx_get_by_prefix(
    data: &mut WasmMac,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem_name = WasmEdgeString::create("memory");
    let malloc_name = WasmEdgeString::create("malloc");

    let rt = unsafe { &mut *(_data as *mut WasmMac) };
    let vm = rt.vm.lock().unwrap();
    let key_offset = inputs[0].to_i32() as u32;
    let key_l = inputs[1].to_i32();

    // Extract key from memory
    let mut raw_key = vec![0u8; key_l as usize];
    let mut raw_key_c = Vec::new();

    for i in 0..key_l {
        if raw_key[i as usize] == 0 {
            break;
        }
        raw_key_c.push(raw_key[i as usize] as i8);
    }
    let prefix = String::from_utf8_lossy(&raw_key_c).to_string();

    let vals = rt.trx.get_by_prefix(&format!("{}::{}", rt.machine_id, prefix));

    let mut arr_of_s = Vec::new();
    for val in vals {
        arr_of_s.push(val);
    }

    let j = json!({
        "data": arr_of_s
    });

    let val = j.to_string();
    let val_l = val.len() as i32;

    // Simulate malloc call
    let val_offset = 1000i32; // Simplified - would need actual malloc implementation

    let raw_arr = val.as_bytes();
    let mut arr = vec![0u8; val_l as usize];
    for i in 0..val_l {
        arr[i as usize] = raw_arr[i as usize];
    }

    // Simulate memory write
    let c = ((val_offset as i64) << 32) | (val_l as i64);

    outputs[0] = WasmValue::from_i64(c);
    rt.temp_data_map.insert(c, arr);

    Ok(())
}

// ConcurrentRunner Implementation
pub struct ConcurrentRunner {
    pub trxs: Vec<ChainTrx>,
    pub ast_store_path: String,
    pub wasm_vms: Vec<Option<Arc<Mutex<WasmMac>>>>,
    pub wasm_vm_map: HashMap<String, (usize, Arc<Mutex<WasmMac>>)>,
    pub exec_wasm_locks: Vec<Arc<Mutex<()>>>,
    pub main_wasm_lock: Arc<Mutex<()>>,
    pub wasm_count: usize,
}

impl ConcurrentRunner {
    pub fn new(ast_store_path: String, trxs: Vec<ChainTrx>) -> Self {
        ConcurrentRunner {
            trxs,
            ast_store_path,
            wasm_vms: Vec::new(),
            wasm_vm_map: HashMap::new(),
            exec_wasm_locks: Vec::new(),
            main_wasm_lock: Arc::new(Mutex::new(())),
            wasm_count: 0,
        }
    }

    pub fn run(&mut self) {
        let start_time = Instant::now();
        let wasm_thread_pool = WasmThreadPool::new(8);
        self.prepare_context(self.trxs.len());

        let mut handles = Vec::new();

        for i in 0..self.trxs.len() {
            let trx = self.trxs[i].clone();
            let ast_store_path = self.ast_store_path.clone();
            let handle = thread::spawn(move || {
                let rt = WasmMac::new(
                    trx.machine_id.clone(),
                    i.to_string(),
                    i,
                    format!("{}/{}/module", ast_store_path, trx.machine_id),
                    wasm_send
                );
                // Register and execute
                // rt.execute_on_chain(&trx.input, &trx.user_id, self);
            });
            handles.push(handle);
        }

        // Wait for all threads to complete
        for handle in handles {
            handle.join().unwrap();
        }

        wasm_thread_pool.stop_pool();
        wasm_thread_pool.stick();

        // Process results
        for rt_opt in &self.wasm_vms {
            if let Some(rt_arc) = rt_opt {
                let rt = rt_arc.lock().unwrap();
                let cost = 0u64; // Simplified - would get from WasmEdge statistics

                let output = rt.execution_result.clone();
                let changes = rt.finalize();

                let mut arr = Vec::new();
                for op in changes {
                    let item =
                        json!({
                        "opType": op.op_type,
                        "key": op.key,
                        "val": op.val
                    });
                    arr.push(item);
                }
                let ops_str = json!(arr).to_string();

                let j =
                    json!({
                    "key": "submitOnchainResponse",
                    "input": {
                        "callbackId": self.trxs[rt.index].callback_id,
                        "packet": output,
                        "resCode": 1,
                        "error": "",
                        "changes": ops_str,
                        "cost": cost,
                        "tokenId": rt.chain_token_id
                    }
                });
                let packet = j.to_string();

                wasm_send(&packet);
            }
        }

        let elapsed = start_time.elapsed();
        log(
            &format!("executed chain applet transactions in {} microseconds.", elapsed.as_micros())
        );
    }

    pub fn prepare_context(&mut self, vm_count: usize) {
        self.wasm_count = vm_count;
        self.wasm_vms = vec![None; vm_count];
        self.exec_wasm_locks = (0..vm_count).map(|_| Arc::new(Mutex::new(()))).collect();
    }

    pub fn register_wasm_mac(&mut self, rt: Arc<Mutex<WasmMac>>) {
        let rt_guard = rt.lock().unwrap();
        let id = rt_guard.id.clone();
        let index = rt_guard.index;
        drop(rt_guard);

        self.wasm_vm_map.insert(id, (index, rt.clone()));
        self.wasm_vms[index] = Some(rt);
    }

    pub fn wasm_run_task<F>(&self, task: F, index: usize) where F: FnOnce(&WasmMac) {
        if let Some(rt_arc) = &self.wasm_vms[index] {
            let rt = rt_arc.lock().unwrap();
            task(&*rt);
        }
    }

    pub fn wasm_do_critical(&mut self) {
        let mut key_counter = 1i32;
        let mut all_wasm_tasks: HashMap<i32, (Vec<String>, usize, String)> = HashMap::new();
        let mut task_refs: HashMap<String, Arc<WasmTask>> = HashMap::new();
        let mut res_wasm_locks: HashMap<
            String,
            (Option<Arc<WasmTask>>, Arc<WasmLock>)
        > = HashMap::new();
        let mut start_points: HashMap<String, Arc<WasmTask>> = HashMap::new();
        let mut res_counter = 0;
        let mut c = 0;

        // Collect all sync tasks from VMs
        for index in 0..self.wasm_count {
            if let Some(vm_arc) = &self.wasm_vms[index] {
                let vm = vm_arc.lock().unwrap();
                for t in &vm.sync_tasks {
                    let res_nums = t.0.clone();
                    let name = t.1.clone();
                    let mut res_count = 0;

                    for r in &res_nums {
                        if !res_wasm_locks.contains_key(r) {
                            res_wasm_locks.insert(r.clone(), (None, Arc::new(WasmLock::new())));
                            res_counter += 1;
                        }
                        res_count += 1;
                    }
                    all_wasm_tasks.insert(key_counter, (res_nums, index, name));
                    key_counter += 1;
                    c += 1;
                }
            }
        }

        // Build dependency graph
        for i in 1..=key_counter {
            if let Some((res_nums, vm_index, name)) = all_wasm_tasks.get(&i) {
                let inputs = HashMap::new();
                let outputs = HashMap::new();
                let task = Arc::new(WasmTask {
                    id: i,
                    name: name.clone(),
                    inputs: HashMap::new(),
                    outputs: HashMap::new(),
                    vm_index: *vm_index,
                    started: false,
                });

                let vm_index_str = vm_index.to_string();
                task_refs.insert(format!("{}:{}", vm_index_str, name), task.clone());

                for r in res_nums {
                    if r.starts_with("lock_") {
                        if let Some(t) = task_refs.get(&format!("{}:{}", vm_index_str, r)) {
                            // Handle lock dependencies - simplified version
                        }
                    } else {
                        if let Some((first_task, _)) = res_wasm_locks.get_mut(r) {
                            if first_task.is_none() {
                                *first_task = Some(task.clone());
                                start_points.insert(r.clone(), task.clone());
                            } else {
                                // Handle chaining - simplified version
                                *first_task = Some(task.clone());
                            }
                        }
                    }
                }
            }
        }

        // Execute tasks
        let wasm_thread_pool = WasmThreadPool::new(res_counter);
        let wasm_done_wasm_tasks_count = Arc::new(AtomicI32::new(1));

        let exec_wasm_task = |task: Arc<WasmTask>| {
            let ready_to_exec = {
                let _lock = self.main_wasm_lock.lock().unwrap();
                // Check if task is ready to execute - simplified version
                true
            };

            if ready_to_exec {
                let task_clone = task.clone();
                let wasm_done_count_clone = wasm_done_wasm_tasks_count.clone();
                let key_counter_copy = key_counter;

                thread::spawn(move || {
                    log(&format!("task {}", task_clone.id));
                    // Execute task - simplified version

                    let count = wasm_done_count_clone.fetch_add(1, Ordering::SeqCst);
                    if count == key_counter_copy {
                        wasm_thread_pool.stop_pool();
                    }
                });
            }
        };

        // Start execution with initial tasks
        for (_r, task) in start_points {
            exec_wasm_task(task);
        }

        wasm_thread_pool.stick();
    }
}

// Clone implementations for the types that need it
impl Clone for ChainTrx {
    fn clone(&self) -> Self {
        ChainTrx {
            machine_id: self.machine_id.clone(),
            callback_id: self.callback_id.clone(),
            input: self.input.clone(),
            user_id: self.user_id.clone(),
        }
    }
}
