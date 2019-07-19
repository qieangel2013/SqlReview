package julive_com

import (
	"../redis"
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
)

func ExcuteSql(sql ExplainsResult, appconfig *AppConfig, SqlExplain chan ExplainsResult) {
	sqlStr := string(sql.Arr[0].Info)
	sqlStr = strings.Replace(sqlStr, "'", "\"", -1)
	sqlStr = strings.Replace(sqlStr, "\"", "\"", -1)
	cmdstr := fmt.Sprintf("echo '%s'|./soarbin/soar -test-dsn='%s:%s@%s:%d/%s' -online-dsn='%s:%s@%s:%d/%s' -allow-online-as-test -report-type json", sqlStr, appconfig.SoarTestDsnUser, appconfig.SoarTestDsnPw, appconfig.SoarTestDsnHost, appconfig.SoarTestDsnPort, appconfig.SoarTestDsnDb, appconfig.SoarOnlineDsnUser, appconfig.SoarOnlineDsnPw, appconfig.SoarOnlineDsnHost, appconfig.SoarOnlineDsnPort, appconfig.SoarOnlineDsnDb)
	go execCommandSql(cmdstr, sql, appconfig, SqlExplain)
}

func getUnicode(data string) string {
	start := strings.Index(data, "\\u003c")
	end := strings.Index(data[start:], "\\u003e")
	data = strings.Replace(data, data[start:start+end+6], "", -1)
	return data
}

func analyseSql(sqlContent string, sql ExplainsResult, appconfig *AppConfig, SqlExplain chan ExplainsResult) {
	//sql.Explain = sqlContent
	comma := strings.Index(sqlContent, "Explain")
	start := strings.Index(sqlContent[comma:], "Content")
	pos := strings.Index(sqlContent[comma+start:], "Position")
	explainContent := sqlContent[comma+start+11 : comma+start+pos-10]
	Info.Println(explainContent)
	data := strings.SplitN(explainContent, "\\n", -1)
	Info.Println(data[0])
	Info.Println(data[2])
	explanHeader := strings.SplitN(data[0], "|", -1)
	explanPargam := strings.SplitN(data[2], "|", -1)
	explainResult := make(map[string]interface{})
	for i := 0; i < len(explanHeader); i++ {
		if explanHeader[i] != "" {
			explanHeader[i] = strings.Replace(explanHeader[i], "\\\\", "", -1)
			explanPargam[i] = strings.Replace(explanPargam[i], "\\\\", "", -1)
			if strings.Index(explanPargam[i], "\\u") != -1 {
				explanPargam[i] = getUnicode(explanPargam[i])
			}
			explainResult[explanHeader[i]] = explanPargam[i]
		}
	}

	Info.Println(explainResult)
	// SqlExplain <- sql
}

func execCommandSql(params string, sql ExplainsResult, appconfig *AppConfig, SqlExplain chan ExplainsResult) bool {
	sysType := runtime.GOOS
	var cmmd, cmmdTmp string
	if sysType == "linux" {
		// LINUX系统
		cmmd = "/bin/bash"
		cmmdTmp = "-c"
	}
	if sysType == "windows" {
		// windows系统
		cmmd = "cmd"
		cmmdTmp = "/C"
	}
	//函数返回一个*Cmd，用于使用给出的参数执行name指定的程序
	cmd := exec.Command(cmmd, cmmdTmp, params)
	//StdoutPipe方法返回一个在命令Start后与命令标准输出关联的管道。Wait方法获知命令结束后会关闭这个管道，一般不需要显式的关闭该管道。
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		Error.Println(err)
		return false
	}
	defer stdout.Close()
	cmd.Start()
	//创建一个流来读取管道内内容，这里逻辑是通过一行一行的读取的
	reader := bufio.NewReader(stdout)
	//实时循环读取输出流中的一行内容
	sqlContent := ""
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		sqlContent += line
	}
	Info.Println(sqlContent)
	go analyseSql(sqlContent, sql, appconfig, SqlExplain)
	//阻塞直到该命令执行完成，该命令必须是被Start方法开始执行的
	cmd.Wait()
	return true
}

