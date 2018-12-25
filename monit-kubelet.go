package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	mp "github.com/jiaxuanzhou/monit-kubelet/monit-pods"

	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	url         string
	timeout     int64
	h           bool
	cmd         string
	period      int64
	checkPeriod time.Duration = 10
)

func init() {
	flag.Int64Var(&timeout, "timeout", 10, "timeout in seconds to get response from the app")
	flag.StringVar(&url, "url", "http://127.0.0.1:10255/healthz", "endpoint to check the health status of the app")
	flag.BoolVar(&h, "h", false, "help")
	flag.StringVar(&cmd, "cmd", "systemctl restart kubelet", "actions if the app is not healthy for the specific duration")
	flag.Int64Var(&period, "period", 20, "period in minutes to check the health status of the app")
}

func main() {
	flag.Parse()
	if h || len(os.Args) < 2 {
		flag.Usage()
		os.Exit(0)
	}

	to := time.Duration(time.Second * time.Duration(timeout))
	hc := &http.Client{Timeout: to}
	go wait.Forever(func() {
		mp.LogUnhealthyPods(hc)
	}, time.Hour*1)

	for {
		time.Sleep(checkPeriod * time.Second)
		res, err := hc.Get(url)
		if err != nil || !checkRes(res) {
			log.Printf("[WARNING] kubelet is unhealthy, looping check!")
			loopCheck(hc)
		}
	}
}

func checkRes(res *http.Response) bool {
	if res.Body != nil {
		body, _ := ioutil.ReadAll(res.Body)
		if string(body) == "ok" {
			return true
		}
	}
	return false
}

func loopCheck(hc *http.Client) {
	c := make(chan bool, 1)
	loopOut := false
	go func() {
		select {
		case <-c:
			return
		case <-time.After(time.Minute * time.Duration(period)):
			out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
			if err != nil {
				log.Printf("[ERROR] err exeute the command, err:%v, output: %s", err, string(out))
			} else {
				log.Printf("[NORMAL] successfully excute the command %s", cmd)
			}
			loopOut = true
			return
		}
	}()

	for {
		if loopOut {
			break
		}
		time.Sleep(checkPeriod * time.Second)
		res, err := hc.Get(url)
		if err != nil || !checkRes(res) {
			log.Printf("[WARNING] loop checking: kubelet is unhealthy!")
			continue
		}
		log.Println("kubelet is back to healthy and break out")
		c <- true
		break
	}
}
