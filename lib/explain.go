package julive_com

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	ExplainType = map[string]string{
		"ALL":             "L1",
		"index":           "L1",
		"range":           "L2",
		"index_subquery":  "L2",
		"unique_subquery": "L2",
		"index_merge":     "L2",
		"ref_or_null":     "L2",
		"fulltext":        "L2",
		"ref":             "L3",
		"eq_ref":          "L3",
		"const":           "L4",
		"system":          "L4",
	}
	ExplainTypeScore = map[string]int{
		"L1": 15,
		"L2": 10,
		"L3": 5,
		"L4": 0,
	}
)

func Capitalize(str string) string {
	var upperStr string
	vv := []rune(str)
	for i := 0; i < len(vv); i++ {
		if i == 0 {
			if vv[i] >= 97 && vv[i] <= 122 {
				vv[i] -= 32
				upperStr += string(vv[i])
			} else {
				Info.Println(str)
				Error.Println("Not begins with lowercase letter")
				return str
			}
		} else {
			upperStr += string(vv[i])
		}
	}
	return upperStr
}

func ExcuteDbSql(sql ExplainsResult, appconfig *AppConfig, SqlExplain chan ExplainsResult) {
	//创建mysql 客户端
	var Host, Db, User, Pw, Charset string
	var Port, MaxOpenConns, MaxIdleConns int
	configTmp := reflect.ValueOf(appconfig).Elem()
	typeOfType := configTmp.Type()
	DbUpon := ""
	for i := 0; i < configTmp.NumField(); i++ {
		k := configTmp.Field(i)
		if k.Type().String() == "string" {
			if k.Interface().(string) == sql.Db {
				DbUpon = typeOfType.Field(i).Name
			}
		}
	}
	Info.Println(DbUpon)
	for i := 0; i < configTmp.NumField(); i++ {
		k := configTmp.Field(i)
		switch typeOfType.Field(i).Name {
		case string(DbUpon + "_MysqlHost"):
			Host = k.Interface().(string)
		case string(DbUpon + "_MysqlDb"):
			Db = k.Interface().(string)
		case string(DbUpon + "_MysqlUser"):
			User = k.Interface().(string)
		case string(DbUpon + "_MysqlPw"):
			Pw = k.Interface().(string)
		case string(DbUpon + "_MysqlCharset"):
			Charset = k.Interface().(string)
		case string(DbUpon + "_MysqlPort"):
			Port = k.Interface().(int)
		case string(DbUpon + "_MysqlMaxOpenConns"):
			MaxOpenConns = k.Interface().(int)
		case string(DbUpon + "_MysqlMaxIdleConns"):
			MaxIdleConns = k.Interface().(int)
		}
	}
	mysql_client_config := MysqlConfType{
		MysqlHost:         Host,
		MysqlDb:           Db,
		MysqlUser:         User,
		MysqlPw:           Pw,
		MysqlCharset:      Charset,
		MysqlPort:         Port,
		MysqlMaxOpenConns: MaxOpenConns,
		MysqlMaxIdleConns: MaxIdleConns,
	}
	Info.Println(mysql_client_config)
	mysql_client := InitMySQLPool(mysql_client_config)
	defer mysql_client.Close()
	sqlStr := string(sql.Arr[0].Info)
	sqlStr = strings.Replace(sqlStr, "'", "\"", -1)
	sqlStr = strings.Replace(sqlStr, "\"", "\"", -1)
	if strings.Index(sqlStr, "u003c") != -1 {
		sqlStr = strings.Replace(sqlStr, "u003c", "<", -1)
	}
	if strings.Index(sqlStr, "u003e") != -1 {
		sqlStr = strings.Replace(sqlStr, "u003e", ">", -1)
	}
	sqlStr = "EXPLAIN " + sqlStr
	Explaindata, err := mysql_client.Query(sqlStr)
	if err != nil {
		Error.Println(err)
	}
	if len(Explaindata) > 0 {
		for _, sqlTmp := range Explaindata {
			explainsTmp := Explains{}
			explainsTmp.Id = interface{}(sqlTmp["id"]).(uint64)
			explainsTmp.Select_type = interface{}(sqlTmp["select_type"]).(string)
			explainsTmp.Table = interface{}(sqlTmp["table"]).(string)
			explainsTmp.Type = interface{}(sqlTmp["type"]).(string)
			explainsTmp.Possible_keys = interface{}(sqlTmp["possible_keys"]).(string)
			explainsTmp.Key = interface{}(sqlTmp["key"]).(string)
			explainsTmp.Key_len = interface{}(sqlTmp["key_len"]).(string)
			explainsTmp.Ref = interface{}(sqlTmp["ref"]).(string)
			explainsTmp.Rows = interface{}(sqlTmp["rows"]).(uint64)
			explainsTmp.Extra = interface{}(sqlTmp["Extra"]).(string)
			sql.Explain = append(sql.Explain, explainsTmp)
		}
	}
	ExplaindataConst, err := mysql_client.Query(`show status like 'Last_query_cost'`)
	if err != nil {
		Error.Println(err)
	}
	if len(ExplaindataConst) > 0 {
		last_query_cost, _ := interface{}(ExplaindataConst[0]["Value"]).(string)
		sql.Last_query_cost = last_query_cost
	}
	Info.Println(sql)
	go getScore(sql, appconfig, SqlExplain)
}

