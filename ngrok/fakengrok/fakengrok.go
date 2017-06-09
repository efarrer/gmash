package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	VALID = `ngrok by @inconshreveable                                                                                                                                                                                                                                                                                                                                   (Ctrl+C to quit)

Session Status                online
Account                       Users Name (Plan: Free)
Version                       2.2.4
Region                        United States (us)
Web Interface                 http://127.0.0.1:4040
Forwarding                    tcp://0.tcp.ngrok.io:15120 -> localhost:22

Connections                   ttl     opn     rt1     rt5     p50     p90
                              0       0       0.00    0.00    0.00    0.00
`
	NOAUTH = `Tunnel session failed: TCP tunnels are only available after you sign up.
   Sign up at: https://ngrok.com/signup

   If you have already signed up, make sure your authtoken is installed.
   Your authtoken is available on your dashboard: https://dashboard.ngrok.com

   ERR_NGROK_302`
)

func getIntEnv(key string, def int) int {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	ivalue, _ := strconv.Atoi(value)
	return ivalue
}

func main() {
	delayMS := getIntEnv("DELAY_MS", 0)
	characters := getIntEnv("CHARACTERS", 20000)
	_type := os.Getenv("TYPE")
	hang := getIntEnv("HANG_HOURS", 250)

	var data string
	if _type == "VALID" {
		data = VALID
	} else if _type == "NOAUTH" {
		data = NOAUTH
	} else {
		data = _type
	}

	for len(data) > 0 {
		time.Sleep(time.Duration(delayMS) * time.Millisecond)
		if characters > len(data) {
			characters = len(data)
		}
		fmt.Printf(data[:characters])
		data = data[characters:]
	}

	time.Sleep(time.Duration(hang) * time.Hour)
}
