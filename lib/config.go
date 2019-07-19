package julive_com

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	// "time"
)

// 定义一个通知的接口
type Notifyer interface {
	Callback(*Config)
}

type Config struct {
	filename       string
	lastModifyTime int64
	data           map[string]string
	rwLock         sync.RWMutex
	notifyList     []Notifyer
}

type AppConfig struct {
	// redis config
	RedisHost        string
	RedisPort        int
	RedisPw          string
	RedisDb          int
	RedisMaxActive   int
	RedisMaxIdle     int
	RedisIdleTimeOut int
	Redisquename     string
	RedisSampling    int
	// mysql config
	MysqlHost         string
	MysqlDb           string
	MysqlUser         string
	MysqlPw           string
	MysqlCharset      string
	MysqlPort         int
	MysqlMaxOpenConns int
	MysqlMaxIdleConns int
	// kafka config
	KafkaHost      string
	KafkaPort      int
	KafkaTopic     string
	KafkaChan      int
	KafkaThreadNum int
	// branch config
	BranchActive int
	TimerActive  int //单位秒
	// request
	CurlUrl  string
	CurlHost string
	// log config
	ServerLog string
	// Listen Port
	ListenHost string
	ListenPort int
	// soar config
	SoarTestDsnHost   string
	SoarTestDsnPort   int
	SoarTestDsnDb     string
	SoarTestDsnUser   string
	SoarTestDsnPw     string
	SoarOnlineDsnHost string
	SoarOnlineDsnPort int
	SoarOnlineDsnDb   string
	SoarOnlineDsnUser string
	SoarOnlineDsnPw   string
	//explain db
	ExplainDBTimer  int
	ExplainDBNumber int
	// comjia
	Comjia_MysqlHost         string
	Comjia_MysqlDb           string
	Comjia                   string
	Comjia_MysqlUser         string
	Comjia_MysqlPw           string
	Comjia_MysqlCharset      string
	Comjia_MysqlPort         int
	Comjia_MysqlMaxOpenConns int
	Comjia_MysqlMaxIdleConns int
	//esf
	Esf                   string
	Esf_MysqlHost         string
	Esf_MysqlDb           string
	Esf_MysqlUser         string
	Esf_MysqlPw           string
	Esf_MysqlCharset      string
	Esf_MysqlPort         int
	Esf_MysqlMaxOpenConns int
	Esf_MysqlMaxIdleConns int
	// bbs
	Bbs                   string
	Bbs_MysqlHost         string
	Bbs_MysqlDb           string
	Bbs_MysqlUser         string
	Bbs_MysqlPw           string
	Bbs_MysqlCharset      string
	Bbs_MysqlPort         int
	Bbs_MysqlMaxOpenConns int
	Bbs_MysqlMaxIdleConns int
	//dsp
	Dsp                   string
	Dsp_MysqlHost         string
	Dsp_MysqlDb           string
	Dsp_MysqlUser         string
	Dsp_MysqlPw           string
	Dsp_MysqlCharset      string
	Dsp_MysqlPort         int
	Dsp_MysqlMaxOpenConns int
	Dsp_MysqlMaxIdleConns int
}

type AppconfigMgr struct {
	config atomic.Value
}

var appConfigMgr = &AppconfigMgr{}

func NewConfig(filename string) (conf *Config, err error) {
	conf = &Config{
		filename: filename,
		data:     make(map[string]string, 1024),
	}
	m, err := conf.parse()
	if err != nil {
		return
	}
	conf.rwLock.Lock()
	conf.data = m
	conf.rwLock.Unlock()
	go conf.reload()
	return
}

func (c *Config) AddNotifyer(n Notifyer) {
	c.notifyList = append(c.notifyList, n)
}

func (c *Config) reload() {
	// 这里启动一个定时器，每5秒重新加载一次配置文件
	// ticker := time.NewTicker(time.Second * 5)
	// for _ = range ticker.C {
	// 	func() {
	file, err := os.Open(c.filename)
	if err != nil {
		Error.Println(err)
		return
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		Error.Println(err)
		return
	}
	curModifyTime := fileInfo.ModTime().Unix()
	if curModifyTime > c.lastModifyTime {
		m, err := c.parse()
		if err != nil {
			Error.Println(err)
			return
		}
		c.rwLock.Lock()
		c.data = m
		c.rwLock.Unlock()
		for _, n := range c.notifyList {
			n.Callback(c)
		}
		c.lastModifyTime = curModifyTime
		// 	}
		// }()
	}
}

