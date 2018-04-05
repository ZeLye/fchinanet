package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type user struct {
	Account   string `xml:"account"`
	Passwd    string `xml:"passwd"`
	Id        string `xml:"id"`
	ServerId  string `xml:"serverId"`
	AutoLogin bool   `xml:"autoLogin"`
	Status    int    `xml:"status"`
	LastIp    string `xml:"lastIp"`
}
type XMLKey struct {
	Account, Passwd, Id, ServerId, AutoLogin, Status, LastIp string
}

var XMLKeyWord = XMLKey{"account", "passwd", "id", "serverId", "autoLogin", "status", "lastIp"}

var (
	id        string
	serverId  string
	account   = ""
	passwd    = ""
	autoLogin = false
	client    *http.Client
	re        string
	brasIp    string
	wanIp     string
)

func saveUser() {
	println("autoLogin", autoLogin)
	u := &user{base64.StdEncoding.EncodeToString([]byte(account)), base64.StdEncoding.EncodeToString([]byte(passwd)), id, serverId, autoLogin, 0, "0"}
	body, errMarshal := xml.MarshalIndent(u, "", "	")
	checkErr(errMarshal, "error marshal xml")

	errWrite := ioutil.WriteFile("user.xml", append([]byte(xml.Header), body...), 0777)
	checkErr(errWrite, "error writing")
}

func updateUser(name string, content string) {
	body, errRead := ioutil.ReadFile("user.xml")
	checkErr(errRead, "error reading user.xml")
	var u user
	xml.Unmarshal(body, &u)
	switch name {
	case XMLKeyWord.Account:
		u.Account = content
	case XMLKeyWord.Passwd:
		u.Passwd = content
	case XMLKeyWord.Id:
		u.Id = content
	case XMLKeyWord.ServerId:
		u.ServerId = content
	case XMLKeyWord.AutoLogin:
		u.AutoLogin, _ = strconv.ParseBool(content)
	case XMLKeyWord.Status:
		code, err := strconv.Atoi(content)
		if err == nil {
			u.Status = code
		}
	case XMLKeyWord.LastIp:
		u.LastIp = content
	}
	body, errMarshal := xml.MarshalIndent(u, "", "	")
	checkErr(errMarshal, "error marshaling")
	errWrite := ioutil.WriteFile("user.xml", body, 0777)
	checkErr(errWrite, "error writing")
}
func getUser() *user {
	body, errRead := ioutil.ReadFile("user.xml")
	checkErr(errRead, "error reading")
	var u user
	xml.Unmarshal(body, &u)
	return &u
}

func newClient() *http.Client {
	if client == nil {
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		client = &http.Client{Transport: tr}
	}
	return client
}

func checkErr(err error, info string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("[W]", r)
		}
	}()
	if err != nil {
		log.Println("panic:" + err.Error() + " info:" + info)
	}
}

func login(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			re = "登陆异常，请重新尝试"
			http.Redirect(w, req, "/result", http.StatusFound)
		}
	}()
	account = req.FormValue("account")
	passwd = req.FormValue("passwd")
	if req.FormValue("autoLogin") == "1" {
		autoLogin = true
	}

	if account == "" || passwd == "" {
		return
	}
	if re = loginChinaNet(); re == "0" {
		updateUser(XMLKeyWord.Status, strconv.Itoa(1))
		updateUser(XMLKeyWord.LastIp, wanIp)
		re = "登陆成功"
	}
	http.Redirect(w, req, "/result", http.StatusFound)
}

func initial() {
	req, err := http.NewRequest("GET", "HTTP://test.f-young.cn/", nil)
	checkErr(err, "error Requesting")
	rep, err1 := http.DefaultTransport.RoundTrip(req)
	checkErr(err1, "error Getting")
	content := rep.Header.Get("Location")
	argString := strings.Split(content, "?")
	args := strings.SplitN(argString[1], "&", -1)
	wanIp = strings.Split(args[0], "=")[1]
	brasIp = strings.Split(args[1], "=")[1]
}

func loginChinaNet() string {
	//判断是否已经登陆
	u := getUser()
	if u.Account == "" || u.Passwd == "" || u.Id == "" || u.ServerId == "" {

		request, errNewRequest := http.NewRequest("GET", "https://www.loocha.com.cn:8443/login", nil)
		checkErr(errNewRequest, "error new request")

		request.SetBasicAuth(account, passwd)
		response, errRequest := newClient().Do(request)
		checkErr(errRequest, "error requseting")

		body, errRead := ioutil.ReadAll(response.Body)
		checkErr(errRead, "error reading")
		response.Body.Close()
		reg := regexp.MustCompile(`\bid":"[\d]+"`)
		id = reg.FindString(string(body))
		if id != "" {
			id = id[5 : len(id)-1]
			reg = regexp.MustCompile(`"did":".+?#`)
			serverId = reg.FindString(string(body))
			serverId = serverId[7 : len(serverId)-1]
			saveUser()
		}
	}
	updateUser(XMLKeyWord.AutoLogin, strconv.FormatBool(autoLogin))
	if id == "" || serverId == "" {
		u = getUser()
		id = u.Id
		serverId = u.ServerId

	}
	result, oIp := isLogin()
	if result {
		return "登陆成功"
	} else {
		if oIp != "0" {
			kickOffDevice(oIp, brasIp)
		}
	}
	return online()

	return "登陆异常，请重新尝试"
}

