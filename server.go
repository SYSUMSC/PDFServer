package main

import (
	"flag"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"unsafe"

	"github.com/valyala/fasthttp"
)

var (
	addr              = flag.String("addr", "192.168.1.102:8080", "TCP address to listen to")
	dir               = flag.String("dir", "./", "Directory to serve static files from")
	ipnet             = flag.String("ipnet", "192.168.1", "IPNet to allow access from")
	jsonSuccess       = []byte(`{"code": 200}`)
	jsonWrongIPNet    = []byte(`{"code": 20000, "msg": "不在同一网段，请您到指定地点参与二试"}`)
	jsonWrongPassword = []byte(`{"code": 20001, "msg": "输入的密码错误，请您重试"}`)
	keys              = []string{}
	localnet          = []byte{}
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
	ctx.Response.Header.SetContentType("application/json")
	ctx.Response.Header.SetConnectionClose()

	// IP filtering
	remote := ctx.RemoteIP().To4()
	for i := 0; i < 3; i++ {
		if remote[i] != localnet[i] {
			ctx.Write(jsonWrongIPNet)
			return
		}
	}

	path := string((ctx.Path())[1:])
	switch path {
	case "":
		ctx.Response.Header.SetContentType("text/html")
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
			ctx.SendFile(*dir + "/" + path + ".pdf")
		} else {
			ctx.Write(jsonWrongPassword)
		}
	}
}

func main() {
	flag.Parse()

	// validate addr argument
	if *addr == "" {
		log.Fatalln("addr cannot be empty")
		return
	}

	// validate dir argument
	if *dir == "" {
		log.Fatalln("dir cannot be empty")
		return
	}

	// validate ipnet argument
	if *ipnet == "" {
		log.Fatalln("ipnet cannot be empty")
		return
	}
	arr := strings.Split(*ipnet, ".")
	if len(arr) != 3 {
		log.Fatalln("ipnet format error")
		return
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
			keys = append(keys, strings.TrimSuffix(f.Name(), ".pdf"))
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
