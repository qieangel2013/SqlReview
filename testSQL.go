package main

import (
	"./lib"
	"./redis"
	// "fmt"
	"strconv"
	"time"
)

func main() {
	// 加载配置
	appConfig := julive_com.NewConf("test")
	// 创建一个管道
	SqlExplain := make(chan julive_com.ExplainsResult, 10)

	addr := appConfig.RedisHost + ":" + strconv.Itoa(appConfig.RedisPort)
	redis_client_config := julive_com.RedisConfType{
		RedisPw:          appConfig.RedisPw,
		RedisHost:        addr,
		RedisDb:          appConfig.RedisDb,
		RedisMaxActive:   appConfig.RedisMaxActive,
		RedisMaxIdle:     appConfig.RedisMaxIdle,
		RedisIdleTimeOut: appConfig.RedisIdleTimeOut,
	}
	redis_client := julive_com.NewRedisPool(redis_client_config)
	go julive_com.NewExplainData(appConfig, redis_client, SqlExplain)

	go dispatch("ALL", appConfig, SqlExplain)

	timerActive := time.NewTimer(time.Second * time.Duration(appConfig.TimerActive))

	for {
		select {
		case <-timerActive.C:
			go checkBranch(redis_client, appConfig, SqlExplain)
			timerActive.Reset(time.Second * time.Duration(appConfig.TimerActive))
		}
	}
}

func checkBranch(redis_client *redis.Pool, appconfig *julive_com.AppConfig, SqlExplain chan julive_com.ExplainsResult) {
	branch := dispatch("single", appconfig, SqlExplain)
	for _, v := range branch {
		data, err := julive_com.ListCount(redis_client, v)
		if err != nil {
			julive_com.Error.Println(err)
		}
		if data != 0 {
			go sendData(v, appconfig, SqlExplain)
		}
	}
}

func dispatch(Type string, appconfig *julive_com.AppConfig, SqlExplain chan julive_com.ExplainsResult) []string {
	branchActive := int(appconfig.BranchActive)
	branchArr := make([]string, branchActive)
	for i := 0; i < branchActive; i++ {
		branch := "SQL_" + strconv.Itoa(i)
		if Type == "ALL" {
			go sendData(branch, appconfig, SqlExplain)
		} else {
			branchArr[i] = branch
		}
	}
	return branchArr
}

func sendData(branchName string, appconfig *julive_com.AppConfig, SqlExplain chan julive_com.ExplainsResult) {
	addr := appconfig.RedisHost + ":" + strconv.Itoa(appconfig.RedisPort)
	redis_client_config := julive_com.RedisConfType{
		RedisPw:          appconfig.RedisPw,
		RedisHost:        addr,
		RedisDb:          appconfig.RedisDb,
		RedisMaxActive:   appconfig.RedisMaxActive,
		RedisMaxIdle:     appconfig.RedisMaxIdle,
		RedisIdleTimeOut: appconfig.RedisIdleTimeOut,
	}
	redis_client := julive_com.NewRedisPool(redis_client_config)
	for {
		data, err1 := julive_com.ListPop(redis_client, branchName)
		if err1 != nil {
			julive_com.Error.Println(err1)
		}
		if data == "" {
			break
		} else {
			dataTmp, err := julive_com.JsonDecode(data)
			if err != nil {
				julive_com.Error.Println(err)
			}
			go julive_com.ExcuteUnique(dataTmp, redis_client_config, appconfig, SqlExplain)
		}
	}
}
