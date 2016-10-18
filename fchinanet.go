package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type user struct {
	Account  string `xml:"account"`
	Passwd   string `xml:"passwd"`
	Id       string `xml:"id"`
	ServerId string `xml:"serverId"`
	Status   int    `xml:"status"`
	LastIp   string `xml:"lastIp"`
	BrasIp   string `xml:"BrasIp"`
}

type OnlinesS struct {
	Device string `json:"device"`
	Type   int    `json:"type"`
	Time   string `json:"time"`
	Code   int    `json:"code"`
	BrasIp string `json:"brasIp"`
	WanIp  string `json:"wanIp"`
}
type WifiOnlinesS struct {
	Onlines []OnlinesS `json:"onlines"`
}
type OnlineDevice struct {
	Status      string       `json:"status"`
	WifiOnlines WifiOnlinesS `json:"wifiOnlines"`
}

type Account_namesS string
type UserS struct {
	Id                 string           `json:"id"`
	Name               string           `json:"name"`
	Mobile             string           `json:"mobile"`
	Vcard              string           `json:"vcard"`
	Avatar             string           `json:"avatar"`
	Savatar            string           `json:"savatar"`
	Time               string           `json:"time"`
	Update             string           `json:"update"`
	Code               string           `json:"code"`
	Country            string           `json:"country"`
	Continue_login_day string           `json:"continue_login_day"`
	Last_login_time    string           `json:"last_login_time"`
	Carrier            string           `json:"carrier"`
	Clear              string           `json:"clear"`
	City_id            int              `json:"city_id"`
	Admin_flag         string           `json:"admin_flag"`
	Did                string           `json:"did"`
	Sign               int              `json:"sign"`
	Account_names      []Account_namesS `json:"account_names"`
}
type LoginResult struct {
	Status string `json:"status"`
	User   UserS  `json:"user"`
}

type TelecomWifiResS struct {
	Password string `json:"password"`
	Code     int    `json:"code"`
}
type PasswdJson struct {
	Status         string          `json:"status"`
	TelecomWifiRes TelecomWifiResS `json:"telecomWifiRes"`
}

type QTelecomWifiResS struct {
	Id       string `json:"id"`
	Password string `json:"password"`
	Code     int    `json:"code"`
}
type QrcodeJson struct {
	Status         string           `json:"status"`
	TelecomWifiRes QTelecomWifiResS `json:"telecomWifiRes"`
}

type OnlineResult struct {
	Status   string `json:"status"`
	Response string `json:"response"`
}
type XMLKey struct {
	Account, Passwd, Id, ServerId, Status, LastIp, BrasIp string
}

const divider = "##############################################"

var XMLKeyWord = XMLKey{"account", "passwd", "id", "serverId", "status", "lastIp", "brasIp"}

var (
	deviceStatus = [...]bool{false, false, false, false}
	id           string
	serverId     string
	account      = ""
	passwd       = ""
	client       *http.Client
	re           string
	brasIp       string
	wanIp        string
)

func saveUser() {
	u := &user{base64.StdEncoding.EncodeToString([]byte(account)), base64.StdEncoding.EncodeToString([]byte(passwd)), id, serverId, 0, "0", "0"}
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
	case XMLKeyWord.Status:
		code, err := strconv.Atoi(content)
		if err == nil {
			u.Status = code
		}
	case XMLKeyWord.LastIp:
		u.LastIp = content
	case XMLKeyWord.BrasIp:
		u.BrasIp = content
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

func login() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			re = "未知异常"
		}
		fmt.Println(re)
	}()
	if !checkNet() {
		fmt.Println("未连接校园网")
		return
	}

	if re = loginChinaNet(); re == "0" {
		updateUser(XMLKeyWord.Status, strconv.Itoa(1))
		updateUser(XMLKeyWord.LastIp, wanIp)
		re = "登陆成功"
	}
}

func checkEncry() bool {
	user := getUser()
	account = user.Account
	passwd = user.Passwd
	if account == "" || passwd == "" {
		return false
	}
	if strings.HasSuffix(account, "=") {
		a, errDecodeA := base64.StdEncoding.DecodeString(account)
		checkErr(errDecodeA, "Error Decoding account")
		p, errDecodeP := base64.StdEncoding.DecodeString(passwd)
		checkErr(errDecodeP, "Error Decoding password")
		account = string(a)
		passwd = string(p)
	} else {
		updateUser(XMLKeyWord.Account, base64.StdEncoding.EncodeToString([]byte(account)))
		updateUser(XMLKeyWord.Passwd, base64.StdEncoding.EncodeToString([]byte(passwd)))
	}
	return true
}

