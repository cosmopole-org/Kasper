use std::cell::{ RefCell };
use std::collections::{ HashMap, VecDeque, BTreeMap };
use std::ops::{ DerefMut };
use std::rc::Rc;
use std::sync::{ Arc, Condvar, Mutex };
use std::{ thread };
use std::sync::atomic::{ AtomicBool, AtomicI64, Ordering };
use serde_json::{ json, Value as JsonValue };
use wasmedge_sys::{ Validator };
use wasmedge_sys::{
    Executor,
    Function,
    ImportModule,
    Instance,
    Loader,
    Statistics,
    Store,
    config::Config,
    AsInstance,
    CallingFrame,
    WasmValue,
};

use wasmedge_types::{ ValType, error::CoreError };

use once_cell::sync::Lazy;

use rocksdb::{
    Options,
    ReadOptions,
    TransactionDB,
    TransactionDBOptions,
    TransactionOptions,
    WriteOptions,
};

use blockingqueue::BlockingQueue;

static RESP_MAP: Lazy<Arc<Mutex<HashMap<i64, String>>>> = Lazy::new(|| {
    Arc::new(Mutex::new(HashMap::new()))
});
static TRIGGER_MAP: Lazy<Arc<Mutex<HashMap<i64, Arc<Condvar>>>>> = Lazy::new(|| {
    Arc::new(Mutex::new(HashMap::new()))
});
static REQ_ID_COUNTER: Lazy<AtomicI64> = Lazy::new(|| { AtomicI64::new(0) });
static GLOBAL_REQ_CHAN: Lazy<BlockingQueue<String>> = Lazy::new(|| { BlockingQueue::new() });

fn main() {
    let receiver_handler = thread::spawn(|| {
        let context = zmq::Context::new();
        let responder = Arc::new(Mutex::new(context.socket(zmq::REP).unwrap()));
        {
            assert!(responder.lock().unwrap().bind("tcp://*:5556").is_ok());
        }
        let mut msg = zmq::Message::new();
        crossbeam
            ::scope(|s| {
                loop {
                    {
                        responder.lock().unwrap().recv(&mut msg, 0).unwrap();
                    }
                    let data = msg.as_str().unwrap();
                    println!("recevied {data}");
                    let packet: JsonValue = serde_json::from_str(data).unwrap();
                    if packet["type"] == "runOffChain" {
                        s.spawn(move |_| {
                            let ast_path = packet["astPath"].as_str().unwrap().to_string();
                            let input = packet["input"].as_str().unwrap().to_string();
                            let machine_id = packet["machineId"].as_str().unwrap().to_string();
                            wasm_run_vm(ast_path, input, machine_id);
                        });
                    } else if packet["type"] == "apiResponse" {
                        let request_id = packet["requestId"].as_i64().unwrap();
                        RESP_MAP.lock()
                            .unwrap()
                            .insert(request_id, packet["data"].as_str().unwrap().to_string());
                        TRIGGER_MAP.lock().unwrap().get(&request_id).unwrap().notify_one();
                    }
                    {
                        responder.lock().unwrap().send("", 0).unwrap();
                    }
                }
            })
            .unwrap();
    });
    let chan = GLOBAL_REQ_CHAN.clone();
    let sender_handler = thread::spawn(move || {
        println!("Connecting to host platform server...\n");
        let context = zmq::Context::new();
        let requester = context.socket(zmq::REQ).unwrap();
        assert!(requester.connect("tcp://localhost:5555").is_ok());
        let mut msg = zmq::Message::new();
        loop {
            let packet = chan.pop();
            requester.send(&packet, 0).unwrap();
            requester.recv(&mut msg, 0).unwrap();
        }
    });
    receiver_handler.join().unwrap();
    sender_handler.join().unwrap();
}

pub fn wasm_run_vm(ast_path: String, input: String, machine_id: String) {
    let inp1 = input.clone();
    let input_json: JsonValue = serde_json::from_str(&inp1).unwrap();
    let point_id = input_json["point"].as_object().unwrap()["id"].as_str().unwrap().to_string();
    let mut rt = WasmMac::new_offchain(
        machine_id.clone(),
        point_id,
        ast_path.clone(),
        Box::new(wasm_send)
    );
    rt.execute_on_update(inp1);
}

static GLOBAL_DB: Lazy<Arc<Mutex<TransactionDB>>> = Lazy::new(|| {
    let path = "appletdb";
    let _ = std::fs::remove_dir_all(path);

    let mut db_options = Options::default();
    db_options.create_if_missing(true);

    let txn_db_options = TransactionDBOptions::default();

    let db = TransactionDB::open(&db_options, &txn_db_options, path).unwrap();
    Arc::new(Mutex::new(db))
});

static mut STEP: i32 = 0;

fn wasm_send(mut data: JsonValue) -> std::string::String {
    let req_id = REQ_ID_COUNTER.fetch_add(1, Ordering::Acquire);
    data["requestId"] = JsonValue::from(req_id);
    let cv_ = Arc::new(Condvar::new());
    {
        let cv_clone = Arc::clone(&cv_);
        TRIGGER_MAP.lock().unwrap().insert(req_id, cv_clone);
    }
    {
        GLOBAL_REQ_CHAN.clone().push(data.to_string());
    }
    let triggers = Arc::clone(&RESP_MAP);
    let res = {
        {
            let triggers_ref = triggers.lock().unwrap();
            let _mg: std::sync::MutexGuard<'_, HashMap<i64, String>> = cv_
                .wait_while(triggers_ref, |tr| { tr.is_empty() && !tr.contains_key(&req_id) })
                .unwrap();
        }
        let t = triggers.lock().unwrap().remove(&req_id);
        let t_final = if t.is_none() { "".to_string() } else { t.unwrap().clone() };
        t_final
    };
    res.to_string()
}

fn log(text: String) {
    let j = json!({
        "key": "log",
        "input": {
            "text": text
        }
    });
    wasm_send(j);
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
    pub inputs: Arc<Mutex<HashMap<i32, (bool, Arc<Mutex<WasmTask>>)>>>,
    pub outputs: Arc<Mutex<HashMap<i32, Arc<Mutex<WasmTask>>>>>,
    pub vm_index: i32,
    pub started: bool,
}