func ExcuteUnique(sql SqlInfo, redis_conf RedisConfType, appconfig *AppConfig, SqlExplain chan ExplainsResult) {
	sqlStr := string(sql.Info)
	sqlStr = strings.Replace(sqlStr, "'", "\"", -1)
	sqlStr = strings.Replace(sqlStr, "\"", "\"", -1)
	if strings.Index(sql.Info, "\\u003c") != -1 {
		sql.Info = strings.Replace(sql.Info, "\\u003c", "<", -1)
	}
	if strings.Index(sql.Info, "\\u003e") != -1 {
		sql.Info = strings.Replace(sql.Info, "\\u003e", ">", -1)
	}
	if strings.Index(sql.Info, "\u003c") != -1 {
		sql.Info = strings.Replace(sql.Info, "\u003c", "<", -1)
	}
	if strings.Index(sql.Info, "\u003e") != -1 {
		sql.Info = strings.Replace(sql.Info, "\u003e", ">", -1)
	}
	// cmdstr := fmt.Sprintf("echo '%s'|./soarbin/soar -report-type fingerprint", sqlStr)
	if sqlStr != "" {
		fingerprint := strings.TrimSpace(Fingerprint(sqlStr))
		defer func() {
			if err := recover(); err != nil {
				Info.Println("panic %s\n", err)
			}
		}()
		go importToRedis(fingerprint, redis_conf, sql, appconfig, SqlExplain)
	}
	//go execCommand(cmdstr, redis_client, sql, appconfig, SqlExplain)
}

func importToRedis(line string, redis_conf RedisConfType, sql SqlInfo, appconfig *AppConfig, SqlExplain chan ExplainsResult) bool {
	if line == "" {
		Error.Println("获取line数据为空")
		return false
	}
	redis_client := NewRedisPool(redis_conf)
	u := NewUniqueQueue(redis_client)
	Info.Println(line)
	// line = line[:len(line)-1]
	if strings.Index(line, "u003e") != -1 {
		line = strings.Replace(line, "u003e", ">", -1)
	}
	if strings.Index(line, "u003c") != -1 {
		line = strings.Replace(line, "u003c", "<", -1)
	}
	Info.Println(line)
	//入去重队列
	branchName := "julive_" + sql.Branch
	resp, err := u.UniquePush(branchName, line)
	if err != nil {
		Error.Println(err)
	}
	if resp == 1 {
		SetKey(redis_client, NewMd5(line), "1")
	}
	// 添加取样数据
	if SqlKey, _ := GetKey(redis_client, NewMd5(line)); SqlKey != "" {
		sqlexplains := ExplainsResult{}
		sqlexplains.Expl = line
		sqlexplains.Branch = sql.Branch
		pos := strings.Index(sql.Dsn, "dbname=")
		end := strings.Index(sql.Dsn[pos:], "]")
		if pos != -1 && end != -1 {
			sqlexplains.Db = sql.Dsn[pos+7 : pos+end]
		}
		sqlexplains.Route = sql.Route
		sqlexplains.Arr = append(sqlexplains.Arr, sql)
		SqlExplain <- sqlexplains
	}
	return true
}