func checkNet() bool {
	fmt.Println("正在检测网络...")
	rep, errGet := http.Get("http://pre.f-young.cn/")
	if errGet != nil {
		return false
	}
	if rep.StatusCode == 200 {
		return true
	}
	return false

}

func initial() {
	fmt.Println("正在初始化...")
	req, err := http.NewRequest("GET", "HTTP://test.f-young.cn/", nil)
	checkErr(err, "error Requesting")
	rep, err1 := http.DefaultTransport.RoundTrip(req)
	checkErr(err1, "error Getting")
	if rep.StatusCode == 302 {
		content := rep.Header.Get("Location")
		argString := strings.Split(content, "?")
		args := strings.SplitN(argString[1], "&", -1)
		wanIp = strings.Split(args[0], "=")[1]
		brasIp = strings.Split(args[1], "=")[1]
	}
}

func loginChinaNet() string {
	u := getUser()
	request, errNewRequest := http.NewRequest("GET", "https://www.loocha.com.cn:8443/login", nil)
	checkErr(errNewRequest, "error new request")

	request.SetBasicAuth(account, passwd)
	response, errRequest := newClient().Do(request)
	checkErr(errRequest, "error requseting")
	if response.StatusCode == http.StatusOK {
		body, errRead := ioutil.ReadAll(response.Body)
		checkErr(errRead, "error reading")
		defer response.Body.Close()
		loginResult := &LoginResult{}
		errUnmarshal := json.Unmarshal(body, loginResult)
		checkErr(errUnmarshal, "error unmarshal")
		id = loginResult.User.Id
		serverId = loginResult.User.Did
		if serverId != "" {
			serverId = strings.Split(serverId, "#")[0]
		}

		if id == "" || serverId == "" {
			u = getUser()
			id = u.Id
			serverId = u.ServerId

		} else {
			saveUser()
		}

		result, full := checkLogin()
		if result {
			return "登陆成功"
		} else {
			if full {
				return "设备已满"
			} else {
				return online()
			}

		}
	}
	if response.StatusCode == http.StatusUnauthorized {
		return "账号或密码错误"
	}

	return "登陆异常，请重新尝试"
}

func checkLogin() (result bool, full bool) {
	u := getUser()
	result = false
	full = false
	request, errNewRequest := http.NewRequest("GET", "https://wifi.loocha.cn/"+u.Id+"/wifi/status", nil)
	checkErr(errNewRequest, "error new request")
	a, _ := base64.StdEncoding.DecodeString(u.Account)
	p, _ := base64.StdEncoding.DecodeString(u.Passwd)
	request.SetBasicAuth(string(a), string(p))
	response, errRequest := newClient().Do(request)
	checkErr(errRequest, "error requseting")
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		onlineDevice := &OnlineDevice{}
		body, errRead := ioutil.ReadAll(response.Body)
		checkErr(errRead, "error reading")
		errUnmarshal := json.Unmarshal(body, onlineDevice)
		checkErr(errUnmarshal, "Error Unmarshal")
		for _, d := range onlineDevice.WifiOnlines.Onlines {
			if d.WanIp == u.LastIp {
				result = true
			}
		}
		if len(onlineDevice.WifiOnlines.Onlines) == 3 {
			full = true
		}

	}

	return
}
func kickOffDevice(ip, brasIp string) {
	fmt.Println("正在下线...")
	checkEncry()
	u := getUser()
	request, errNewRequest := http.NewRequest("DELETE", "https://wifi.loocha.cn/"+u.Id+"/wifi/kickoff?wanip="+ip+"&brasip="+brasIp, nil)
	checkErr(errNewRequest, "error new request")
	a, _ := base64.StdEncoding.DecodeString(u.Account)
	p, _ := base64.StdEncoding.DecodeString(u.Passwd)
	request.SetBasicAuth(string(a), string(p))
	response, errRequest := newClient().Do(request)
	checkErr(errRequest, "error requseting")
	if responseCode := response.StatusCode; responseCode == 200 {
		defer response.Body.Close()
		fmt.Println("下线成功")
		return
	}
	fmt.Println("下线失败")
}