func getScore(sql ExplainsResult, appconfig *AppConfig, SqlExplain chan ExplainsResult) {
	//计算得分
	var score = 100
	//type计算
	scoreTypeTmp := 0
	flag := 0
	scoreType := 0
	for _, sqlTmp := range sql.Explain {
		if sqlTmp.Type == "" {
			if strings.Index(sqlTmp.Extra, "const table") != -1 {
				sql.Level = "L1"
				scoreType += 1
				sql.Content += sqlTmp.Table + "使用type为：Null,未找到const tables，扣分为-1,"
			} else {
				sql.Content += sqlTmp.Table + "使用type为：Null,扣分为-15,"
				sql.Level = "L1"
				scoreType += 15
				if sqlTmp.Key == "" {
					scoreType += 5
					sql.Content += "使用索引key为Null,扣分为-5,"
				}
				if strings.Index(sqlTmp.Extra, "index") == -1 {
					scoreType += 5
					sql.Content += "未使用index索引,扣分为-5,"
				} else if strings.Index(sqlTmp.Extra, "where") == -1 {
					scoreType += 3
					sql.Content += "未使用where过滤条件,扣分为-3,"
				}
				if int(sqlTmp.Rows) > 5000 && int(sqlTmp.Rows) < 10000 {
					scoreType += 5
					sql.Content += "返回rows行数为：" + strconv.Itoa(int(sqlTmp.Rows)) + ",扣分为-5,"
				} else if int(sqlTmp.Rows) > 10000 && int(sqlTmp.Rows) < 100000 {
					scoreType += 10
					sql.Content += "返回rows行数为：" + strconv.Itoa(int(sqlTmp.Rows)) + ",扣分为-10,"
				} else if int(sqlTmp.Rows) > 100000 && int(sqlTmp.Rows) < 1000000 {
					scoreType += 15
					sql.Content += "返回rows行数为：" + strconv.Itoa(int(sqlTmp.Rows)) + ",扣分为-15,"
				} else if int(sqlTmp.Rows) > 1000000 && int(sqlTmp.Rows) < 10000000 {
					scoreType += 20
					sql.Content += "返回rows行数为：" + strconv.Itoa(int(sqlTmp.Rows)) + ",扣分为-20,"
				}
			}
			break
		}
		if sqlTmp.Type == "ALL" {
			sql.Content += sqlTmp.Table + "使用type为：ALL,扣分为-15,"
			sql.Level = "L1"
			scoreType += 15
			if sqlTmp.Key == "" {
				scoreType += 5
				sql.Content += "使用索引key为Null,扣分为-5,"
			}
			if strings.Index(sqlTmp.Extra, "index") == -1 {
				scoreType += 5
				sql.Content += "未使用index索引,扣分为-5,"
			} else if strings.Index(sqlTmp.Extra, "where") == -1 {
				scoreType += 3
				sql.Content += "未使用where过滤条件,扣分为-3,"
			}
			if int(sqlTmp.Rows) > 5000 && int(sqlTmp.Rows) < 10000 {
				scoreType += 5
				sql.Content += "返回rows行数为：" + strconv.Itoa(int(sqlTmp.Rows)) + ",扣分为-5,"
			} else if int(sqlTmp.Rows) > 10000 && int(sqlTmp.Rows) < 100000 {
				scoreType += 10
				sql.Content += "返回rows行数为：" + strconv.Itoa(int(sqlTmp.Rows)) + ",扣分为-10,"
			} else if int(sqlTmp.Rows) > 100000 && int(sqlTmp.Rows) < 1000000 {
				scoreType += 15
				sql.Content += "返回rows行数为：" + strconv.Itoa(int(sqlTmp.Rows)) + ",扣分为-15,"
			} else if int(sqlTmp.Rows) > 1000000 && int(sqlTmp.Rows) < 10000000 {
				scoreType += 20
				sql.Content += "返回rows行数为：" + strconv.Itoa(int(sqlTmp.Rows)) + ",扣分为-20,"
			}
			break
		}
		if scoreTypeTmp != 0 && scoreTypeTmp < ExplainTypeScore[ExplainType[sqlTmp.Type]] {
			scoreTypeTmp = ExplainTypeScore[ExplainType[sqlTmp.Type]]
		}
		if flag == 0 {
			scoreTypeTmp = ExplainTypeScore[ExplainType[sqlTmp.Type]]
			flag = ExplainTypeScore[ExplainType[sqlTmp.Type]]
		}

	}
	ExplainTypeScoreTmp := make(map[int]string)
	for k, v := range ExplainTypeScore {
		ExplainTypeScoreTmp[v] = k
	}
	if sql.Level == "" {
		sql.Level = ExplainTypeScoreTmp[scoreTypeTmp]
	}
	scoreType += scoreTypeTmp

	// 按其他项计算
	for _, sqlTmp := range sql.Explain {
		if sqlTmp.Type != "" && sqlTmp.Type != "ALL" && ExplainType[sqlTmp.Type] == ExplainTypeScoreTmp[scoreTypeTmp] {
			sql.Content += sqlTmp.Table + "使用type为：" + sqlTmp.Type + ",扣分为-" + strconv.Itoa(scoreTypeTmp) + ","
			if sqlTmp.Key == "" {
				scoreType += 5
				sql.Content += "使用索引key为Null,扣分为-5,"
			}
			if strings.Index(sqlTmp.Extra, "index") == -1 {
				scoreType += 5
				sql.Content += "未使用index索引,扣分为-5,"
			} else if strings.Index(sqlTmp.Extra, "where") == -1 {
				scoreType += 3
				sql.Content += "未使用where过滤条件,扣分为-3,"
			}
			if int(sqlTmp.Rows) > 5000 && int(sqlTmp.Rows) < 10000 {
				scoreType += 5
				sql.Content += "返回rows行数为：" + strconv.Itoa(int(sqlTmp.Rows)) + ",扣分为-5,"
			} else if int(sqlTmp.Rows) > 10000 && int(sqlTmp.Rows) < 100000 {
				scoreType += 10
				sql.Content += "返回rows行数为：" + strconv.Itoa(int(sqlTmp.Rows)) + ",扣分为-10,"
			} else if int(sqlTmp.Rows) > 100000 && int(sqlTmp.Rows) < 1000000 {
				scoreType += 15
				sql.Content += "返回rows行数为：" + strconv.Itoa(int(sqlTmp.Rows)) + ",扣分为-15,"
			} else if int(sqlTmp.Rows) > 1000000 && int(sqlTmp.Rows) < 10000000 {
				scoreType += 20
				sql.Content += "返回rows行数为：" + strconv.Itoa(int(sqlTmp.Rows)) + ",扣分为-20,"
			}

			break
		}
	}

	//计算lastQueryConst
	if num, _ := strconv.ParseFloat(sql.Last_query_cost, 32); int(num) > 500 && int(num) < 1000 {
		scoreType += 5
		sql.Content += "查询Last_query_cost为：" + sql.Last_query_cost + ",扣分为-5,"
	} else if num, _ := strconv.ParseFloat(sql.Last_query_cost, 32); int(num) > 1000 && int(num) < 10000 {
		scoreType += 10
		sql.Content += "查询Last_query_cost为：" + sql.Last_query_cost + ",扣分为-10,"
	} else if num, _ := strconv.ParseFloat(sql.Last_query_cost, 32); int(num) > 10000 && int(num) < 50000 {
		scoreType += 15
		sql.Content += "查询Last_query_cost为：" + sql.Last_query_cost + ",扣分为-15,"
	} else if num, _ := strconv.ParseFloat(sql.Last_query_cost, 32); int(num) > 50000 && int(num) < 100000 {
		scoreType += 20
		sql.Content += "查询Last_query_cost为：" + sql.Last_query_cost + ",扣分为-20,"
	} else if num, _ := strconv.ParseFloat(sql.Last_query_cost, 32); int(num) > 100000 && int(num) < 150000 {
		scoreType += 25
		sql.Content += "查询Last_query_cost为：" + sql.Last_query_cost + ",扣分为-25,"
	} else if num, _ := strconv.ParseFloat(sql.Last_query_cost, 32); int(num) > 150000 && int(num) < 200000 {
		scoreType += 30
		sql.Content += "查询Last_query_cost为：" + sql.Last_query_cost + ",扣分为-30,"
	} else if num, _ := strconv.ParseFloat(sql.Last_query_cost, 32); int(num) > 200000 && int(num) < 500000 {
		scoreType += 35
		sql.Content += "查询Last_query_cost为：" + sql.Last_query_cost + ",扣分为-35,"
	} else if num, _ := strconv.ParseFloat(sql.Last_query_cost, 32); int(num) > 500000 && int(num) < 1000000 {
		scoreType += 40
		sql.Content += "查询Last_query_cost为：" + sql.Last_query_cost + ",扣分为-40,"
	} else if num, _ := strconv.ParseFloat(sql.Last_query_cost, 32); int(num) > 1000000 && int(num) < 10000000 {
		scoreType += 50
		sql.Content += "查询Last_query_cost为：" + sql.Last_query_cost + ",扣分为-50,"
	}
	//计算最后得分
	sql.Score = score - scoreType

	Info.Println(sql)

	SqlExplain <- sql
}

