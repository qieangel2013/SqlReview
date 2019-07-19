package main

import (
	"./lib"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

var appConfig = &julive_com.AppConfig{}
var redis_client_config julive_com.RedisConfType

// 创建一个管道
var SqlExplain = make(chan julive_com.ExplainsResult, 10)

func handler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() //解析参数，默认是不会解析的
	if r.Method == "POST" {
		result, err := ioutil.ReadAll(r.Body)
		if err != nil {
			julive_com.Error.Println(err)
		}
		r.Body.Close()
		go func() {
			//创建 redis客户端
			redis_client := julive_com.NewRedisPool(redis_client_config)
			julive_com.Info.Println(string(result))
			julive_com.ListPush(redis_client, appConfig.Redisquename, string(result))
		}()
		//导入到kafka
		go func() {
			kafka_client_config := julive_com.KafkaConfType{
				KafkaHost:      appConfig.KafkaHost,
				KafkaPort:      appConfig.KafkaPort,
				KafkaTopic:     appConfig.KafkaTopic,
				KafkaChan:      appConfig.KafkaChan,
				KafkaThreadNum: appConfig.KafkaThreadNum,
			}
			julive_com.Info.Println(kafka_client_config)
			kafkaData, err := julive_com.JsonSqlExpDecode(string(result))
			if err != nil {
				julive_com.Error.Println(err)
			}
			kafkaTmp := make(map[string]interface{})
			kafkaTmp["expl"] = kafkaData.Expl
			kafkaTmp["arr"] = kafkaData.Arr[0]
			kafkaTmp["branch"] = kafkaData.Branch
			kafkaTmp["db"] = kafkaData.Db
			kafkaTmp["route"] = kafkaData.Route
			kafkaResult, err := json.Marshal(kafkaTmp)
			if err != nil {
				julive_com.Error.Println(err)
			}
			kafkaDataRes := string(kafkaResult)
			kafkaDataRes = strings.Replace(kafkaDataRes, "u003c", "<", -1)
			kafkaDataRes = strings.Replace(kafkaDataRes, "u003e", ">", -1)
			kafkaDataRes = strings.Replace(kafkaDataRes, "&lt", "<", -1)
			kafkaDataRes = strings.Replace(kafkaDataRes, "&gt", ">", -1)
			kafkaDataRes = strings.Replace(kafkaDataRes, "u0026", "&", -1)
			julive_com.Info.Println(kafkaDataRes)
			//生产者
			kafka_produce_client, err := julive_com.NewKafkaSender(kafka_client_config)
			if err != nil {
				julive_com.Error.Println(err)
			}
			// 添加一条消息
			julive_com.NewMessage(kafka_produce_client, appConfig.KafkaTopic, kafkaDataRes)
		}()
	}
}

func main() {
	// 加载配置
	appConfig = julive_com.NewConf("prod")
	//创建 redis客户端
	redis_addr := appConfig.RedisHost + ":" + strconv.Itoa(appConfig.RedisPort)
	redis_client_config = julive_com.RedisConfType{
		RedisPw:          appConfig.RedisPw,
		RedisHost:        redis_addr,
		RedisDb:          appConfig.RedisDb,
		RedisMaxActive:   appConfig.RedisMaxActive,
		RedisMaxIdle:     appConfig.RedisMaxIdle,
		RedisIdleTimeOut: appConfig.RedisIdleTimeOut,
	}
	//获取explain结果
	go julive_com.NewExplainResult(appConfig, redis_client_config, SqlExplain)
	// 入库
	go julive_com.NewExplainDb(appConfig, redis_client_config, SqlExplain)
	addr := appConfig.ListenHost + ":" + strconv.Itoa(appConfig.ListenPort)
	http.HandleFunc("/", handler) //	设置访问路由
	http.ListenAndServe(addr, nil)
}