func execCommand(params string, redis_client *redis.Pool, sql SqlInfo, appconfig *AppConfig, SqlExplain chan ExplainsResult) bool {
	sysType := runtime.GOOS
	var cmmd, cmmdTmp string
	if sysType == "linux" {
		// LINUX系统
		cmmd = "/bin/sh"
		cmmdTmp = "-c"
	}

	if sysType == "windows" {
		// windows系统
		cmmd = "cmd"
		cmmdTmp = "/C"
	}

	output, err := exec.Command(cmmd, cmmdTmp, params).CombinedOutput()

	if err != nil {
		Error.Println(err)
		return false
	}
	line := fmt.Sprintf("%v", string(output))
	//函数返回一个*Cmd，用于使用给出的参数执行name指定的程序
	// cmd := exec.Command(cmmd, cmmdTmp, params)
	// //StdoutPipe方法返回一个在命令Start后与命令标准输出关联的管道。Wait方法获知命令结束后会关闭这个管道，一般不需要显式的关闭该管道。
	// stdout, err := cmd.StdoutPipe()
	// if err != nil {
	// 	Error.Println(err)
	// 	return false
	// }

	// if err := cmd.Start(); err != nil {
	// 	Error.Println("Start: ", err.Error())
	// }
	// //创建一个流来读取管道内内容，这里逻辑是通过一行一行的读取的
	// reader := bufio.NewReader(stdout)
	// //实时循环读取输出流中的一行内容
	// for {
	// 	line, err2 := reader.ReadString('\n')
	// 	if err2 != nil || io.EOF == err2 {
	// 		break
	// 	}
	if line == "" {
		Error.Println("获取line数据为空")
		return false
	}
	u := NewUniqueQueue(redis_client)
	Info.Println(line)
	line = line[:len(line)-1]
	if strings.Index(line, "u003e") != -1 {
		line = strings.Replace(line, "u003e", ">", -1)
	}
	if strings.Index(line, "u003c") != -1 {
		line = strings.Replace(line, "u003c", "<", -1)
	}
	Info.Println(line)
	//入去重队列
	branchName := "julive_" + sql.Branch
	resp, err := u.UniquePush(branchName, line)
	if err != nil {
		Error.Println(err)
	}
	if resp == 1 {
		SetKey(redis_client, NewMd5(line), "1")
	}
	// 添加取样数据
	if SqlKey, _ := GetKey(redis_client, NewMd5(line)); SqlKey != "" {
		sqlexplains := ExplainsResult{}
		sqlexplains.Expl = line
		sqlexplains.Branch = sql.Branch
		pos := strings.Index(sql.Dsn, "dbname=")
		end := strings.Index(sql.Dsn[pos:], "]")
		if pos != -1 && end != -1 {
			sqlexplains.Db = sql.Dsn[pos+7 : pos+end]
		}
		sqlexplains.Route = sql.Route
		sqlexplains.Arr = append(sqlexplains.Arr, sql)
		SqlExplain <- sqlexplains
	}
	//}
	// //阻塞直到该命令执行完成，该命令必须是被Start方法开始执行的
	// if err := cmd.Wait(); err != nil {
	// 	Error.Println("Wait: ", err.Error())
	// }
	// defer stdout.Close()
	return true
}

func NewExplainData(appconfig *AppConfig, redis_client *redis.Pool, SqlExplain chan ExplainsResult) {
	SqlExplainArr := make(map[string]ExplainsResult)
	for {
		select {
		//读取sqldata
		case sqldata := <-SqlExplain:
			_, ok := SqlExplainArr[NewMd5(sqldata.Expl)]
			if ok {
				flag := 0
				if len(SqlExplainArr[NewMd5(sqldata.Expl)].Arr) < int(appconfig.RedisSampling) {
					for _, sqlTmp := range SqlExplainArr[NewMd5(sqldata.Expl)].Arr {
						if sqldata.Arr[0].Info == sqlTmp.Info {
							flag = 1
						}
					}
					if flag == 0 {
						sqlExData := SqlExplainArr[NewMd5(sqldata.Expl)]
						sqlExData.Arr = append(sqlExData.Arr, sqldata.Arr[0])
						SqlExplainArr[NewMd5(sqldata.Expl)] = sqlExData
					}
				} else {
					DelKey(redis_client, NewMd5(sqldata.Expl))
					sqlExDataTmp := SqlExplainArr[NewMd5(sqldata.Expl)]
					delete(SqlExplainArr, NewMd5(sqldata.Expl))
					go curlURL(sqlExDataTmp, appconfig)
				}
			} else {
				SqlExplainArr[NewMd5(sqldata.Expl)] = sqldata
			}
		}
	}
}

func curlURL(sql ExplainsResult, appconfig *AppConfig) {
	url := appconfig.CurlUrl
	headers := map[string]string{
		"User-Agent": "go-agent",
		// "Authorization": "Bearer access_token",
		"Content-Type": "application/json",
	}
	req := NewRequest()
	resp, err := req.
		SetUrl(url).
		SetHost(appconfig.CurlHost).
		SetHeaders(headers).
		SetPostData(sql).
		Post()
	if err != nil {
		Error.Println(err)
	} else {
		if resp.IsOk() {
			Info.Println(resp.Body)
		} else {
			Error.Println(resp.Raw)
		}
	}
}
