package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"unsafe"

	"github.com/asaskevich/govalidator"
	"github.com/valyala/fasthttp"
)

var (
	addr                 = flag.String("addr", "192.168.1.102:8080", "TCP address to listen to")
	dir                  = flag.String("dir", "./", "Directory to serve static files from")
	ipnet                = flag.String("ipnet", "192.168.1", "IPNet to allow access from")
	jsonSuccess          = []byte(`{"code": 200}`)
	jsonWrongIPNet       = []byte(`{"code": 20000, "msg": "不在同一网段，请您到指定地点参与二试"}`)
	jsonWrongPassword    = []byte(`{"code": 20001, "msg": "输入的密码错误，请您重试"}`)
	bytesContentTypeJSON = []byte("application/json")
	bytesContentTypeHTML = []byte("text/html")
	bytesContentTypePDF  = []byte("application/pdf")
	keys                 = []string{}
	localnet             = []byte{}
)

var indexPage = []byte(`
<!DOCTYPE html>
<html lang="en">

<head>
  <meta charset="UTF-8">
  <link rel="shortcut icon" href="https://www.microsoft.com/favicon.ico?v2">
  <title>SYSU-MSC-Studio</title>
  <style>
    body {
      background: #2d343d;
      font-family: "Arial"
    }
    .login {
      background: white;
      border: 1px solid #c4c4c4;
      margin: 20px auto;
      padding: 30px 25px;
      text-align: center;
      width: 28em;
    }
    h1.login-title {
      margin: -28px -25px 25px;
      padding: 15px 25px;
      line-height: 30px;
      font-size: 25px;
      font-weight: 300;
      color: #adadad;
      text-align: center;
      background: #f7f7f7;
    }
    .login-input {
      width: 285px;
      height: 50px;
      margin-bottom: 25px;
      padding-left: 10px;
      font-size: 15px;
      background: #fff;
      border: 1px solid #ccc;
      border-radius: 4px;
    }
    .login-button {
      width: 30%;
      height: 53px;
      padding: 0;
      font-size: 20px;
      color: #fff;
      text-align: center;
      background: #f0776c;
      border: 0;
      border-radius: 5px;
      cursor: pointer;
      outline: 0;
    }
    .notice {
      text-align:center;
      margin-bottom:0px;
    }
    .notice a{
      color:#666;
      text-decoration:none;
    }
  </style>
  <script>
    function onConfirmButtonPressed(password) {
      const apiUrl = window.location.origin + "/api?password=" + password
      fetch(apiUrl)
        .then(res => res.json())
        .then(res => {
          switch (res.code) {
            case 200:
              window.location.href = window.location.origin + "/" + password;
              break;
            case 20000:
            case 20001:
              alert(res.msg);
              break;
          }
        })
        .catch(err => console.log(err))
    }
  </script>
</head>

<body>
  <form class="login" action="javascript:onConfirmButtonPressed(password.value)">
    <h1 class="title">中山大学微软俱乐部二面试题</h1>
    <input type="password" id="password" class="login-input" placeholder="请输入获取题目的密码" required>
    <input type="submit" value="Let's Go!" class="login-button">
    <p class="notice">
      可点击 <a href="https://sysumsc.com">此处</a> 查看密码
    </p>
  </form>
</body>

</html>
`)

func bytes2string(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func isLocalIP(ip string) (bool, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false, err
	}
	for _, addr := range addrs {
		intf, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			return false, err
		}
		if net.ParseIP(ip).Equal(intf) {
			return true, nil
		}
	}
	return false, nil
}

func match(data string) bool {
	isMatched := false
	for _, key := range keys {
		if key == data {
			isMatched = true
			break
		}
	}
	return isMatched
}

func handler(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.SetContentTypeBytes(bytesContentTypeJSON)
	ctx.Response.Header.SetConnectionClose()

	// IP filtering
	remote := ctx.RemoteIP().To4()
	for i := 0; i < len(localnet); i++ {
		if remote[i] != localnet[i] {
			ctx.Write(jsonWrongIPNet)
			return
		}
	}

	path := bytes2string((ctx.Path())[1:])
	switch path {
	case "":
		ctx.Response.Header.SetContentTypeBytes(bytesContentTypeHTML)
		ctx.Write(indexPage)
	case "api":
		password := bytes2string(ctx.QueryArgs().Peek("password"))
		if match(password) {
			ctx.Write(jsonSuccess)
		} else {
			ctx.Write(jsonWrongPassword)
		}
	default:
		if match(path) {
			var builder strings.Builder
			builder.WriteString(*dir)
			builder.WriteString("/")
			builder.WriteString(path)
			builder.WriteString(".pdf")
			ctx.Response.Header.SetContentTypeBytes(bytesContentTypePDF)
			ctx.SendFile(builder.String())
		} else {
			ctx.Write(jsonWrongPassword)
		}
	}
}

func main() {
	flag.Parse()

	var err error

	// validate addr argument
	if *addr == "" {
		log.Fatalln("addr cannot be empty")
		return
	}
	if govalidator.IsDialString(*addr) != true {
		log.Fatalln("addr format error")
		return
	}
	h, _, err := net.SplitHostPort(*addr)
	if err != nil {
		log.Fatalln("unknown error")
		return
	}
	if isLocal, err := isLocalIP(h); isLocal != true {
		if err != nil {
			log.Fatalln("local network error")
		} else {
			log.Fatalln("addr must be local")
		}
		return
	}

	// validate dir argument
	if *dir == "" {
		log.Fatalln("dir cannot be empty")
		return
	}
	_, err = os.Stat(*dir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalln("dir cannot be found")
		} else {
			log.Fatalln("dir cannot be used for some reasons")
		}
		return
	}

	// validate ipnet argument
	if *ipnet != "" {
		if govalidator.IsIPv4(*ipnet) && govalidator.IsIPv4(*ipnet+".") != true && govalidator.IsIPv4(*ipnet+".1.") && govalidator.IsIPv4(*ipnet+".1.1.") {
			log.Fatalln("ipnet format error")
			return
		}
	}
	var arr []string
	if *ipnet != "" {
		arr = strings.Split(*ipnet, ".")
	}

	// establish rules for IP filtering
	for _, str := range arr {
		i, _ := strconv.Atoi(str)
		localnet = append(localnet, byte(i))
	}

	// establish files map
	files, _ := ioutil.ReadDir(*dir)
	for _, f := range files {
		if f.IsDir() == false {
			name := f.Name()
			if strings.HasSuffix(name, ".pdf") == true {
				keys = append(keys, strings.TrimSuffix(name, ".pdf"))
			}
		}
	}

	go func() {
		if err := fasthttp.ListenAndServe(*addr, handler); err != nil {
			log.Fatalf("error in ListenAndServe: %s", err)
		}
	}()

	log.Printf("Serving files on %q", *addr)
	log.Printf("Serving files from directory %q", *dir)

	select {}
}
