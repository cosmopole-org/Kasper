package sdk

//go:module env
//export httpPost
func HttpPost(k int32, kl int32, v int32, lv int32, p int32, pv int32) int64

//go:module env
//export plantTrigger
func PlantTrigger(k int32, kl int32, v int32, lv int32, p int32, pv int32, count int32) int64

//go:module env
//export signalPoint
func SignalPoint(k int32, kl int32, v int32, lv int32, p int32, pv int32, c int32, cv int32) int64

//go:module env
//export runDocker
func RunDocker(k int32, kl int32, v int32, lv int32, c int32, cv int32) int64

//go:module env
//export execDocker
func ExecDocker(k int32, kl int32, v int32, lv int32, c int32, cv int32) int64

//go:module env
//export copyToDocker
func CopyToDocker(k int32, kl int32, v int32, lv int32, c int32, cv int32, con int32, conv int32) int64

//go:module env
//export put
func Put(k int32, kl int32, v int32, lv int32) int64

//go:module env
//export del
func Del(k int32, kl int32) int64

//go:module env
//export get
func Get(k int32, kl int32) int64

//go:module env
//export getByPrefix
func GetByPrefix(k int32, kl int32) int64

//go:module env
//export consoleLog
func ConsoleLog(k int32, kl int32) int64

//go:module env
//export submitOnchainTrx
func SubmitOnchainTrx(tmO int32, tmL int32, keyO int32, keyL int32, inputO int32, inputL int32, metaO int32, metaL int32) int64

//go:module env
//export output
func Output(k int32, kl int32) int64

//go:module env
//export newSyncTask
func NewSyncTask(k int32, kl int32) int64