func (c *Config) parse() (m map[string]string, err error) {
	// 读文件并或将文件中的数据以k/v的形式存储到map中
	m = make(map[string]string, 1024)
	file, err := os.Open(c.filename)
	if err != nil {
		Error.Println(err)
		return
	}
	var lineNo int
	reader := bufio.NewReader(file)
	for {
		// 一行行的读文件
		line, errRet := reader.ReadString('\n')
		if errRet == io.EOF {
			// 表示读到文件的末尾
			break
		}
		if errRet != nil {
			// 表示读文件出问题
			err = errRet
			return
		}
		lineNo++
		line = strings.TrimSpace(line) // 取出空格
		if len(line) == 0 || line[0] == '\n' || line[0] == '+' || line[0] == ';' {
			// 当前行为空行或者是注释行等
			continue
		}
		arr := strings.Split(line, "=") // 通过=进行切割取出k/v结构
		if len(arr) == 0 {
			Error.Println("invalid config,line:%d\n", lineNo)
			continue
		}
		key := strings.TrimSpace(arr[0])
		if len(key) == 0 {
			Error.Println("invalid config,line:%d\n", lineNo)
			continue
		}
		if len(arr) == 1 {
			m[key] = ""
			continue
		}
		value := strings.TrimSpace(arr[1])
		m[key] = value
	}
	return
}

func (c *Config) GetInt(key string) (value int, err error) {
	// 根据int获取
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()
	str, ok := c.data[key]
	if !ok {
		// err = fmt.Errorf("key[%s] not found", key)
		Error.Println("key[%s] not found", key)
		return
	}
	value, err = strconv.Atoi(str)
	return
}

func (c *Config) GetIntDefault(key string, defval int) (value int) {
	// 默认值
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()
	str, ok := c.data[key]
	if !ok {
		value = defval
		return
	}
	value, err := strconv.Atoi(str)
	if err != nil {
		value = defval
		return
	}
	return
}

func (c *Config) GetString(key string) (value string, err error) {
	// 根据字符串获取
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()
	value, ok := c.data[key]
	if !ok {
		//err = fmt.Errorf("key[%s] not found", key)
		Error.Println("key[%s] not found", key)
		return
	}
	return
}

func (a *AppconfigMgr) Callback(conf *Config) {
	appConfig := getConf(conf)
	appConfigMgr.config.Store(appConfig)
}