func NewExplainResult(appconfig *AppConfig, redis_conf RedisConfType, SqlExplain chan ExplainsResult) {
	timerActive := time.NewTimer(time.Second * time.Duration(appconfig.ExplainDBTimer))
	for {
		select {
		case <-timerActive.C:
			redis_client := NewRedisPool(redis_conf)
			for i := 0; i < appconfig.ExplainDBNumber; i++ {
				data, _ := ListPop(redis_client, appconfig.Redisquename)
				if data == "" {
					continue
				}
				dataTmp, err := JsonSqlExpDecode(string(data))
				if err != nil {
					Error.Println(err)
				}
				go ExcuteDbSql(dataTmp, appconfig, SqlExplain)
			}
			timerActive.Reset(time.Second * time.Duration(appconfig.ExplainDBTimer))
		}
	}
}

func NewExplainDb(appconfig *AppConfig, redis_conf RedisConfType, SqlExplain chan ExplainsResult) {
	mysql_client_config := MysqlConfType{
		MysqlHost:         appconfig.MysqlHost,
		MysqlDb:           appconfig.MysqlDb,
		MysqlUser:         appconfig.MysqlUser,
		MysqlPw:           appconfig.MysqlPw,
		MysqlCharset:      appconfig.MysqlCharset,
		MysqlPort:         appconfig.MysqlPort,
		MysqlMaxOpenConns: appconfig.MysqlMaxOpenConns,
		MysqlMaxIdleConns: appconfig.MysqlMaxIdleConns,
	}
	for {
		select {
		//读取sqldata
		case sqldata := <-SqlExplain:
			go func() {
				now := time.Now()
				sqlInfo := sqldata.Arr
				sqlInfoTmp, err := json.Marshal(sqlInfo)
				if err != nil {
					Error.Println(err)
				}
				sqlInfoData := string(sqlInfoTmp)
				sqlInfoData = strings.Replace(sqlInfoData, "u003c", "<", -1)
				sqlInfoData = strings.Replace(sqlInfoData, "u003e", ">", -1)
				sqlInfoData = strings.Replace(sqlInfoData, "&lt", "<", -1)
				sqlInfoData = strings.Replace(sqlInfoData, "&gt", ">", -1)
				sqlInfoData = strings.Replace(sqlInfoData, "u0026", "&", -1)
				sqlInfoData = strings.Replace(sqlInfoData, "'", "\\'", -1)
				explainInfo := sqldata.Explain
				explainInfoTmp, err := json.Marshal(explainInfo)
				if err != nil {
					Error.Println(err)
				}
				explainInfoData := string(explainInfoTmp)
				explainInfoData = strings.Replace(explainInfoData, "u003c", "<", -1)
				explainInfoData = strings.Replace(explainInfoData, "u003e", ">", -1)
				explainInfoData = strings.Replace(explainInfoData, "&lt", "<", -1)
				explainInfoData = strings.Replace(explainInfoData, "&gt", ">", -1)
				explainInfoData = strings.Replace(explainInfoData, "u0026", "&", -1)
				sqldata.Expl = strings.Replace(sqldata.Expl, "u003c", "<", -1)
				sqldata.Expl = strings.Replace(sqldata.Expl, "u003e", ">", -1)
				sqldata.Expl = strings.Replace(sqldata.Expl, "u0026", "&", -1)
				sqldata.Expl = strings.Replace(sqldata.Expl, "&lt", "<", -1)
				sqldata.Expl = strings.Replace(sqldata.Expl, "&gt", ">", -1)
				explainInfoData = strings.Replace(explainInfoData, "'", "\\'", -1)
				redis_client := NewRedisPool(redis_conf)
				u := NewUniqueQueue(redis_client)
				//入去重队列
				branchName := "julive:" + sqldata.Branch
				resp, err := u.UniquePush(branchName, NewMd5(sqldata.Expl))
				if err != nil {
					Error.Println(err)
				}
				flag := 0
				if resp == 1 {
					flag = 1
				}
				mysql_client := InitMySQLPool(mysql_client_config)
				defer mysql_client.Close()
				SqlStr := fmt.Sprintf("INSERT INTO sqlexplain(`expl`,`route`,`sql_info`,`explain`,`last_query_cost`,`level`,`score`,`content`,`branch`,`db`,`flag`,`daytime`,`create_datetime`) VALUES('%s','%s','%s','%s',%v,'%s',%d,'%s','%s','%s',%d,'%s','%s')", sqldata.Expl, sqldata.Route, sqlInfoData, explainInfoData, sqldata.Last_query_cost, sqldata.Level, sqldata.Score, sqldata.Content, sqldata.Branch, sqldata.Db, flag, now.Format("2006/01/02"), now.Format("2006/01/02 15:04:05"))
				Info.Println(SqlStr)
				result, err := mysql_client.Insert(SqlStr)
				if err != nil {
					Error.Println(err)
				}
				Info.Println(result)
			}()
		}
	}
}
