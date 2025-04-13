package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"kasper/src/abstract"
	module_logger "kasper/src/core/module/logger"
	plugger_social "kasper/src/plugins/social/main"
	sigma "kasper/src/shell"
	inputs_users "kasper/src/shell/api/inputs/users"
	plugger_api "kasper/src/shell/api/main"
	outputs_users "kasper/src/shell/api/outputs/users"
	"kasper/src/bots/sampleBot"
	models_hokmaent "kasper/src/bots/sampleBot/models"
	plugger_machiner "kasper/src/shell/machiner/main"
	"kasper/src/shell/utils/future"

	"kasper/src/babble"
	"kasper/src/proxy/inmem"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"gorm.io/driver/mysql"
	// "gorm.io/driver/postgres"

	"kasper/src/shell/layer1/adapters"
	layer1 "kasper/src/shell/layer1/layer"
	module_model "kasper/src/shell/layer1/model"
	module_model1 "kasper/src/shell/layer1/module/toolbox"
	layer2 "kasper/src/shell/layer2/layer"
	layer3 "kasper/src/shell/layer3/layer"
	module_model3 "kasper/src/shell/layer3/model"

	actor_model "kasper/src/core/module/actor/model"

	"os"

	_ "github.com/go-sql-driver/mysql"
)

var KasperApp sigma.Sigma

// NewRunCmd returns the command that starts a Babble node
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Run node",
		PreRunE: bindFlagsLoadViper,
		RunE:    runBabble,
	}
	AddRunFlags(cmd)
	return cmd
}

/*******************************************************************************
* RUN
*******************************************************************************/

func getEnvWithDefault(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func getDSN(ipAddress string) string {
	tidbHost := getEnvWithDefault("TIDB_HOST", ipAddress)
	tidbPort := getEnvWithDefault("TIDB_PORT", "4000")
	tidbUser := getEnvWithDefault("TIDB_USER", "root")
	tidbPassword := getEnvWithDefault("TIDB_PASSWORD", "")
	tidbDBName := getEnvWithDefault("TIDB_DB_NAME", "test")
	useSSL := getEnvWithDefault("USE_SSL", "false")

	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&tls=%s",
		tidbUser, tidbPassword, tidbHost, tidbPort, tidbDBName, useSSL)
}

var exit = make(chan int, 1)

func RunNet() error {

	future.Async(func() {
		cmnd := exec.Command("tidb-server")
		cmnd.Start()
	}, false)

	future.Async(func() {
		cmnd := exec.Command("redis-server")
		cmnd.Start()
	}, false)

	time.Sleep(5 * time.Second)

	logger := new(module_logger.Logger)

	logger.Println("Welcome to Kasper !")

	err2 := godotenv.Load()
	if err2 != nil {
		panic(err2)
	}

	app := sigma.NewApp(sigma.Config{
		Id:  os.Getenv("ORIGIN"),
		Log: logger.Println,
	})

	KasperApp = app

	_config.Babble.Logger().WithFields(logrus.Fields{
		"ProxyAddr":  _config.ProxyAddr,
		"ClientAddr": _config.ClientAddr,
	}).Debug("Config Proxy")

	handler := app.NewHgHandler()

	// We create an InmemProxy based on the handler.
	proxy := inmem.NewInmemProxy(handler, nil)

	// Set the AppProxy in the Babble configuration.
	_config.Babble.Proxy = proxy

	engine := babble.NewBabble(&_config.Babble)

	if err := engine.Init(); err != nil {
		_config.Babble.Logger().Error("Cannot initialize engine:", err)
		return err
	}

	app.Load(
		[]string{
			"keyhan",
		},
		[]abstract.ILayer{
			layer1.New(),
			layer2.New(),
			layer3.New(),
		},
		[]interface{}{
			logger,
			os.Getenv("STORAGE_ROOT_PATH"),
			// postgres.Open(os.Getenv("DB_URI")),
			mysql.Open(getDSN(app.IpAddr())),
			os.Getenv("REDIS_URI"),
			engine,
			proxy,
			os.Getenv("APPLET_DB_PATH"),
		},
	)

	defer func() {
		tbl2 := abstract.UseToolbox[module_model1.IToolboxL1](app.Get(2).Tools())
		dbraw, errdb := tbl2.Storage().Db().DB()
		if errdb != nil {
			log.Println(errdb)
		}
		dbraw.Close()
	}()

	portStr := os.Getenv("MAINPORT")
	port, _ := strconv.ParseInt(portStr, 10, 64)
	plugger_api.PlugAll(app.Get(1), logger, app)
	plugger_machiner.PlugAll(app.Get(1), logger, app)

	abstract.UseToolbox[*module_model3.ToolboxL3](app.Get(3).Tools()).Net().Run(
		map[string]int{
			"http": int(port),
		},
	)

	time.Sleep(time.Duration(5) * time.Second)

	var sampleBotUserId string
	var sampleBotUserToken string
	e2 := abstract.UseToolbox[module_model1.IToolboxL1](app.Get(2).Tools()).Storage().DoTrx(func(trx adapters.ITrx) error {
		_, res, err := app.Get(1).Actor().FetchAction("/users/login").Act(app.Get(1).Sb().NewState(actor_model.NewInfo("", "", "", ""), trx), inputs_users.LoginInput{
			Username: "sampleBot",
		})
		if err != nil {
			log.Println(err)
			return err
		}
		sampleBotUserId = res.(outputs_users.LoginOutput).User.Id
	    sampleBotUserToken = res.(outputs_users.LoginOutput).Session.Token
		return nil
	})
	if e2 != nil {
		panic(e2)
	}
	ha := &hokmagent.HokmAgent{}
	ha.Install(app, sampleBotUserToken)
	abstract.UseToolbox[*module_model1.ToolboxL1](app.Get(1).Tools()).Signaler().ListenToSingle(&module_model.Listener{
		Id: sampleBotUserId,
		Signal: func(a any) {
			data := string(a.([]byte))
			dataParts := strings.Split(data, " ")
			if dataParts[1] == "topics/send" {
				data = data[len(dataParts[0])+1+len(dataParts[1])+1:]
				var inp models_hokmaent.Send
				e := json.Unmarshal([]byte(data), &inp)
				if e != nil {
					log.Println(e)
				}
				ha.OnTopicSend(inp)
			}
		},
	})

	plugger_social.PlugAll(app.Get(2), logger, app)

	<-exit

	return nil
}