impl WasmTask {
    pub fn new() -> Self {
        WasmTask {
            id: 0,
            name: String::new(),
            inputs: Arc::new(Mutex::new(HashMap::new())),
            outputs: Arc::new(Mutex::new(HashMap::new())),
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
        Trx {
            write_options: WriteOptions::default(),
            read_options: ReadOptions::default(),
            txn_options: TransactionOptions::default(),
            store: BTreeMap::new(),
            newly_created: BTreeMap::new(),
            newly_deleted: BTreeMap::new(),
            ops: Vec::new(),
        }
    }

    pub fn get_bytes_of_str(&self, str_val: String) -> Vec<u8> {
        let mut bytes: Vec<u8> = str_val.into_bytes();
        bytes.push(0);
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
        for (key, value) in &self.store {
            if WasmUtils::startswith(key, &prefix) {
                result.push(value.clone());
            }
        }
        for t in GLOBAL_DB.lock().unwrap().prefix_iterator(prefix.as_bytes().to_vec().as_slice()) {
            let item = t.unwrap();
            let key = str::from_utf8(item.0.to_vec().as_slice()).unwrap().to_string();
            let val = str::from_utf8(item.1.to_vec().as_slice()).unwrap().to_string();
            if !self.store.contains_key(&key) && !self.store.contains_key(&key) {
                result.push(val);
            }
        }
        result
    }

    pub fn get(&mut self, key: String) -> String {
        if let Some(value) = self.store.get(&key) {
            value.clone()
        } else if self.newly_deleted.contains_key(&key) {
            return "".to_string();
        } else {
            let raw_val = GLOBAL_DB.lock().unwrap().get(key.as_bytes().to_vec().as_slice());
            let value: String;
            if raw_val.is_ok() {
                value = if let Some(val) = raw_val.unwrap() {
                    str::from_utf8(val.as_slice()).unwrap().to_string()
                } else {
                    "".to_string()
                };
            } else {
                value = "".to_string();
            }
            self.store.insert(key.clone(), value.clone());
            value
        }
    }

    pub fn del(&mut self, key: String) {
        let k = String::from(key);
        self.ops.push(WasmDbOp {
            type_: "del".to_string(),
            key: k.clone(),
            val: String::new(),
        });
        self.store.remove(&k);
        self.newly_created.remove(&k);
        self.newly_deleted.insert(k, true);
    }

    pub fn commit_as_offchain(&mut self) {
        let global_db = GLOBAL_DB.lock().unwrap();
        let trx = global_db.transaction();
        for op in &self.ops {
            if op.type_ == "put" {
                trx.put(&op.key, &op.val).unwrap();
            } else if op.type_ == "del" {
                trx.delete(&op.key).unwrap();
            }
        }
        trx.commit().unwrap();
        log("committed transaction successfully.".to_string());
    }

    pub fn dummy_commit(&mut self) {}
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
        let vd: VecDeque<Box<dyn FnOnce() + Send>> = VecDeque::new();
        let tasks_: Arc<Mutex<VecDeque<Box<dyn FnOnce() + Send>>>> = Arc::new(Mutex::new(vd));
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
                        let tasks = tasks_clone.lock().unwrap();
                        let _mg: std::sync::MutexGuard<
                            '_,
                            VecDeque<Box<dyn FnOnce() + Send>>
                        > = cv_clone
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

// WasmMac Implementation: -------------------------------------------------
pub struct WasmMac {
    pub onchain: bool,
    pub chain_token_id: String,
    pub is_token_valid: bool,
    pub callback: Box<dyn (Fn(JsonValue) -> String) + Send + Sync>,
    pub machine_id: String,
    pub point_id: String,
    pub id: String,
    pub index: i32,
    pub trx: Box<Trx>,
    pub looper: Option<thread::JoinHandle<()>>,
    pub tasks: VecDeque<Box<dyn FnOnce() + Send>>,
    pub sync_tasks: Vec<SyncTask>,
    pub queue_mutex_: Mutex<()>,
    pub cv_: Condvar,
    pub stop_: bool,
    pub mod_path: String,
    pub vm_executor: Option<Rc<RefCell<Executor>>>,
    pub vm_instance: Option<Rc<RefCell<Instance>>>,
    pub vm_stats: Option<Arc<Mutex<Statistics>>>,

    execution_result: String,
    has_output: bool,
}

pub struct HostData {
    runtime: *mut WasmMac,
    exec: *mut Executor,
}

impl WasmMac {
    fn prepare_looper(&mut self) {
        // Create a channel for communication instead of directly accessing self
        let (_, receiver) = std::sync::mpsc::channel::<Box<dyn FnOnce() + Send>>();

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
        cb: Box<dyn (Fn(JsonValue) -> String) + Send + Sync>
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
            sync_tasks: Vec::new(),
            queue_mutex_: Mutex::new(()),
            cv_: Condvar::new(),
            stop_: false,
            mod_path,
            execution_result: "".to_string(),
            has_output: false,
            vm_executor: None,
            vm_instance: None,
            vm_stats: None,
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
        cb: Box<dyn (Fn(JsonValue) -> String) + Send + Sync>
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
            sync_tasks: Vec::new(),
            queue_mutex_: Mutex::new(()),
            cv_: Condvar::new(),
            stop_: false,
            mod_path,
            execution_result: "".to_string(),
            has_output: false,
            vm_executor: None,
            vm_instance: None,
            vm_stats: None,
        };

        if wasm_mac.onchain {
            wasm_mac.prepare_looper();
        }
        wasm_mac
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
        let stats = Statistics::create().unwrap();
        let mut store = Store::create().unwrap();

        let wasi_mod = wasmedge_sys::WasiModule::create(None, None, None).unwrap();

        let mut dummy: i32 = 1;
        let extern_mod = &mut ImportModule::create("env", Box::new(&mut dummy)).unwrap();

        let mut exec = Executor::create(Some(&config), Some(&stats)).unwrap();

        extern_mod.add_func("newSyncTask", unsafe {
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(vec![ValType::I32, ValType::I32], vec![ValType::I64]),
                new_sync_task,
                &mut (HostData { runtime: self, exec: &mut exec }),
                1
            ).unwrap()
        });
        extern_mod.add_func("runDocker", unsafe {
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
                    vec![ValType::I64]
                ),
                run_docker,
                &mut (HostData { runtime: self, exec: &mut exec }),
                1
            ).unwrap()
        });
        extern_mod.add_func("execDocker", unsafe {
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
                    vec![ValType::I64]
                ),
                exec_docker,
                &mut (HostData { runtime: self, exec: &mut exec }),
                1
            ).unwrap()
        });
        extern_mod.add_func("copyToDocker", unsafe {
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
                    vec![ValType::I64]
                ),
                copy_to_docker,
                &mut (HostData { runtime: self, exec: &mut exec }),
                1
            ).unwrap()
        });
        extern_mod.add_func("consoleLog", unsafe {
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(vec![ValType::I32, ValType::I32], vec![ValType::I64]),
                console_log,
                &mut (HostData { runtime: self, exec: &mut exec }),
                1
            ).unwrap()
        });
        extern_mod.add_func("output", unsafe {
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(vec![ValType::I32, ValType::I32], vec![ValType::I64]),
                output,
                &mut (HostData { runtime: self, exec: &mut exec }),
                1
            ).unwrap()
        });
        extern_mod.add_func("httpPost", unsafe {
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
                    vec![ValType::I64]
                ),
                http_post,
                &mut (HostData { runtime: self, exec: &mut exec }),
                1
            ).unwrap()
        });
        extern_mod.add_func("plantTrigger", unsafe {
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
                    vec![ValType::I64]
                ),
                plant_trigger,
                &mut (HostData { runtime: self, exec: &mut exec }),
                1
            ).unwrap()
        });
        extern_mod.add_func("signalPoint", unsafe {
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
                    vec![ValType::I64]
                ),
                signal_point,
                &mut (HostData { runtime: self, exec: &mut exec }),
                1
            ).unwrap()
        });
        extern_mod.add_func("submitOnchainTrx", unsafe {
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
                    vec![ValType::I64]
                ),
                submit_onchain_trx,
                &mut (HostData { runtime: self, exec: &mut exec }),
                1
            ).unwrap()
        });
        extern_mod.add_func("put", unsafe {
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(
                    vec![ValType::I32, ValType::I32, ValType::I32, ValType::I32],
                    vec![ValType::I64]
                ),
                trx_put,
                &mut (HostData { runtime: self, exec: &mut exec }),
                1
            ).unwrap()
        });
        extern_mod.add_func("del", unsafe {
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(vec![ValType::I32, ValType::I32], vec![ValType::I64]),
                trx_del,
                &mut (HostData { runtime: self, exec: &mut exec }),
                1
            ).unwrap()
        });
        extern_mod.add_func("get", unsafe {
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(vec![ValType::I32, ValType::I32], vec![ValType::I64]),
                trx_get,
                &mut (HostData { runtime: self, exec: &mut exec }),
                1
            ).unwrap()
        });
        extern_mod.add_func("getByPrefix", unsafe {
            Function::create_sync_func(
                &wasmedge_sys::FuncType::new(vec![ValType::I32, ValType::I32], vec![ValType::I64]),
                trx_get_by_prefix,
                &mut (HostData { runtime: self, exec: &mut exec }),
                1
            ).unwrap()
        });

        exec.register_import_module(&mut store, &wasi_mod).unwrap();
        exec.register_import_module(&mut store, extern_mod).unwrap();

        let conf = Config::create().unwrap();
        let loader = Loader::create(Some(&conf)).unwrap();
        let main_mod_raw = loader.from_file(self.mod_path.clone()).unwrap();
        let conf2 = Config::create().unwrap();
        let v = Validator::create(Some(&conf2)).unwrap();
        v.validate(&main_mod_raw).unwrap();

        let vm_instance = exec.register_active_module(&mut store, &main_mod_raw).unwrap();

        self.vm_executor = Some(Rc::new(RefCell::new(exec)));
        self.vm_instance = Some(Rc::new(RefCell::new(vm_instance)));
        self.vm_stats = Some(Arc::new(Mutex::new(stats)));

        let instance_item = Rc::clone(&self.vm_instance.as_ref().unwrap());
        let mut instance_item_ref = instance_item.borrow_mut();

        let exec_item = Rc::clone(&self.vm_executor.as_ref().unwrap());
        let mut exec_item_ref = exec_item.borrow_mut();
        let mut binding = instance_item_ref.get_func_mut("_start").unwrap();

        exec_item_ref.call_func(&mut binding, []).unwrap();

        let val_l = input.len() as i32;
        let mut malloc_fn = instance_item_ref.get_func_mut("malloc").unwrap();
        let res2 = exec_item_ref.call_func(&mut malloc_fn, [WasmValue::from_i32(val_l)]).unwrap();

        let val_offset = res2[0].to_i32();
        let raw_arr = input.as_bytes();
        let arr: Vec<u8> = raw_arr.to_vec();
        let mem = instance_item_ref.get_memory_mut("memory");
        mem.unwrap().set_data(arr, val_offset.cast_unsigned()).unwrap();
        let c = ((val_offset as i64) << 32) | (val_l as i64);

        let mut run_fn = instance_item_ref.get_func_mut("run").unwrap();
        exec_item_ref.call_func(&mut run_fn, [WasmValue::from_i64(c)]).unwrap();
    }

    pub fn run_task(&mut self, task_id: String) {
        let val_l = task_id.len() as i32;
        let inst_ref1 = Rc::clone(&self.vm_instance.as_ref().unwrap());
        let mut inst_ref1_mut = inst_ref1.borrow_mut();
        let mut malloc_fn = inst_ref1_mut.get_func_mut("malloc").unwrap();
        let mfn = malloc_fn.deref_mut();
        let exec_ref1 = Rc::clone(&self.vm_executor.as_ref().unwrap());
        let res2 = exec_ref1
            .borrow_mut()
            .call_func(mfn, [WasmValue::from_i32(val_l)])
            .unwrap();

        let val_offset = res2[0].to_i32();
        let raw_arr = task_id.as_bytes();
        let arr: Vec<u8> = raw_arr.to_vec();
        let inst_ref2 = Rc::clone(&self.vm_instance.as_ref().unwrap());
        let mut inst_ref2_mut = inst_ref2.borrow_mut();
        let mem = inst_ref2_mut.get_memory_mut("memory");
        mem.unwrap().set_data(arr, val_offset.cast_unsigned()).unwrap();
        let c = ((val_offset as i64) << 32) | (val_l as i64);

        let inst_ref3 = Rc::clone(&self.vm_instance.as_ref().unwrap());
        let mut inst_ref3_mut = inst_ref3.borrow_mut();
        let mut run_fn = inst_ref3_mut.get_func_mut("runTask").unwrap();
        let rfn = run_fn.deref_mut();

        let exec_ref2 = Rc::clone(&self.vm_executor.as_ref().unwrap());
        exec_ref2
            .borrow_mut()
            .call_func(rfn, [WasmValue::from_i64(c)])
            .unwrap();
    }

    pub fn execute_on_chain(&mut self, input: String, user_id: String) {
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

        let val = (self.callback)(j);

        let jsn: JsonValue = serde_json::from_str(&val).unwrap();
        let gas_limit = jsn["gasLimit"].as_u64().unwrap_or(0);

        if gas_limit > 0 {
            self.is_token_valid = true;

            let mut store = Store::create().unwrap();

            let wasi_mod = wasmedge_sys::WasiModule::create(None, None, None).unwrap();

            let mut dummy: i32 = 1;
            let extern_mod = &mut ImportModule::create("env", Box::new(&mut dummy)).unwrap();

            let mut config = Config::create().unwrap();
            config.measure_cost(true);
            let mut stats = Statistics::create().unwrap();
            stats.set_cost_limit(gas_limit);
            let mut exec = Executor::create(Some(&config), Some(&stats)).unwrap();

            extern_mod.add_func("newSyncTask", unsafe {
                Function::create_sync_func(
                    &wasmedge_sys::FuncType::new(
                        vec![ValType::I32, ValType::I32],
                        vec![ValType::I64]
                    ),
                    new_sync_task,
                    &mut (HostData { runtime: self, exec: &mut exec }),
                    1
                ).unwrap()
            });
            extern_mod.add_func("runDocker", unsafe {
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
                        vec![ValType::I64]
                    ),
                    run_docker,
                    &mut (HostData { runtime: self, exec: &mut exec }),
                    1
                ).unwrap()
            });
            extern_mod.add_func("execDocker", unsafe {
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
                        vec![ValType::I64]
                    ),
                    exec_docker,
                    &mut (HostData { runtime: self, exec: &mut exec }),
                    1
                ).unwrap()
            });
            extern_mod.add_func("copyToDocker", unsafe {
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
                        vec![ValType::I64]
                    ),
                    copy_to_docker,
                    &mut (HostData { runtime: self, exec: &mut exec }),
                    1
                ).unwrap()
            });
            extern_mod.add_func("consoleLog", unsafe {
                Function::create_sync_func(
                    &wasmedge_sys::FuncType::new(
                        vec![ValType::I32, ValType::I32],
                        vec![ValType::I64]
                    ),
                    console_log,
                    &mut (HostData { runtime: self, exec: &mut exec }),
                    1
                ).unwrap()
            });
            extern_mod.add_func("output", unsafe {
                Function::create_sync_func(
                    &wasmedge_sys::FuncType::new(
                        vec![ValType::I32, ValType::I32],
                        vec![ValType::I64]
                    ),
                    output,
                    &mut (HostData { runtime: self, exec: &mut exec }),
                    1
                ).unwrap()
            });
            extern_mod.add_func("httpPost", unsafe {
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
                        vec![ValType::I64]
                    ),
                    http_post,
                    &mut (HostData { runtime: self, exec: &mut exec }),
                    1
                ).unwrap()
            });
            extern_mod.add_func("plantTrigger", unsafe {
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
                        vec![ValType::I64]
                    ),
                    plant_trigger,
                    &mut (HostData { runtime: self, exec: &mut exec }),
                    1
                ).unwrap()
            });
            extern_mod.add_func("signalPoint", unsafe {
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
                        vec![ValType::I64]
                    ),
                    signal_point,
                    &mut (HostData { runtime: self, exec: &mut exec }),
                    1
                ).unwrap()
            });
            extern_mod.add_func("submitOnchainTrx", unsafe {
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
                        vec![ValType::I64]
                    ),
                    submit_onchain_trx,
                    &mut (HostData { runtime: self, exec: &mut exec }),
                    1
                ).unwrap()
            });
            extern_mod.add_func("put", unsafe {
                Function::create_sync_func(
                    &wasmedge_sys::FuncType::new(
                        vec![ValType::I32, ValType::I32, ValType::I32, ValType::I32],
                        vec![ValType::I64]
                    ),
                    trx_put,
                    &mut (HostData { runtime: self, exec: &mut exec }),
                    1
                ).unwrap()
            });
            extern_mod.add_func("del", unsafe {
                Function::create_sync_func(
                    &wasmedge_sys::FuncType::new(
                        vec![ValType::I32, ValType::I32],
                        vec![ValType::I64]
                    ),
                    trx_del,
                    &mut (HostData { runtime: self, exec: &mut exec }),
                    1
                ).unwrap()
            });
            extern_mod.add_func("get", unsafe {
                Function::create_sync_func(
                    &wasmedge_sys::FuncType::new(
                        vec![ValType::I32, ValType::I32],
                        vec![ValType::I64]
                    ),
                    trx_get,
                    &mut (HostData { runtime: self, exec: &mut exec }),
                    1
                ).unwrap()
            });
            extern_mod.add_func("getByPrefix", unsafe {
                Function::create_sync_func(
                    &wasmedge_sys::FuncType::new(
                        vec![ValType::I32, ValType::I32],
                        vec![ValType::I64]
                    ),
                    trx_get_by_prefix,
                    &mut (HostData { runtime: self, exec: &mut exec }),
                    1
                ).unwrap()
            });

            exec.register_import_module(&mut store, &wasi_mod).unwrap();
            exec.register_import_module(&mut store, extern_mod).unwrap();

            let loader = Loader::create(Some(&config)).unwrap();
            let main_mod_raw = loader.from_file(self.mod_path.clone()).unwrap();
            let config2 = Config::create().unwrap();
            let v = Validator::create(Some(&config2)).unwrap();
            v.validate(&main_mod_raw).unwrap();

            let vm_instance = exec.register_active_module(&mut store, &main_mod_raw).unwrap();

            self.vm_executor = Some(Rc::new(RefCell::new(exec)));
            self.vm_instance = Some(Rc::new(RefCell::new(vm_instance)));
            self.vm_stats = Some(Arc::new(Mutex::new(stats)));

            let instance_item = Rc::clone(&self.vm_instance.as_ref().unwrap());
            let mut instance_item_ref = instance_item.borrow_mut();

            let exec_item = Rc::clone(&self.vm_executor.as_ref().unwrap());
            let mut exec_item_ref = exec_item.borrow_mut();
            let mut binding = instance_item_ref.get_func_mut("_start").unwrap();

            exec_item_ref.call_func(&mut binding, []).unwrap();

            let val_l = input.len() as i32;
            let mut malloc_fn = instance_item_ref.get_func_mut("malloc").unwrap();
            let res2 = exec_item_ref
                .call_func(&mut malloc_fn, [WasmValue::from_i32(val_l)])
                .unwrap();

            let val_offset = res2[0].to_i32();
            let raw_arr = params.as_bytes();
            let arr: Vec<u8> = raw_arr.to_vec();
            let mem = instance_item_ref.get_memory_mut("memory");
            mem.unwrap().set_data(arr, val_offset.cast_unsigned()).unwrap();
            let c = ((val_offset as i64) << 32) | (val_l as i64);

            let mut run_fn = instance_item_ref.get_func_mut("run").unwrap();
            exec_item_ref.call_func(&mut run_fn, [WasmValue::from_i64(c)]).unwrap();
        }

        if self.onchain {
            let cr_cloned = Arc::clone(unsafe { &GLOBAL_CR });
            let cr = cr_cloned.lock().unwrap();
            let _lock = cr.wasm_global_lock.lock().unwrap();
            let cr_cloned2 = Arc::clone(unsafe { &GLOBAL_CR });
            let mut cr2 = cr_cloned2.lock().unwrap();
            cr2.wasm_done_tasks += 1;
            if cr2.wasm_done_tasks == cr2.wasm_count {
                unsafe {
                    if STEP == 0 {
                        cr2.wasm_done_tasks = 0;
                        STEP += 1;
                        cr2.wasm_do_critical();
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

pub fn new_sync_task(
    host_data: &mut HostData,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem = _caller.memory_ref(0).unwrap();

    let key_offset = _input[0].to_i32();
    let key_l = _input[1].to_i32();
    let text_bytes_vec = mem.get_data(key_offset.cast_unsigned(), key_l.cast_unsigned()).unwrap();
    let text_bytes = text_bytes_vec.as_slice();
    let text = str::from_utf8(&text_bytes).unwrap();

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

    let rt: &mut WasmMac = unsafe { &mut *host_data.runtime };
    rt.sync_tasks.push(SyncTask { deps, name });

    Ok(vec![WasmValue::from_i64(0)])
}

pub fn output(
    host_data: &mut HostData,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem = _caller.memory_ref(0).unwrap();
    let rt: &mut WasmMac = unsafe { &mut *host_data.runtime };

    let key_offset = _input[0].to_i32();
    let key_l = _input[1].to_i32();
    let text_bytes = mem.get_data(key_offset.cast_unsigned(), key_l.cast_unsigned());
    let text_bytes_next = text_bytes.unwrap();
    let text = str::from_utf8(&text_bytes_next).unwrap();

    rt.execution_result = text.to_string();
    rt.has_output = true;

    Ok(vec![WasmValue::from_i64(0)])
}

pub fn console_log(
    _: &mut HostData,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem = _caller.memory_ref(0).unwrap();

    let key_offset = _input[0].to_i32();
    let key_l = _input[1].to_i32();
    let text_bytes = mem.get_data(key_offset.cast_unsigned(), key_l.cast_unsigned());
    let text_bytes_next = text_bytes.unwrap();
    let text = str::from_utf8(&text_bytes_next).unwrap();

    println!("{text}");

    Ok(vec![WasmValue::from_i64(0)])
}

pub fn submit_onchain_trx(
    host_data: &mut HostData,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem = _caller.memory_ref(0).unwrap();
    let rt: &mut WasmMac = unsafe { &mut *host_data.runtime };
    let exec: &mut Executor = unsafe { &mut *host_data.exec };

    let tm_offset = _input[0].to_i32();
    let tm_l = _input[1].to_i32();

    let key_offset = _input[2].to_i32();
    let key_l = _input[3].to_i32();
    let key_bytes = mem.get_data(key_offset.cast_unsigned(), key_l.cast_unsigned());
    let key_bytes_next = key_bytes.unwrap();
    let key = str::from_utf8(&key_bytes_next).unwrap();

    let input_offset = _input[4].to_i32();
    let input_l = _input[5].to_i32();
    let input_bytes = mem.get_data(input_offset.cast_unsigned(), input_l.cast_unsigned());
    let input_bytes_next = input_bytes.unwrap();
    let input = str::from_utf8(&input_bytes_next).unwrap();

    let meta_offset = _input[6].to_i32();
    let meta_l = _input[7].to_i32();
    let meta_bytes = mem.get_data(meta_offset.cast_unsigned(), meta_l.cast_unsigned());
    let meta_bytes_next = meta_bytes.unwrap();
    let meta = str::from_utf8(&meta_bytes_next).unwrap();

    let is_base = meta.chars().nth(0).unwrap_or('0') == '1';
    let is_file = meta.chars().nth(1).unwrap_or('0') == '1';

    let tag = meta.chars().skip(2).collect::<String>();

    let target_machine_id = if !is_base {
        str::from_utf8(&mem.get_data(tm_offset.cast_unsigned(), tm_l.cast_unsigned()).unwrap())
            .unwrap()
            .to_string()
    } else {
        "".to_string()
    };

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

    let val = wasm_send(j);
    let val_l = val.len();

    let mut malloc_fn = _inst.get_func_mut("malloc").unwrap();
    let mfn = malloc_fn.deref_mut();
    let res = exec.call_func(mfn, [WasmValue::from_i32(val_l as i32)]).unwrap();
    let val_offset = res[0].to_i32();

    let arr = val.as_bytes().to_vec();

    let mut mem2 = _inst.get_memory_mut("memory").unwrap();

    mem2.set_data(arr, val_offset.cast_unsigned()).unwrap();
    let c = ((val_offset as i64) << 32) | (val_l as i64);

    Ok(vec![WasmValue::from_i64(c)])
}

pub fn plant_trigger(
    host_data: &mut HostData,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem = _caller.memory_mut(0).unwrap();
    let rt: &mut WasmMac = unsafe { &mut *host_data.runtime };

    let in_offset = _input[0].to_i32();
    let in_l = _input[1].to_i32();
    let in_bytes = mem.get_data(in_offset.cast_unsigned(), in_l.cast_unsigned());
    let in_bytes_next = in_bytes.unwrap();
    let tag = str::from_utf8(&in_bytes_next).unwrap();

    let key_offset = _input[2].to_i32();
    let key_l = _input[3].to_i32();
    let key_bytes = mem.get_data(key_offset.cast_unsigned(), key_l.cast_unsigned());
    let key_bytes_next = key_bytes.unwrap();
    let text = str::from_utf8(&key_bytes_next).unwrap();

    let pi_offset = _input[4].to_i32();
    let pi_l = _input[5].to_i32();
    let pi_bytes = mem.get_data(pi_offset.cast_unsigned(), pi_l.cast_unsigned());
    let pi_bytes_next = pi_bytes.unwrap();
    let point_id = str::from_utf8(&pi_bytes_next).unwrap();

    let count = _input[6].to_i32();

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

    (rt.callback)(j);

    Ok(vec![WasmValue::from_i64(0)])
}

pub fn http_post(
    host_data: &mut HostData,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem = _caller.memory_mut(0).unwrap();
    let rt: &mut WasmMac = unsafe { &mut *host_data.runtime };
    let exec: &mut Executor = unsafe { &mut *host_data.exec };

    let url_offset = _input[0].to_i32();
    let url_l = _input[1].to_i32();
    let url_bytes = mem.get_data(url_offset.cast_unsigned(), url_l.cast_unsigned());
    let url_bytes_next = url_bytes.unwrap();
    let url = str::from_utf8(&url_bytes_next).unwrap();

    let heads_offset = _input[2].to_i32();
    let heads_l = _input[3].to_i32();
    let heads_bytes = mem.get_data(heads_offset.cast_unsigned(), heads_l.cast_unsigned());
    let heads_bytes_next = heads_bytes.unwrap();
    let headers = str::from_utf8(&heads_bytes_next).unwrap();

    let body_offset = _input[4].to_i32();
    let body_l = _input[5].to_i32();
    let body_bytes = mem.get_data(body_offset.cast_unsigned(), body_l.cast_unsigned());
    let body_bytes_next = body_bytes.unwrap();
    let body = str::from_utf8(&body_bytes_next).unwrap();

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

    let val = (rt.callback)(j);
    let val_l = val.len();

    let mut malloc_fn = _inst.get_func_mut("malloc").unwrap();
    let mfn = malloc_fn.deref_mut();
    let res = exec.call_func(mfn, [WasmValue::from_i32(val_l as i32)]).unwrap();
    let val_offset = res[0].to_i32();

    let arr = val.as_bytes().to_vec();

    let mut mem2 = _inst.get_memory_mut("memory").unwrap();

    mem2.set_data(arr, val_offset.cast_unsigned()).unwrap();
    let c = ((val_offset as i64) << 32) | (val_l as i64);

    Ok(vec![WasmValue::from_i64(c)])
}

pub fn run_docker(
    host_data: &mut HostData,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let rt: &mut WasmMac = unsafe { &mut *host_data.runtime };
    let exec: &mut Executor = unsafe { &mut *host_data.exec };

    let mem = _inst.get_memory_mut("memory").unwrap();

    let in_offset = _input[0].to_i32();
    let in_l = _input[1].to_i32();
    let in_bytes = mem.get_data(in_offset.cast_unsigned(), in_l.cast_unsigned());
    let in_bytes_next = in_bytes.unwrap();
    let image_name = str::from_utf8(&in_bytes_next).unwrap();

    let key_offset = _input[2].to_i32();
    let key_l = _input[3].to_i32();
    let key_bytes = mem.get_data(key_offset.cast_unsigned(), key_l.cast_unsigned());
    let key_bytes_next = key_bytes.unwrap();
    let text = str::from_utf8(&key_bytes_next).unwrap();

    let cn_offset = _input[4].to_i32();
    let cn_l = _input[5].to_i32();
    let cn_bytes = mem.get_data(cn_offset.cast_unsigned(), cn_l.cast_unsigned());
    let cn_bytes_next = cn_bytes.unwrap();
    let container_name = str::from_utf8(&cn_bytes_next).unwrap();

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

    let val = (rt.callback)(j);

    let val_l = val.len();

    let mut malloc_fn = _inst.get_func_mut("malloc").unwrap();
    let mfn = malloc_fn.deref_mut();
    let res = exec.call_func(mfn, [WasmValue::from_i32(val_l as i32)]).unwrap();
    let val_offset = res[0].to_i32();

    let arr = val.as_bytes().to_vec();

    let mut mem2 = _inst.get_memory_mut("memory").unwrap();

    mem2.set_data(arr, val_offset.cast_unsigned()).unwrap();
    let c = ((val_offset as i64) << 32) | (val_l as i64);

    Ok(vec![WasmValue::from_i64(c)])
}

pub fn exec_docker(
    host_data: &mut HostData,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem = _caller.memory_mut(0).unwrap();
    let rt: &mut WasmMac = unsafe { &mut *host_data.runtime };
    let exec: &mut Executor = unsafe { &mut *host_data.exec };

    let in_offset = _input[0].to_i32();
    let in_l = _input[1].to_i32();
    let in_bytes = mem.get_data(in_offset.cast_unsigned(), in_l.cast_unsigned());
    let in_bytes_next = in_bytes.unwrap();
    let image_name = str::from_utf8(&in_bytes_next).unwrap();

    let key_offset = _input[2].to_i32();
    let key_l = _input[3].to_i32();
    let key_bytes = mem.get_data(key_offset.cast_unsigned(), key_l.cast_unsigned());
    let key_bytes_next = key_bytes.unwrap();
    let container_name = str::from_utf8(&key_bytes_next).unwrap();

    let cn_offset = _input[4].to_i32();
    let cn_l = _input[5].to_i32();
    let cn_bytes = mem.get_data(cn_offset.cast_unsigned(), cn_l.cast_unsigned());
    let cn_bytes_next = cn_bytes.unwrap();
    let command = str::from_utf8(&cn_bytes_next).unwrap();

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

    let res = (rt.callback)(j);

    let jres = json!({
            "data": res
        });

    let val = jres.to_string();
    let val_l = val.len();

    let mut malloc_fn = _inst.get_func_mut("malloc").unwrap();
    let mfn = malloc_fn.deref_mut();
    let res = exec.call_func(mfn, [WasmValue::from_i32(val_l as i32)]).unwrap();
    let val_offset = res[0].to_i32();

    let arr = val.as_bytes().to_vec();

    let mut mem2 = _inst.get_memory_mut("memory").unwrap();

    mem2.set_data(arr, val_offset.cast_unsigned()).unwrap();
    let c = ((val_offset as i64) << 32) | (val_l as i64);

    Ok(vec![WasmValue::from_i64(c)])
}

pub fn copy_to_docker(
    host_data: &mut HostData,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem = _caller.memory_mut(0).unwrap();
    let rt: &mut WasmMac = unsafe { &mut *host_data.runtime };
    let exec: &mut Executor = unsafe { &mut *host_data.exec };

    let in_offset = _input[0].to_i32();
    let in_l = _input[1].to_i32();
    let in_bytes = mem.get_data(in_offset.cast_unsigned(), in_l.cast_unsigned());
    let in_bytes_next = in_bytes.unwrap();
    let image_name = str::from_utf8(&in_bytes_next).unwrap();

    let key_offset = _input[2].to_i32();
    let key_l = _input[3].to_i32();
    let key_bytes = mem.get_data(key_offset.cast_unsigned(), key_l.cast_unsigned());
    let key_bytes_next = key_bytes.unwrap();
    let container_name = str::from_utf8(&key_bytes_next).unwrap();

    let co_offset = _input[4].to_i32();
    let co_l = _input[5].to_i32();
    let co_bytes = mem.get_data(co_offset.cast_unsigned(), co_l.cast_unsigned());
    let co_bytes_next = co_bytes.unwrap();
    let file_name = str::from_utf8(&co_bytes_next).unwrap();

    let cn_offset = _input[6].to_i32();
    let cn_l = _input[7].to_i32();
    let cn_bytes = mem.get_data(cn_offset.cast_unsigned(), cn_l.cast_unsigned());
    let cn_bytes_next = cn_bytes.unwrap();
    let content = str::from_utf8(&cn_bytes_next).unwrap();

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

    let res = (rt.callback)(j);

    let jres = json!({
            "data": res
        });

    let val = jres.to_string();
    let val_l = val.len();

    let mut malloc_fn = _inst.get_func_mut("malloc").unwrap();
    let mfn = malloc_fn.deref_mut();
    let res = exec.call_func(mfn, [WasmValue::from_i32(val_l as i32)]).unwrap();
    let val_offset = res[0].to_i32();

    let arr = val.as_bytes().to_vec();

    let mut mem2 = _inst.get_memory_mut("memory").unwrap();

    mem2.set_data(arr, val_offset.cast_unsigned()).unwrap();
    let c = ((val_offset as i64) << 32) | (val_l as i64);

    Ok(vec![WasmValue::from_i64(c)])
}

pub fn signal_point(
    host_data: &mut HostData,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem = _caller.memory_mut(0).unwrap();
    let rt: &mut WasmMac = unsafe { &mut *host_data.runtime };
    let exec: &mut Executor = unsafe { &mut *host_data.exec };

    let in_offset = _input[0].to_i32();
    let in_l = _input[1].to_i32();
    let in_bytes = mem.get_data(in_offset.cast_unsigned(), in_l.cast_unsigned());
    let in_bytes_next = in_bytes.unwrap();
    let typ = str::from_utf8(&in_bytes_next).unwrap();

    let key_offset = _input[2].to_i32();
    let key_l = _input[3].to_i32();
    let key_bytes = mem.get_data(key_offset.cast_unsigned(), key_l.cast_unsigned());
    let key_bytes_next = key_bytes.unwrap();
    let point_id = str::from_utf8(&key_bytes_next).unwrap();

    let co_offset = _input[4].to_i32();
    let co_l = _input[5].to_i32();
    let co_bytes = mem.get_data(co_offset.cast_unsigned(), co_l.cast_unsigned());
    let co_bytes_next = co_bytes.unwrap();
    let user_id = str::from_utf8(&co_bytes_next).unwrap();

    let cn_offset = _input[6].to_i32();
    let cn_l = _input[7].to_i32();
    let cn_bytes = mem.get_data(cn_offset.cast_unsigned(), cn_l.cast_unsigned());
    let cn_bytes_next = cn_bytes.unwrap();
    let payload = str::from_utf8(&cn_bytes_next).unwrap();

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

    let val = (rt.callback)(j);
    let val_l = val.len();

    let mut malloc_fn = _inst.get_func_mut("malloc").unwrap();
    let mfn = malloc_fn.deref_mut();
    let res = exec.call_func(mfn, [WasmValue::from_i32(val_l as i32)]).unwrap();
    let val_offset = res[0].to_i32();

    let arr = val.as_bytes().to_vec();

    let mut mem2 = _inst.get_memory_mut("memory").unwrap();

    mem2.set_data(arr, val_offset.cast_unsigned()).unwrap();
    let c = ((val_offset as i64) << 32) | (val_l as i64);

    Ok(vec![WasmValue::from_i64(c)])
}

use std::sync::{ atomic::{ AtomicI32 } };
use std::time::Instant;

pub fn trx_put(
    host_data: &mut HostData,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem = _caller.memory_mut(0).unwrap();
    let rt: &mut WasmMac = unsafe { &mut *host_data.runtime };

    let in_offset = _input[0].to_i32();
    let in_l = _input[1].to_i32();
    let in_bytes = mem.get_data(in_offset.cast_unsigned(), in_l.cast_unsigned());
    let in_bytes_next = in_bytes.unwrap();
    let key = str::from_utf8(&in_bytes_next).unwrap();

    let val_offset = _input[2].to_i32();
    let val_l = _input[3].to_i32();
    let val_bytes = mem.get_data(val_offset.cast_unsigned(), val_l.cast_unsigned());
    let val_bytes_next = val_bytes.unwrap();
    let val = str::from_utf8(&val_bytes_next).unwrap();

    rt.trx.put(format!("{}::{}", rt.machine_id, key), val.to_string());

    Ok(vec![WasmValue::from_i64(0)])
}

pub fn trx_del(
    host_data: &mut HostData,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem = _caller.memory_mut(0).unwrap();
    let rt: &mut WasmMac = unsafe { &mut *host_data.runtime };

    let in_offset = _input[0].to_i32();
    let in_l = _input[1].to_i32();
    let in_bytes = mem.get_data(in_offset.cast_unsigned(), in_l.cast_unsigned());
    let in_bytes_next = in_bytes.unwrap();
    let key = str::from_utf8(&in_bytes_next).unwrap();

    rt.trx.del(format!("{}::{}", rt.machine_id, key));

    Ok(vec![WasmValue::from_i64(0)])
}

pub fn trx_get(
    host_data: &mut HostData,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem = _caller.memory_mut(0).unwrap();
    let rt: &mut WasmMac = unsafe { &mut *host_data.runtime };
    let exec: &mut Executor = unsafe { &mut *host_data.exec };

    let in_offset = _input[0].to_i32();
    let in_l = _input[1].to_i32();
    let in_bytes = mem.get_data(in_offset.cast_unsigned(), in_l.cast_unsigned());
    let in_bytes_next = in_bytes.unwrap();
    let key = str::from_utf8(&in_bytes_next).unwrap();

    let val = rt.trx.get(format!("{}::{}", rt.machine_id, key));
    let val_l = val.len();

    let mut malloc_fn = _inst.get_func_mut("malloc").unwrap();
    let mfn = malloc_fn.deref_mut();
    let res = exec.call_func(mfn, [WasmValue::from_i32(val_l as i32)]).unwrap();
    let val_offset = res[0].to_i32();

    let arr = val.as_bytes().to_vec();

    let mut mem2 = _inst.get_memory_mut("memory").unwrap();

    mem2.set_data(arr, val_offset.cast_unsigned()).unwrap();
    let c = ((val_offset as i64) << 32) | (val_l as i64);

    Ok(vec![WasmValue::from_i64(c)])
}

pub fn trx_get_by_prefix(
    host_data: &mut HostData,
    _inst: &mut Instance,
    _caller: &mut CallingFrame,
    _input: Vec<WasmValue>
) -> Result<Vec<WasmValue>, CoreError> {
    let mem = _caller.memory_mut(0).unwrap();
    let rt: &mut WasmMac = unsafe { &mut *host_data.runtime };
    let exec: &mut Executor = unsafe { &mut *host_data.exec };

    let in_offset = _input[0].to_i32();
    let in_l = _input[1].to_i32();
    let in_bytes = mem.get_data(in_offset.cast_unsigned(), in_l.cast_unsigned());
    let in_bytes_next = in_bytes.unwrap();
    let prefix = str::from_utf8(&in_bytes_next).unwrap();

    let vals = rt.trx.get_by_prefix(format!("{}::{}", rt.machine_id, prefix));
    let j = json!({
        "data": vals
    });

    let val = j.to_string();
    let val_l = vals.len();

    let mut malloc_fn = _inst.get_func_mut("malloc").unwrap();
    let mfn = malloc_fn.deref_mut();
    let res = exec.call_func(mfn, [WasmValue::from_i32(val_l as i32)]).unwrap();
    let val_offset = res[0].to_i32();

    let arr = val.as_bytes().to_vec();

    let mut mem2 = _inst.get_memory_mut("memory").unwrap();

    mem2.set_data(arr, val_offset.cast_unsigned()).unwrap();
    let c = ((val_offset as i64) << 32) | (val_l as i64);

    Ok(vec![WasmValue::from_i64(c)])
}

pub struct ConcurrentRunner {
    pub wasm_global_lock: Mutex<()>,
    pub wasm_done_tasks: i32,
    pub wasm_count: i32,
    pub trxs: Vec<ChainTrx>,
    pub ast_store_path: String,
    pub wasm_vms: Vec<Option<Arc<Mutex<WasmMac>>>>,
    pub wasm_vm_map: HashMap<String, (usize, Arc<Mutex<WasmMac>>)>,
    pub exec_wasm_locks: Vec<Arc<Mutex<()>>>,
    pub main_wasm_lock: Arc<Mutex<()>>,
    pub thread_pool: Arc<Mutex<WasmThreadPool>>,
    pub saved_key_counter: i32,
}

static mut GLOBAL_CR: Lazy<Arc<Mutex<ConcurrentRunner>>> = Lazy::new(|| {
    Arc::new(
        Mutex::new(ConcurrentRunner {
            wasm_global_lock: Mutex::new(()),
            wasm_done_tasks: 0,
            wasm_count: 0,
            trxs: vec![],
            ast_store_path: "".to_string(),
            wasm_vms: vec![],
            wasm_vm_map: HashMap::new(),
            exec_wasm_locks: vec![],
            main_wasm_lock: Arc::new(Mutex::new(())),
            thread_pool: Arc::new(Mutex::new(WasmThreadPool::generate(Some(8)))),
            saved_key_counter: 0,
        })
    )
});

impl ConcurrentRunner {
    pub fn init(ast_store_path: String, trxs: Vec<ChainTrx>) {
        unsafe {
            let gcr_raw = Arc::clone(&GLOBAL_CR);
            let mut gcr = gcr_raw.lock().unwrap();
            gcr.trxs = trxs;
            gcr.ast_store_path = ast_store_path;
            gcr.wasm_global_lock = Mutex::new(());
            gcr.wasm_done_tasks = 0;
            gcr.wasm_count = 0;
            gcr.wasm_vms = vec![];
            gcr.wasm_vm_map = HashMap::new();
            gcr.exec_wasm_locks = vec![];
            gcr.main_wasm_lock = Arc::new(Mutex::new(()));
            gcr.thread_pool = Arc::new(Mutex::new(WasmThreadPool::generate(Some(8))));
            gcr.saved_key_counter = 0;
        }
    }

    pub fn run(&mut self) {
        let start_time = Instant::now();
        let mut wasm_thread_pool = WasmThreadPool::generate(Some(8));
        self.prepare_context(self.trxs.len());

        let mut handles = Vec::new();

        for i in 0..self.trxs.len() {
            let trx = self.trxs[i].clone();
            let ast_store_path = self.ast_store_path.clone();
            let handle = thread::spawn(move || {
                let mut rt = WasmMac::new_onchain(
                    trx.machine_id.clone(),
                    i.to_string(),
                    i as i32,
                    format!("{}/{}/module", ast_store_path, trx.machine_id),
                    Box::new(wasm_send)
                );
                rt.execute_on_chain(trx.input, trx.user_id);
            });
            handles.push(handle);
        }

        for handle in handles {
            handle.join().unwrap();
        }

        wasm_thread_pool.stop_pool();
        wasm_thread_pool.stick();

        for rt_opt in &self.wasm_vms {
            if let Some(rt_arc) = rt_opt {
                let mut rt = rt_arc.lock().unwrap();
                let stats = rt.vm_stats.clone().unwrap();
                let cloned_stats = Arc::clone(&stats);
                let cost = cloned_stats.lock().unwrap().cost_in_total();

                let output = rt.execution_result.clone();
                let changes = rt.finalize();

                let mut arr = Vec::new();
                for op in changes {
                    let item =
                        json!({
                        "opType": op.type_,
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
                        "callbackId": self.trxs.get(rt.index as usize).unwrap().callback_id,
                        "packet": output,
                        "resCode": 1,
                        "error": "",
                        "changes": ops_str,
                        "cost": cost,
                        "tokenId": rt.chain_token_id
                    }
                });

                wasm_send(j);
            }
        }

        let elapsed = start_time.elapsed();
        log(format!("executed chain applet transactions in {} microseconds.", elapsed.as_micros()));
    }

    pub fn prepare_context(&mut self, vm_count: usize) {
        self.wasm_count = vm_count as i32;
        self.wasm_vms = vec![None; vm_count];
        self.exec_wasm_locks = (0..vm_count).map(|_| Arc::new(Mutex::new(()))).collect();
    }

    pub fn register_wasm_mac(&mut self, rt: Arc<Mutex<WasmMac>>) {
        let rt_guard = rt.lock().unwrap();
        let id = rt_guard.id.clone();
        let index = rt_guard.index;
        drop(rt_guard);

        let cloned_rt1 = Arc::clone(&rt);
        let cloned_rt2 = Arc::clone(&rt);

        self.wasm_vm_map.insert(id, (index as usize, cloned_rt1));
        self.wasm_vms[index as usize] = Some(cloned_rt2);
    }

    pub fn wasm_run_task<F>(&self, task: F, index: usize) where F: Fn(&mut WasmMac) {
        if let Some(rt_arc) = &self.wasm_vms[index] {
            let mut rt = rt_arc.lock().unwrap();
            task(&mut rt);
        }
    }

    pub fn wasm_do_critical(&mut self) {
        let mut key_counter = 1;
        let mut all_wasm_tasks: HashMap<i32, (Vec<String>, usize, String)> = HashMap::new();
        let mut task_refs: HashMap<String, Arc<Mutex<WasmTask>>> = HashMap::new();
        let mut res_wasm_locks: HashMap<
            String,
            (Option<Arc<Mutex<WasmTask>>>, Arc<WasmLock>)
        > = HashMap::new();
        let mut start_points: HashMap<String, Arc<Mutex<WasmTask>>> = HashMap::new();
        let mut res_counter = 0;
        let mut c = 0;

        for index in 0..self.wasm_count {
            if let Some(vm_arc) = &self.wasm_vms[index as usize] {
                let vm = vm_arc.lock().unwrap();
                for t in &vm.sync_tasks {
                    let res_nums = t.deps.clone();
                    let name = t.name.clone();

                    for r in &res_nums {
                        if !res_wasm_locks.contains_key(r) {
                            res_wasm_locks.insert(r.clone(), (
                                None,
                                Arc::new(WasmLock::generate()),
                            ));
                            res_counter += 1;
                        }
                    }
                    all_wasm_tasks.insert(key_counter, (res_nums, index as usize, name));
                    key_counter += 1;
                    c += 1;
                }
            }
        }

        for i in 1..=key_counter {
            if let Some((res_nums, vm_index, name)) = all_wasm_tasks.get(&i) {
                let inputs = Arc::new(Mutex::new(HashMap::new()));
                let outputs = Arc::new(Mutex::new(HashMap::new()));
                let task = Arc::new(
                    Mutex::new(WasmTask {
                        id: i,
                        name: name.clone(),
                        inputs: inputs,
                        outputs: outputs,
                        vm_index: *vm_index as i32,
                        started: false,
                    })
                );

                let vm_index_str = vm_index.to_string();
                let cloned_task = Arc::clone(&task);
                task_refs.insert(format!("{}:{}", vm_index_str, name), cloned_task);

                for r in res_nums {
                    if r.starts_with("lock_") {
                        if let Some(t) = task_refs.get(&format!("{}:{}", vm_index_str, r)) {
                            let t_cloned = Arc::clone(&t);
                            let t_cloned_ref = t_cloned.lock().unwrap();
                            let inputs_cloned = Arc::clone(&t_cloned_ref.inputs);
                            let mut inputs_cloned_ref = inputs_cloned.lock().unwrap();
                            let t_cloned2 = Arc::clone(&t);
                            inputs_cloned_ref.insert(t_cloned_ref.id, (false, t_cloned2));
                            let outputs_cloned = Arc::clone(&t_cloned_ref.outputs);
                            let mut outputs_cloned_ref = outputs_cloned.lock().unwrap();
                            let t_cloned3 = Arc::clone(&t);
                            outputs_cloned_ref.insert(t_cloned_ref.id.clone(), t_cloned3);
                        }
                    } else {
                        if let Some((first_task, _)) = res_wasm_locks.get_mut(r) {
                            if first_task.is_none() {
                                let cloned_task = Arc::clone(&task);
                                *first_task = Some(cloned_task);
                                let cloned_task2 = Arc::clone(&task);
                                start_points.insert(r.clone(), cloned_task2);
                            } else {
                                let cloned_task = Arc::clone(&task);
                                *first_task = Some(cloned_task);
                                let cloned_task2 = Arc::clone(&task);
                                let cloned_task_ref2 = cloned_task2.lock().unwrap();
                                let inputs_cloned = Arc::clone(&cloned_task_ref2.inputs);
                                let mut inputs_cloned_ref = inputs_cloned.lock().unwrap();
                                let item = first_task.as_ref().unwrap();
                                let t_cloned = Arc::clone(&item);
                                let t_cloned_ref = t_cloned.lock().unwrap();
                                let t_cloned_t = Arc::clone(&item);
                                inputs_cloned_ref.insert(t_cloned_ref.id, (false, t_cloned_t));
                                let outputs_cloned = Arc::clone(&t_cloned_ref.outputs);
                                let mut outputs_cloned_ref = outputs_cloned.lock().unwrap();
                                let cloned_task4 = Arc::clone(&task);
                                outputs_cloned_ref.insert(
                                    cloned_task_ref2.id.clone(),
                                    cloned_task4
                                );
                                let cloned_task5 = Arc::clone(&task);
                                let l = &res_wasm_locks.get(r).unwrap().1;
                                let l_cloned = Arc::clone(l);
                                res_wasm_locks.insert(r.clone().to_string(), (
                                    Some(cloned_task5),
                                    l_cloned,
                                ));
                            }
                        }
                    }
                }
            }
        }

        {
            let cloned_cr0 = Arc::clone(unsafe { &GLOBAL_CR });
            let mut cloned_cr_ref0 = cloned_cr0.lock().unwrap();
            cloned_cr_ref0.thread_pool = Arc::new(
                Mutex::new(WasmThreadPool::generate(Some(res_counter)))
            );
            cloned_cr_ref0.saved_key_counter = key_counter;
        }

        {
            for (_r, task) in start_points {
                let cloned_cr5 = Arc::clone(unsafe { &GLOBAL_CR });
                let mut cloned_cr_ref5 = cloned_cr5.lock().unwrap();
                cloned_cr_ref5.exec_wasm_task(task);
            }
        }

        let cloned_cr_t3 = Arc::clone(unsafe { &GLOBAL_CR });
        let cloned_cr_ref_t3 = cloned_cr_t3.lock().unwrap();
        cloned_cr_ref_t3.thread_pool.lock().unwrap().stick();
    }

    pub fn exec_wasm_task(&mut self, task: Arc<Mutex<WasmTask>>) {
        unsafe {
            let wasm_done_wasm_tasks_count = Arc::new(AtomicI32::new(1));
            let mut ready_to_exec = false;
            {
                let _gcr = GLOBAL_CR.lock().unwrap();
                let _mg = _gcr.wasm_global_lock.lock().unwrap();
                let cloned_task_t = Arc::clone(&task);
                let mut cloned_task_ref_t = cloned_task_t.lock().unwrap();
                if cloned_task_ref_t.started == false {
                    let mut passed = true;
                    for (_, val) in cloned_task_ref_t.inputs.lock().unwrap().iter() {
                        if !val.0 {
                            passed = false;
                            break;
                        }
                    }
                    if passed {
                        cloned_task_ref_t.started = true;
                        ready_to_exec = true;
                    }
                }
            }
            if ready_to_exec {
                let cloned_cr_t = Arc::clone(&GLOBAL_CR);
                let cloned_cr_ref_t = cloned_cr_t.lock().unwrap();
                cloned_cr_ref_t.thread_pool
                    .lock()
                    .unwrap()
                    .enqueue(move || {
                        {
                            let cloned_cr1 = Arc::clone(&GLOBAL_CR);
                            let cloned_cr_ref1 = cloned_cr1.lock().unwrap();
                            let cloned_task_t = Arc::clone(&task);
                            let cloned_task_ref_t = cloned_task_t.lock().unwrap();
                            let _mg = cloned_cr_ref1.exec_wasm_locks
                                .get(cloned_task_ref_t.vm_index as usize)
                                .unwrap()
                                .lock();

                            let cloned_task2 = Arc::clone(&task);
                            cloned_cr_ref1.wasm_run_task(move |vm: &mut WasmMac| {
                                let cloned_task_ref2 = cloned_task2.lock().unwrap();
                                vm.run_task(cloned_task_ref2.name.clone());
                            }, cloned_task_ref_t.vm_index.clone() as usize);
                        }
                        let mut next_wasm_tasks: Vec<Arc<Mutex<WasmTask>>> = Vec::new();
                        {
                            let cloned_cr3 = Arc::clone(&GLOBAL_CR);
                            let cloned_cr_ref3 = cloned_cr3.lock().unwrap();
                            let _mg = cloned_cr_ref3.wasm_global_lock.lock();
                            wasm_done_wasm_tasks_count.fetch_add(1, Ordering::Acquire);
                            if
                                wasm_done_wasm_tasks_count.load(Ordering::Acquire) ==
                                cloned_cr_ref3.saved_key_counter
                            {
                                cloned_cr_ref3.thread_pool.lock().unwrap().stop_pool();
                            }
                            let mut cloned_outputs: Option<
                                HashMap<i32, Arc<Mutex<WasmTask>>>
                            > = None;
                            {
                                let cloned_task1 = Arc::clone(&task);
                                let cloned_task_ref1 = cloned_task1.lock().unwrap();
                                let cloned_outputs_1 = Arc::clone(&cloned_task_ref1.outputs);
                                let cloned_outputs_ref1 = cloned_outputs_1.lock().unwrap();
                                cloned_outputs = Some(cloned_outputs_ref1.clone());
                            }
                            for (_, val) in cloned_outputs.unwrap().iter() {
                                let cloned_task_t2 = Arc::clone(&val);
                                let cloned_task_ref_t2 = cloned_task_t2.lock().unwrap();
                                if !cloned_task_ref_t2.started {
                                    let cloned_task1 = Arc::clone(&task);
                                    let cloned_task_ref1 = cloned_task1.lock().unwrap();
                                    cloned_task_ref_t2.inputs
                                        .lock()
                                        .unwrap()
                                        .get_mut(&cloned_task_ref1.id.clone())
                                        .unwrap().0 = true;
                                    let cloned_task_t3 = Arc::clone(&val);
                                    next_wasm_tasks.push(cloned_task_t3);
                                }
                            }
                        }
                        for t in next_wasm_tasks {
                            let cloned_t = Arc::clone(&t);
                            let cloned_cr4 = Arc::clone(&GLOBAL_CR);
                            let mut cloned_cr_ref4 = cloned_cr4.lock().unwrap();
                            cloned_cr_ref4.exec_wasm_task(cloned_t);
                        }
                    });
            }
        }
    }
}

impl Clone for ChainTrx {
    fn clone(&self) -> Self {
        ChainTrx {
            machine_id: self.machine_id.clone(),
            callback_id: self.callback_id.clone(),
            input: self.input.clone(),
            user_id: self.user_id.clone(),
            key: self.key.clone(),
        }
    }
}
