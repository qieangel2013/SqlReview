package julive_com

import (
	"encoding/json"
	"strings"
)

type traces struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
	Class    string `json:"class"`
	Type     string `json:"type"`
}

type SqlInfo struct {
	Info   string   `json:"info"`
	Trace  []traces `json:"trace"`
	Route  string   `json:"route"`
	Branch string   `json:"branch"`
	Dsn    string   `json:"dsn"`
}

type Explains struct {
	Id            uint64 `json:"id"`
	Select_type   string `json:"select_type"`
	Table         string `json:"table"`
	Type          string `json:"type"`
	Possible_keys string `json:"possible_keys"`
	Key           string `json:"key"`
	Key_len       string `json:"key_len"`
	Ref           string `json:"ref"`
	Rows          uint64 `json:"rows"`
	Extra         string `json:"extra"`
}

type ExplainsResult struct {
	Expl            string     `json:"expl"`
	Arr             []SqlInfo  `json:"arr"`
	Explain         []Explains `json:"explain"`
	Last_query_cost string     `json:"last_query_cost"`
	Level           string     `json:"level"`
	Score           int        `json:"score"`
	Content         string     `json:"content"`
	Branch          string     `json:"branch"`
	Db              string     `json:"db"`
	Route           string     `json:"route"`
}

func JsonEncode(sql SqlInfo) (sqlInfo []byte, err error) {
	sqlInfo, err = json.Marshal(sql)
	if err != nil {
		Error.Println(err)
	}
	//json是[]byte类型，转化成string类型便于查看
	return sqlInfo, err
}

func JsonDecode(data string) (SqlInfo, error) {
	if strings.Index(data, "\"{") == -1 {
		data = strings.Replace(data, "^", "", -1)
		data = strings.Replace(data, "\\", "", -1)
		data = strings.Replace(data, "\"", "\"", -1)
	}
	str := []byte(data)
	//1.Unmarshal的第一个参数是json字符串，第二个参数是接受json解析的数据结构。
	//第二个参数必须是指针，否则无法接收解析的数据
	sqlInfo := SqlInfo{}
	err := json.Unmarshal(str, &sqlInfo)
	//解析失败会报错，如json字符串格式不对，缺"号，缺}等。
	if err != nil {
		Error.Println(err)
	}
	return sqlInfo, err
}

func JsonSqlExpEncode(sql ExplainsResult) (sqlInfo []byte, err error) {
	sqlInfo, err = json.Marshal(sql)
	if err != nil {
		Error.Println(err)
	}
	//jsonStu是[]byte类型，转化成string类型便于查看
	return sqlInfo, err
}

func JsonSqlExpDecode(data string) (ExplainsResult, error) {
	if strings.Index(data, "\"{") == -1 {
		data = strings.Replace(data, "^", "", -1)
		data = strings.Replace(data, "\\", "", -1)
		data = strings.Replace(data, "\"", "\"", -1)
	}

	str := []byte(data)
	//1.Unmarshal的第一个参数是json字符串，第二个参数是接受json解析的数据结构。
	//第二个参数必须是指针，否则无法接收解析的数据
	sqlInfo := ExplainsResult{}
	err := json.Unmarshal(str, &sqlInfo)
	//解析失败会报错，如json字符串格式不对，缺"号，缺}等。
	if err != nil {
		Error.Println(err)
	}
	return sqlInfo, err
}