func getPasswd() string {
	request, errNewRequest := http.NewRequest("GET", "https://www.loocha.com.cn/"+id+"/wifi?server_did="+serverId, nil)
	checkErr(errNewRequest, "error new request")
	request.SetBasicAuth(account, passwd)
	response, errRequest := newClient().Do(request)
	checkErr(errRequest, "error requseting")
	if response.StatusCode == http.StatusOK {
		defer response.Body.Close()
		body, errRead := ioutil.ReadAll(response.Body)
		checkErr(errRead, "error reading")
		passwdJson := &PasswdJson{}
		errUnmarshal := json.Unmarshal(body, passwdJson)
		checkErr(errUnmarshal, "Error Unmarshal")
		code := passwdJson.TelecomWifiRes.Password
		if len(code) != 6 {
			return ""
		}
		return code
	}
	return ""
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
	if response.StatusCode == http.StatusOK {
		defer response.Body.Close()
		body, errRead := ioutil.ReadAll(response.Body)
		checkErr(errRead, "error reading")
		qrcodeJson := &QrcodeJson{}
		errUnmarshal := json.Unmarshal(body, qrcodeJson)
		checkErr(errUnmarshal, "Error Unmarshal")
		qrcode := qrcodeJson.TelecomWifiRes.Password
		if qrcode != "" {
			return qrcode
		}
	}

	return ""

}

func online() string {
	initial()
	updateUser(XMLKeyWord.BrasIp, brasIp)
	if wanIp == "" || brasIp == "" {
		return "0"
	}
	fmt.Println("正在登陆...")
	code := getPasswd()
	if code == "" {
		return "密码获取错误请在掌上大学重新获取密码后尝试"
	}
	history := [...]int{-1, -1}
	for i := 0; i < 3; i++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		t := r.Intn(9)
		for _, h := range history {
			if t == h {
				t = r.Intn(9)
			}
		}
		if i != 2 {
			history[i] = t
		}
		qrcode := getQrCode()
		if qrcode == "" {
			return "code获取异常"
		}
		param := "qrcode=" + qrcode + "&code=" + code + "&type="
		param += strconv.Itoa(t)
		request, errNewRequest := http.NewRequest("POST", "https://wifi.loocha.cn/"+id+"/wifi/enable?"+param, nil)
		checkErr(errNewRequest, "error new request")
		request.SetBasicAuth(account, passwd)
		response, errRequest := newClient().Do(request)
		checkErr(errRequest, "error requseting")
		if response.StatusCode == http.StatusOK {
			defer response.Body.Close()
			body, errRead := ioutil.ReadAll(response.Body)
			checkErr(errRead, "error reading")
			onlineResult := &OnlineResult{}
			errUnmarshal := json.Unmarshal(body, onlineResult)
			checkErr(errUnmarshal, "Error Unmarshal")
			status := onlineResult.Status
			if status == "0" {
				return "0"
			} else {
				r := onlineResult.Response
				if r == "检测到你的帐号在其他设备登录" {
					// continue
				}
				return r
			}
		}
	}
	return "拨号异常"

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

func menu() {
	line := "*****"
	fmt.Println(line, "键入选择的数字后回车", line)
	fmt.Println(line, "1. 登陆", line)
	fmt.Println(line, "2. 下线", line)
	opt := ""
	fmt.Scanln(&opt)
	o := 0
	opt = strings.TrimSpace(opt)
	var err error
	if o, err = strconv.Atoi(opt); err != nil || (o != 1 && o != 2) {
		menu()
	}
	switch o {
	case 1:
		fmt.Println(divider)
		login()
		fmt.Println(divider)
	case 2:
		fmt.Println(divider)
		kickOffDevice(getUser().LastIp, getUser().BrasIp)
		fmt.Println(divider)
	}
	fmt.Println()
	menu()
}

func main() {
	if checkEncry() {
		menu()
	} else {
		fmt.Println("请在user.xml中输入账号密码")
	}
	opt := ""
	fmt.Scanln(&opt)
}