func runBabble(cmd *cobra.Command, args []string) error {
	return RunNet()
}

/*******************************************************************************
* CONFIG
*******************************************************************************/

// AddRunFlags adds flags to the Run command
func AddRunFlags(cmd *cobra.Command) {

	cmd.Flags().String("datadir", _config.Babble.DataDir, "Top-level directory for configuration and data")
	cmd.Flags().String("log", _config.Babble.LogLevel, "debug, info, warn, error, fatal, panic")
	cmd.Flags().String("moniker", _config.Babble.Moniker, "Optional name")
	cmd.Flags().BoolP("maintenance-mode", "R", _config.Babble.MaintenanceMode, "Start Babble in a suspended (non-gossipping) state")

	// Network
	cmd.Flags().StringP("listen", "l", _config.Babble.BindAddr, "Listen IP:Port for babble node")
	cmd.Flags().StringP("advertise", "a", _config.Babble.AdvertiseAddr, "Advertise IP:Port for babble node")
	cmd.Flags().DurationP("timeout", "t", _config.Babble.TCPTimeout, "TCP Timeout")
	cmd.Flags().DurationP("join-timeout", "j", _config.Babble.JoinTimeout, "Join Timeout")
	cmd.Flags().Int("max-pool", _config.Babble.MaxPool, "Connection pool size max")

	// WebRTC
	cmd.Flags().Bool("webrtc", _config.Babble.WebRTC, "Use WebRTC transport")
	cmd.Flags().String("signal-addr", _config.Babble.SignalAddr, "IP:Port of WebRTC signaling server")
	cmd.Flags().Bool("signal-skip-verify", _config.Babble.SignalSkipVerify, "(Insecure) Accept any certificate presented by the signal server")
	cmd.Flags().String("ice-addr", _config.Babble.ICEAddress, "URI of a server providing ICE services such as STUN and TURN")
	cmd.Flags().String("ice-username", _config.Babble.ICEUsername, "Username to authenticate to the ICE server")
	cmd.Flags().String("ice-password", _config.Babble.ICEPassword, "Password to authenticate to the ICE server")

	// Proxy
	cmd.Flags().StringP("proxy-listen", "p", _config.ProxyAddr, "Listen IP:Port for babble proxy")
	cmd.Flags().StringP("client-connect", "c", _config.ClientAddr, "IP:Port to connect to client")

	// Service
	cmd.Flags().Bool("no-service", _config.Babble.NoService, "Disable HTTP service")
	cmd.Flags().StringP("service-listen", "s", _config.Babble.ServiceAddr, "Listen IP:Port for HTTP service")

	// Store
	cmd.Flags().Bool("store", _config.Babble.Store, "Use badgerDB instead of in-mem DB")
	cmd.Flags().String("db", _config.Babble.DatabaseDir, "Dabatabase directory")
	cmd.Flags().Bool("bootstrap", _config.Babble.Bootstrap, "Load from database")
	cmd.Flags().Int("cache-size", _config.Babble.CacheSize, "Number of items in LRU caches")

	// Node configuration
	cmd.Flags().Duration("heartbeat", _config.Babble.HeartbeatTimeout, "Timer frequency when there is something to gossip about")
	cmd.Flags().Duration("slow-heartbeat", _config.Babble.SlowHeartbeatTimeout, "Timer frequency when there is nothing to gossip about")
	cmd.Flags().Int("sync-limit", _config.Babble.SyncLimit, "Max number of events for sync")
	cmd.Flags().Bool("fast-sync", _config.Babble.EnableFastSync, "Enable FastSync")
	cmd.Flags().Int("suspend-limit", _config.Babble.SuspendLimit, "Limit of undetermined events (per node) before entering suspended state")
}

// Bind all flags and read the config into viper
func bindFlagsLoadViper(cmd *cobra.Command, args []string) error {
	// Register flags with viper. Include flags from this command and all other
	// persistent flags from the parent
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	// first unmarshal to read from CLI flags
	if err := viper.Unmarshal(_config); err != nil {
		return err
	}

	// look for config file in [datadir]/babble.toml (.json, .yaml also work)
	viper.SetConfigName("babble")               // name of config file (without extension)
	viper.AddConfigPath(_config.Babble.DataDir) // search root directory

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		_config.Babble.Logger().Debugf("Using config file: %s", viper.ConfigFileUsed())
	} else if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		_config.Babble.Logger().Debugf("No config file found in: %s", filepath.Join(_config.Babble.DataDir, "babble.toml"))
	} else {
		return err
	}

	// second unmarshal to read from config file
	return viper.Unmarshal(_config)
}