func isLogin() (result bool, oIp string) {
	u := getUser()
	result = false
	oIp = "0"
	request, errNewRequest := http.NewRequest("GET", "https://wifi.loocha.cn/"+u.Id+"/wifi/status", nil)
	checkErr(errNewRequest, "error new request")
	a, _ := base64.StdEncoding.DecodeString(u.Account)
	p, _ := base64.StdEncoding.DecodeString(u.Passwd)
	request.SetBasicAuth(string(a), string(p))
	response, errRequest := newClient().Do(request)
	checkErr(errRequest, "error requseting")
	defer response.Body.Close()
	body, errRead := ioutil.ReadAll(response.Body)
	checkErr(errRead, "error reading")

	reg := regexp.MustCompile(`wanIp":".+?"`)

	dIps := reg.FindAllString(string(body), -1)
	for _, dIp := range dIps {
		println(dIp[8:len(dIp)-1], wanIp)
		if dIp[8:len(dIp)-1] == wanIp {
			result = true
		}
		tmp := getUser().LastIp
		if dIp[8:len(dIp)-1] == tmp {
			oIp = tmp
		}
	}
	return
}

func kickOff(w http.ResponseWriter, req *http.Request) {
	println("请求==================")
	if kickOffDevice(wanIp, brasIp) {
		fmt.Fprint(w, "true")
	} else {
		fmt.Fprint(w, "false")
	}

}

func kickOffDevice(ip string, brasIp string) bool {
	u := getUser()
	request, errNewRequest := http.NewRequest("DELETE", "https://wifi.loocha.cn/"+u.Id+"/wifi/kickoff?wanip="+ip+"&brasip="+brasIp, nil)
	checkErr(errNewRequest, "error new request")
	a, _ := base64.StdEncoding.DecodeString(u.Account)
	p, _ := base64.StdEncoding.DecodeString(u.Passwd)
	request.SetBasicAuth(string(a), string(p))
	response, errRequest := newClient().Do(request)
	checkErr(errRequest, "error requseting")
	defer response.Body.Close()
	if responseCode := response.StatusCode; responseCode == 200 {
		return true
	}
	return false
}

func getPasswd() string {
	request, errNewRequest := http.NewRequest("GET", "https://wifi.loocha.cn/"+id+"/wifi/telecom/pwd?type=4", nil)
	checkErr(errNewRequest, "error new request")
	request.SetBasicAuth(account, passwd)
	response, errRequest := newClient().Do(request)
	checkErr(errRequest, "error requseting")
	defer response.Body.Close()
	body, errRead := ioutil.ReadAll(response.Body)
	println(string(body))
	checkErr(errRead, "error reading")
	reg := regexp.MustCompile(`\bpassword":"[\d]+"`)
	code := reg.FindString(string(body))
	if code != "" {
		code = code[11 : len(code)-1]
		return code
	}
	return "登陆异常，请重新尝试"
}

func getQrCode() string {
	ip := wanIp
	if ip == "" {
		return ""
	}
	request, errNewRequest := http.NewRequest("GET", "https://wifi.loocha.cn/0/wifi/qrcode"+"?brasip="+brasIp+"&ulanip="+wanIp+"&wlanip="+wanIp, nil)
	checkErr(errNewRequest, "error new request")
	response, errRequest := newClient().Do(request)
	checkErr(errRequest, "error requseting")
	defer response.Body.Close()
	body, errRead := ioutil.ReadAll(response.Body)
	checkErr(errRead, "error reading")
	reg := regexp.MustCompile(`\bpassword":".+?"`)
	qrcode := reg.FindString(string(body))
	if qrcode != "" {
		qrcode = qrcode[11 : len(qrcode)-1]
		return qrcode
	}
	return "登陆异常，请重新尝试"

}

