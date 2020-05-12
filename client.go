package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

type client struct {
	addr            string
	userName        string
	index           int
	conn            net.Conn
	auth            chan bool
	jobID           string
	extranonce2size int
	sessionID       string
	isconnected     int
}

var (
	userName     *string
	proxyAddress *string
	reconnect    *bool
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
	clientCount := flag.Int("n", 1, "the numberof client connections")
	userName = flag.String("u", "wany", "the name of miner")
	proxyAddress = flag.String("a", "", "dial address")
	debug := flag.Bool("d", false, "debug mode")
	reconnect = flag.Bool("r", false, "reconnect")
	timeDuration := flag.Int("t", 15, "time duration")
	flag.Parse()
	if *proxyAddress == "" {
		logrus.Error("proxyAddress error")
		return
	}
	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	timeStarted := time.Now()
	clientList := make([]*client, 0)
	for i := 0; i < *clientCount; i++ {
		c := NewClient(*proxyAddress, *userName, i)
		go c.Run()
		clientList = append(clientList, c)
	}
	var timeOver time.Duration
	var timeUsed time.Duration
	var sum int
	time.AfterFunc(time.Duration(*timeDuration)*time.Second, func() {
		timeUsed = time.Now().Sub(timeStarted)
		sum = 0
		for _, c := range clientList {
			sum += c.isconnected
		}
		if sum == len(clientList) && timeOver == time.Duration(0) {
			timeOver = timeUsed
		}
		logrus.Infof("timeused: %s, total: %d, connected: %d\n\n", timeUsed, *clientCount, sum)
	})
	c := make(chan os.Signal, 0)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
	<-c
	for _, c := range clientList {
		c.Close()
	}
	logrus.Infof("run time duration: %s, total: %d, connected: %d, time all connected: %s\n", timeUsed, *clientCount, sum, timeOver)

	<-c
}
func NewClient(addr string, userName string, index int) *client {
	return &client{
		addr:     addr,
		userName: userName,
		index:    index,
		auth:     make(chan bool),
	}
}
func (c *client) Run() {
	var err error
	wg := sync.WaitGroup{}
	for {
		c.conn, err = net.DialTimeout("tcp", c.addr, time.Second*3)
		if err != nil {
			logrus.Error(err)
			goto out
		}
		logrus.Info(c.index, " connected to ", c.addr)
		c.isconnected = 1
		wg.Add(2)
		go c.writeHandle(&wg)
		go c.readHandle(&wg)
		wg.Wait()

		c.isconnected = 0
		c.conn.Close()
		if *reconnect == false {
			return
		}
		continue
	out:
		logrus.Infof("retry ... after %s %d", time.Second*10, c.index)
		time.Sleep(time.Second * 3)
		continue
	}
}
func (c *client) writeHandle(wg *sync.WaitGroup) {
	defer wg.Done()
	subscribeStr := `{"id": 1, "method": "mining.subscribe", "params": ["__PoolWatcher__"]}`
	// err := c.conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
	// if err != nil {
	// 	logrus.Error(err, c.index)
	// 	return
	// }
	_, err := c.conn.Write([]byte(subscribeStr + "\n"))
	logrus.Debug("writebuf: ", subscribeStr)
	if err != nil {
		logrus.Error(err, c.index)
		return
	}
	<-c.auth
	authorizeStr := fmt.Sprintf("%s%s.%d%s", `{"id": 7, "method": "mining.authorize", "params": ["`, c.userName, c.index, `", "123"]}`)
	// err = c.conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
	// if err != nil {
	// 	logrus.Error(err, c.index)
	// 	return
	// }
	_, err = c.conn.Write([]byte(authorizeStr + "\n"))
	logrus.Debug("writebuf: ", authorizeStr)
	if err != nil {
		logrus.Error(err, c.index)
		return
	}
	extranonce2 := ""
	for {
		if c.extranonce2size == 4 {
			extranonce2 = "fe4353a1"
		} else {
			extranonce2 = "fe4353a1fe4353a1"
		}
		submitStr := fmt.Sprintf("{\"params\": [\"%s.%d\",\"%s\",\"%s\",\"%8x\",\"%s\"],\"id\":4,\"method\": \"mining.submit\"}", c.userName, c.index, c.jobID, extranonce2, time.Now().Unix(), "11ba3b08")
		// logrus.Info(submitStr)
		// err = c.conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
		// if err != nil {
		// 	logrus.Error(err)
		// 	break
		// }
		_, err := c.conn.Write([]byte(submitStr + "\n"))
		logrus.Debug("writebuf: ", submitStr)
		if err != nil {
			logrus.Error(err, c.index)
			break
		}
		time.Sleep(15 * time.Second)
	}
	logrus.Info("write handle close ", c.index)
}
func (c *client) readHandle(wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(c.conn)
	// err := c.conn.SetReadDeadline(time.Now().Add(time.Second * 10))
	// if err != nil {
	// 	logrus.Error(err)
	// 	return
	// }
	for scanner.Scan() {
		str := scanner.Text()
		logrus.Debug("readbuf: ", str)
		if strings.Contains(str, "client.reconnect") {
			c.addr = fmt.Sprintf("%s:%s", gjson.Get(str, "params.0").String(), gjson.Get(str, "params.1").String())
			logrus.Info(c.index, " change to ", c.addr)
			break
		}
		if strings.Contains(str, "mining.set_difficulty") && strings.Contains(str, "mining.notify") {
			arr := gjson.Get(str, "result").Array()
			c.sessionID = arr[len(arr)-2].String()
			c.extranonce2size = int(arr[len(arr)-1].Int())
			c.auth <- true
		} else if strings.Contains(str, "mining.notify") {
			arr := gjson.Get(str, "params").Array()
			c.jobID = arr[0].String()
		}
		// err := c.conn.SetReadDeadline(time.Now().Add(time.Second * 30))
		// if err != nil {
		// 	logrus.Error(err)
		// 	break
		// }
	}
	logrus.Info("read handle close ", c.index)
}

func (c *client) Close() {
	c.conn.Close()
}