func getConf(conf *Config) *AppConfig {
	var appConfig = &AppConfig{}

	RedisPort, err := conf.GetInt("RedisPort")
	if err != nil {
		Error.Println("get RedisPort failed,err:", err)
	}
	appConfig.RedisPort = RedisPort

	RedisDb, err := conf.GetInt("RedisDb")
	if err != nil {
		Error.Println("get RedisDb failed,err:", err)
	}
	appConfig.RedisDb = RedisDb

	RedisMaxActive, err := conf.GetInt("RedisMaxActive")
	if err != nil {
		Error.Println("get RedisMaxActive failed,err:", err)
	}
	appConfig.RedisMaxActive = RedisMaxActive

	RedisMaxIdle, err := conf.GetInt("RedisMaxIdle")
	if err != nil {
		Error.Println("get RedisMaxIdle failed,err:", err)
	}
	appConfig.RedisMaxIdle = RedisMaxIdle

	RedisIdleTimeOut, err := conf.GetInt("RedisIdleTimeOut")
	if err != nil {
		Error.Println("get RedisIdleTimeOut failed,err:", err)
	}
	appConfig.RedisIdleTimeOut = RedisIdleTimeOut

	RedisSampling, err := conf.GetInt("RedisSampling")
	if err != nil {
		Error.Println("get RedisSampling failed,err:", err)
	}
	appConfig.RedisSampling = RedisSampling

	MysqlPort, err := conf.GetInt("MysqlPort")
	if err != nil {
		Error.Println("get MysqlPort failed,err:", err)
	}
	appConfig.MysqlPort = MysqlPort

	MysqlMaxOpenConns, err := conf.GetInt("MysqlMaxOpenConns")
	if err != nil {
		Error.Println("get MysqlMaxOpenConns failed,err:", err)
	}
	appConfig.MysqlMaxOpenConns = MysqlMaxOpenConns

	MysqlMaxIdleConns, err := conf.GetInt("MysqlMaxIdleConns")
	if err != nil {
		Error.Println("get MysqlMaxIdleConns failed,err:", err)
	}
	appConfig.MysqlMaxIdleConns = MysqlMaxIdleConns

	KafkaPort, err := conf.GetInt("KafkaPort")
	if err != nil {
		Error.Println("get KafkaPort failed,err:", err)
	}
	appConfig.KafkaPort = KafkaPort

	KafkaChan, err := conf.GetInt("KafkaChan")
	if err != nil {
		Error.Println("get KafkaChan failed,err:", err)
	}
	appConfig.KafkaChan = KafkaChan

	KafkaThreadNum, err := conf.GetInt("KafkaThreadNum")
	if err != nil {
		Error.Println("get KafkaThreadNum failed,err:", err)
	}
	appConfig.KafkaThreadNum = KafkaThreadNum

	BranchActive, err := conf.GetInt("BranchActive")
	if err != nil {
		Error.Println("get BranchActive failed,err:", err)
	}
	appConfig.BranchActive = BranchActive

	TimerActive, err := conf.GetInt("TimerActive")
	if err != nil {
		Error.Println("get TimerActive failed,err:", err)
	}

	appConfig.TimerActive = TimerActive

	RedisHost, err := conf.GetString("RedisHost")
	if err != nil {
		Error.Println("get RedisHost failed,err:", err)
	}
	appConfig.RedisHost = RedisHost

	RedisPw, err := conf.GetString("RedisPw")
	if err != nil {
		Error.Println("get RedisPw failed,err:", err)
	}
	appConfig.RedisPw = RedisPw

	Redisquename, err := conf.GetString("Redisquename")
	if err != nil {
		Error.Println("get Redisquename failed,err:", err)
	}
	appConfig.Redisquename = Redisquename

	MysqlHost, err := conf.GetString("MysqlHost")
	if err != nil {
		Error.Println("get MysqlHost failed,err:", err)
	}
	appConfig.MysqlHost = MysqlHost

	MysqlDb, err := conf.GetString("MysqlDb")
	if err != nil {
		Error.Println("get MysqlDb failed,err:", err)
	}
	appConfig.MysqlDb = MysqlDb

	MysqlUser, err := conf.GetString("MysqlUser")
	if err != nil {
		Error.Println("get MysqlUser failed,err:", err)
	}
	appConfig.MysqlUser = MysqlUser

	MysqlPw, err := conf.GetString("MysqlPw")
	if err != nil {
		Error.Println("get MysqlPw failed,err:", err)
	}
	appConfig.MysqlPw = MysqlPw

	MysqlCharset, err := conf.GetString("MysqlCharset")
	if err != nil {
		Error.Println("get MysqlCharset failed,err:", err)
	}
	appConfig.MysqlCharset = MysqlCharset

	KafkaHost, err := conf.GetString("KafkaHost")
	if err != nil {
		Error.Println("get KafkaHost failed,err:", err)
	}
	appConfig.KafkaHost = KafkaHost

	KafkaTopic, err := conf.GetString("KafkaTopic")
	if err != nil {
		Error.Println("get KafkaTopic failed,err:", err)
	}
	appConfig.KafkaTopic = KafkaTopic

	CurlUrl, err := conf.GetString("CurlUrl")
	if err != nil {
		Error.Println("get CurlUrl failed,err:", err)
	}
	appConfig.CurlUrl = CurlUrl

	CurlHost, err := conf.GetString("CurlHost")
	if err != nil {
		Error.Println("get CurlHost failed,err:", err)
	}
	appConfig.CurlHost = CurlHost

	ServerLog, err := conf.GetString("ServerLog")
	if err != nil {
		Error.Println("get ServerLog failed,err:", err)
	}
	appConfig.ServerLog = ServerLog

	ListenHost, err := conf.GetString("ListenHost")
	if err != nil {
		Error.Println("get ListenHost failed,err:", err)
	}
	appConfig.ListenHost = ListenHost

	ListenPort, err := conf.GetInt("ListenPort")
	if err != nil {
		Error.Println("get ListenPort failed,err:", err)
	}

	appConfig.ListenPort = ListenPort

	SoarTestDsnHost, err := conf.GetString("SoarTestDsnHost")
	if err != nil {
		Error.Println("get SoarTestDsnHost failed,err:", err)
	}
	appConfig.SoarTestDsnHost = SoarTestDsnHost

	SoarTestDsnPort, err := conf.GetInt("SoarTestDsnPort")
	if err != nil {
		Error.Println("get SoarTestDsnPort failed,err:", err)
	}

	appConfig.SoarTestDsnPort = SoarTestDsnPort

	SoarOnlineDsnPort, err := conf.GetInt("SoarOnlineDsnPort")
	if err != nil {
		Error.Println("get SoarOnlineDsnPort failed,err:", err)
	}

	appConfig.SoarOnlineDsnPort = SoarOnlineDsnPort

	SoarTestDsnDb, err := conf.GetString("SoarTestDsnDb")
	if err != nil {
		Error.Println("get SoarTestDsnDb failed,err:", err)
	}
	appConfig.SoarTestDsnDb = SoarTestDsnDb

	SoarTestDsnUser, err := conf.GetString("SoarTestDsnUser")
	if err != nil {
		Error.Println("get SoarTestDsnUser failed,err:", err)
	}
	appConfig.SoarTestDsnUser = SoarTestDsnUser

	SoarTestDsnPw, err := conf.GetString("SoarTestDsnPw")
	if err != nil {
		Error.Println("get SoarTestDsnPw failed,err:", err)
	}
	appConfig.SoarTestDsnPw = SoarTestDsnPw

	SoarOnlineDsnDb, err := conf.GetString("SoarOnlineDsnDb")
	if err != nil {
		Error.Println("get SoarOnlineDsnDb failed,err:", err)
	}
	appConfig.SoarOnlineDsnDb = SoarOnlineDsnDb

	SoarOnlineDsnHost, err := conf.GetString("SoarOnlineDsnHost")
	if err != nil {
		Error.Println("get SoarOnlineDsnHost failed,err:", err)
	}
	appConfig.SoarOnlineDsnHost = SoarOnlineDsnHost

	SoarOnlineDsnUser, err := conf.GetString("SoarOnlineDsnUser")
	if err != nil {
		Error.Println("get SoarOnlineDsnUser failed,err:", err)
	}
	appConfig.SoarOnlineDsnUser = SoarOnlineDsnUser

	SoarOnlineDsnPw, err := conf.GetString("SoarOnlineDsnPw")
	if err != nil {
		Error.Println("get SoarOnlineDsnPw failed,err:", err)
	}
	appConfig.SoarOnlineDsnPw = SoarOnlineDsnPw

	ExplainDBTimer, err := conf.GetInt("ExplainDBTimer")
	if err != nil {
		Error.Println("get ExplainDBTimer failed,err:", err)
	}
	appConfig.ExplainDBTimer = ExplainDBTimer

	ExplainDBNumber, err := conf.GetInt("ExplainDBNumber")
	if err != nil {
		Error.Println("get ExplainDBNumber failed,err:", err)
	}
	appConfig.ExplainDBNumber = ExplainDBNumber

	//comjia
	Comjia, err := conf.GetString("Comjia")
	if err != nil {
		Error.Println("get Comjia failed,err:", err)
	}
	appConfig.Comjia = Comjia

	Comjia_MysqlHost, err := conf.GetString("Comjia_MysqlHost")
	if err != nil {
		Error.Println("get Comjia_MysqlHost failed,err:", err)
	}
	appConfig.Comjia_MysqlHost = Comjia_MysqlHost

	Comjia_MysqlDb, err := conf.GetString("Comjia_MysqlDb")
	if err != nil {
		Error.Println("get Comjia_MysqlDb failed,err:", err)
	}
	appConfig.Comjia_MysqlDb = Comjia_MysqlDb

	Comjia_MysqlUser, err := conf.GetString("Comjia_MysqlUser")
	if err != nil {
		Error.Println("get Comjia_MysqlUser failed,err:", err)
	}
	appConfig.Comjia_MysqlUser = Comjia_MysqlUser

	Comjia_MysqlPw, err := conf.GetString("Comjia_MysqlPw")
	if err != nil {
		Error.Println("get Comjia_MysqlPw failed,err:", err)
	}
	appConfig.Comjia_MysqlPw = Comjia_MysqlPw

	Comjia_MysqlCharset, err := conf.GetString("Comjia_MysqlCharset")
	if err != nil {
		Error.Println("get Comjia_MysqlCharset failed,err:", err)
	}
	appConfig.Comjia_MysqlCharset = Comjia_MysqlCharset

	Comjia_MysqlPort, err := conf.GetInt("Comjia_MysqlPort")
	if err != nil {
		Error.Println("get Comjia_MysqlPort failed,err:", err)
	}
	appConfig.Comjia_MysqlPort = Comjia_MysqlPort

	Comjia_MysqlMaxOpenConns, err := conf.GetInt("Comjia_MysqlMaxOpenConns")
	if err != nil {
		Error.Println("get Comjia_MysqlMaxOpenConns failed,err:", err)
	}
	appConfig.Comjia_MysqlMaxOpenConns = Comjia_MysqlMaxOpenConns

	Comjia_MysqlMaxIdleConns, err := conf.GetInt("Comjia_MysqlMaxIdleConns")
	if err != nil {
		Error.Println("get Comjia_MysqlMaxIdleConns failed,err:", err)
	}
	appConfig.Comjia_MysqlMaxIdleConns = Comjia_MysqlMaxIdleConns

	//esf
	Esf, err := conf.GetString("Esf")
	if err != nil {
		Error.Println("get Esf failed,err:", err)
	}
	appConfig.Esf = Esf

	Esf_MysqlHost, err := conf.GetString("Esf_MysqlHost")
	if err != nil {
		Error.Println("get Esf_MysqlHost failed,err:", err)
	}
	appConfig.Esf_MysqlHost = Esf_MysqlHost

	Esf_MysqlDb, err := conf.GetString("Esf_MysqlDb")
	if err != nil {
		Error.Println("get Esf_MysqlDb failed,err:", err)
	}
	appConfig.Esf_MysqlDb = Esf_MysqlDb

	Esf_MysqlUser, err := conf.GetString("Esf_MysqlUser")
	if err != nil {
		Error.Println("get Esf_MysqlUser failed,err:", err)
	}
	appConfig.Esf_MysqlUser = Esf_MysqlUser

	Esf_MysqlPw, err := conf.GetString("Esf_MysqlPw")
	if err != nil {
		Error.Println("get Esf_MysqlPw failed,err:", err)
	}
	appConfig.Esf_MysqlPw = Esf_MysqlPw

	Esf_MysqlCharset, err := conf.GetString("Esf_MysqlCharset")
	if err != nil {
		Error.Println("get Esf_MysqlCharset failed,err:", err)
	}
	appConfig.Esf_MysqlCharset = Esf_MysqlCharset

	Esf_MysqlPort, err := conf.GetInt("Esf_MysqlPort")
	if err != nil {
		Error.Println("get Esf_MysqlPort failed,err:", err)
	}
	appConfig.Esf_MysqlPort = Esf_MysqlPort

	Esf_MysqlMaxOpenConns, err := conf.GetInt("Esf_MysqlMaxOpenConns")
	if err != nil {
		Error.Println("get Esf_MysqlMaxOpenConns failed,err:", err)
	}
	appConfig.Esf_MysqlMaxOpenConns = Esf_MysqlMaxOpenConns

	Esf_MysqlMaxIdleConns, err := conf.GetInt("Esf_MysqlMaxIdleConns")
	if err != nil {
		Error.Println("get Esf_MysqlMaxIdleConns failed,err:", err)
	}
	appConfig.Esf_MysqlMaxIdleConns = Esf_MysqlMaxIdleConns

	//bbs
	Bbs, err := conf.GetString("Bbs")
	if err != nil {
		Error.Println("get Bbs failed,err:", err)
	}
	appConfig.Bbs = Bbs

	Bbs_MysqlHost, err := conf.GetString("Bbs_MysqlHost")
	if err != nil {
		Error.Println("get Bbs_MysqlHost failed,err:", err)
	}
	appConfig.Bbs_MysqlHost = Bbs_MysqlHost

	Bbs_MysqlDb, err := conf.GetString("Bbs_MysqlDb")
	if err != nil {
		Error.Println("get Bbs_MysqlDb failed,err:", err)
	}
	appConfig.Bbs_MysqlDb = Bbs_MysqlDb

	Bbs_MysqlUser, err := conf.GetString("Bbs_MysqlUser")
	if err != nil {
		Error.Println("get Bbs_MysqlUser failed,err:", err)
	}
	appConfig.Bbs_MysqlUser = Bbs_MysqlUser

	Bbs_MysqlPw, err := conf.GetString("Bbs_MysqlPw")
	if err != nil {
		Error.Println("get Bbs_MysqlPw failed,err:", err)
	}
	appConfig.Bbs_MysqlPw = Bbs_MysqlPw

	Bbs_MysqlCharset, err := conf.GetString("Bbs_MysqlCharset")
	if err != nil {
		Error.Println("get Bbs_MysqlCharset failed,err:", err)
	}
	appConfig.Bbs_MysqlCharset = Bbs_MysqlCharset

	Bbs_MysqlPort, err := conf.GetInt("Bbs_MysqlPort")
	if err != nil {
		Error.Println("get Bbs_MysqlPort failed,err:", err)
	}
	appConfig.Bbs_MysqlPort = Bbs_MysqlPort

	Bbs_MysqlMaxOpenConns, err := conf.GetInt("Bbs_MysqlMaxOpenConns")
	if err != nil {
		Error.Println("get Bbs_MysqlMaxOpenConns failed,err:", err)
	}
	appConfig.Bbs_MysqlMaxOpenConns = Bbs_MysqlMaxOpenConns

	Bbs_MysqlMaxIdleConns, err := conf.GetInt("Bbs_MysqlMaxIdleConns")
	if err != nil {
		Error.Println("get Bbs_MysqlMaxIdleConns failed,err:", err)
	}
	appConfig.Bbs_MysqlMaxIdleConns = Bbs_MysqlMaxIdleConns

	//dsp
	Dsp, err := conf.GetString("Dsp")
	if err != nil {
		Error.Println("get Dsp failed,err:", err)
	}
	appConfig.Dsp = Dsp

	Dsp_MysqlHost, err := conf.GetString("Dsp_MysqlHost")
	if err != nil {
		Error.Println("get Dsp_MysqlHost failed,err:", err)
	}
	appConfig.Dsp_MysqlHost = Dsp_MysqlHost

	Dsp_MysqlDb, err := conf.GetString("Dsp_MysqlDb")
	if err != nil {
		Error.Println("get Dsp_MysqlDb failed,err:", err)
	}
	appConfig.Dsp_MysqlDb = Dsp_MysqlDb

	Dsp_MysqlUser, err := conf.GetString("Dsp_MysqlUser")
	if err != nil {
		Error.Println("get Dsp_MysqlUser failed,err:", err)
	}
	appConfig.Dsp_MysqlUser = Dsp_MysqlUser

	Dsp_MysqlPw, err := conf.GetString("Dsp_MysqlPw")
	if err != nil {
		Error.Println("get Dsp_MysqlPw failed,err:", err)
	}
	appConfig.Dsp_MysqlPw = Dsp_MysqlPw

	Dsp_MysqlCharset, err := conf.GetString("Dsp_MysqlCharset")
	if err != nil {
		Error.Println("get Dsp_MysqlCharset failed,err:", err)
	}
	appConfig.Dsp_MysqlCharset = Dsp_MysqlCharset

	Dsp_MysqlPort, err := conf.GetInt("Dsp_MysqlPort")
	if err != nil {
		Error.Println("get Dsp_MysqlPort failed,err:", err)
	}
	appConfig.Dsp_MysqlPort = Dsp_MysqlPort

	Dsp_MysqlMaxOpenConns, err := conf.GetInt("Dsp_MysqlMaxOpenConns")
	if err != nil {
		Error.Println("get Dsp_MysqlMaxOpenConns failed,err:", err)
	}
	appConfig.Dsp_MysqlMaxOpenConns = Dsp_MysqlMaxOpenConns

	Dsp_MysqlMaxIdleConns, err := conf.GetInt("Dsp_MysqlMaxIdleConns")
	if err != nil {
		Error.Println("get Dsp_MysqlMaxIdleConns failed,err:", err)
	}
	appConfig.Dsp_MysqlMaxIdleConns = Dsp_MysqlMaxIdleConns

	return appConfig
}

func NewConf(env string) *AppConfig {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		Error.Println("get current path failed,err:", err)
	}
	configPath := dir + "/env/" + env + ".conf"
	conf, err := NewConfig(configPath)
	if err != nil {
		Error.Println("get config failed,err:", err)
	}
	//打开文件获取内容后，将自己加入到被通知的切片中
	// conf.AddNotifyer(appConfigMgr)
	appConfig := getConf(conf)
	// appConfigMgr.config.Store(appConfig)
	return appConfig
}