func online() string {
	initial()
	if wanIp == "" || brasIp == "" {
		return "0"
	}
	println(wanIp)
	code := getPasswd()
	qrcode := getQrCode()
	if qrcode == "" || code == "" {
		return ""
	}
	param := "qrcode=" + qrcode + "&code=" + code + "&type=1"
	println(param)
	request, errNewRequest := http.NewRequest("POST", "https://wifi.loocha.cn/"+id+"/wifi/telecom/auto/login?"+param, nil)
	checkErr(errNewRequest, "error new request")
	request.SetBasicAuth(account, passwd)
	response, errRequest := newClient().Do(request)
	checkErr(errRequest, "error requseting")
	defer response.Body.Close()
	body, errRead := ioutil.ReadAll(response.Body)
	checkErr(errRead, "error reading")
	arr := strings.Split(string(body), ",")
	status := strings.Replace(strings.Split(arr[0], ":")[1], "\"", "", -1)
	if status == "0" {
		return "0"
	} else {
		response := strings.Replace(strings.Split(arr[1], ":")[1], "\"", "", -1)
		response = response[:len(response)-1]
		return response
	}
}

func createUserFile() {
	file, errCreate := os.Create("user.xml")
	checkErr(errCreate, "error creating user.xml")
	defer file.Close()
}

func checkXML() {
	u := getUser()
	if u.Account == "" || u.Passwd == "" || u.Id == "" || u.ServerId == "" {
		updateUser(XMLKeyWord.Status, "0")
	}
}

func handle(w http.ResponseWriter, req *http.Request) {
	type Tmp struct {
		Account string
		Passwd  string
		Checked string
	}
	_, errOpen := os.Open("user.xml")
	if errOpen != nil {
		createUserFile()
	}

	_, errGet := http.Get("http://pre.f-young.cn/") //检测是否连接了校园网
	if errGet != nil {
		re = "未连接校园网"
		http.Redirect(w, req, "/result", http.StatusFound)
		return
	}
	checkXML()
	u := getUser()
	ok, _ := isLogin()
	if ok && u.Status == 1 { //检测是否已经登陆且账号密码存在
		re = "登陆成功"
		http.Redirect(w, req, "/result", http.StatusFound)
	}
	a, _ := base64.StdEncoding.DecodeString(u.Account)
	p, _ := base64.StdEncoding.DecodeString(u.Passwd)
	account = string(a)
	passwd = string(p)
	id = u.Id
	serverId = u.ServerId
	req.ParseForm()
	backs, ok := req.Form["back"]
	if ok {
		if back := backs[0]; back == "true" { //判断是否由结果页面返回

			tmp := Tmp{account, passwd, ""}
			if u.AutoLogin {
				tmp.Checked = "true"
			}
			t, errParse := template.ParseFiles("index.html")
			checkErr(errParse, "error parsing")
			err := t.Execute(w, tmp)
			checkErr(err, "error executing")
		}

	} else {
		if account != "" && passwd != "" && u.AutoLogin { //账号密码存在且autologin为true
			if re = loginChinaNet(); re == "0" {
				re = "登陆成功"
			}
			http.Redirect(w, req, "/result", http.StatusFound)
		} else {
			tmp := Tmp{account, passwd, ""}
			if u.AutoLogin {
				tmp.Checked = "true"
			}

			fmt.Printf("%+v", tmp)
			t, errParse := template.ParseFiles("index.html")
			checkErr(errParse, "error parsing")
			err := t.Execute(w, tmp)
			checkErr(err, "error executing")

		}
	}

}

func result(w http.ResponseWriter, req *http.Request) {
	type reqResult struct {
		Status string
		Action string
	}
	defer func() {
		if err := recover(); err != nil {
			re = "网络异常或其他原因，请检查您的网络"
			ac := "返回"
			rr := reqResult{re, ac}
			t, errParse := template.ParseFiles("result.html")
			checkErr(errParse, "error parsing result.html")
			err := t.Execute(w, rr)
			checkErr(err, "error executing")
		}
	}()

	ac := "下线"
	if re != "未连接校园网" {
		if ok, _ := isLogin(); ok {
			re = "登陆成功"
		}
	}

	if re != "登陆成功" {
		ac = "返回"
	}
	rr := reqResult{re, ac}
	t, errParse := template.ParseFiles("result.html")
	checkErr(errParse, "error parsing result.html")
	err := t.Execute(w, rr)
	checkErr(err, "error executing")
}

// func handle(w http.ResponseWriter, req *http.Request) {
// 	t, err := template.ParseFiles("index.html")
// 	if err != nil {
// 		log.Println("错误")
// 		return
// 	}
// 	t.Execute(w, nil)
// }

func server() {
	http.HandleFunc("/", handle)
	http.Handle("/login/", http.HandlerFunc(login))
	http.Handle("/kickOff/", http.HandlerFunc(kickOff))
	http.Handle("/result/", http.HandlerFunc(result))
	http.Handle("/css/", http.FileServer(http.Dir("")))
	http.Handle("/js/", http.FileServer(http.Dir("")))
	http.Handle("/img/", http.FileServer(http.Dir("")))
	errListen := http.ListenAndServe(":8090", nil)
	checkErr(errListen, "error listening")
}
func main() {
	println("服务已启动,端口：8090")
	server()
}
