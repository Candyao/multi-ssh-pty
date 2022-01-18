package main

import (
	"context"
	"flag"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"golysh/util"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"sync"
	"time"
)


func main() {
	var clients  []*util.Cli
	var err error
	var configPath = flag.String("f","./config.yaml","config file path")

	flag.Parse()
	util.Config.Init(configPath)
	util.InitTerm()
	fmt.Printf("\n%s CLI - version 1.0 \n\n", util.Prefix)

	clients=util.Config.GetHostFromConfig()
	util.Total = len(clients)

	ctx, cancle := context.WithTimeout(context.Background(), time.Second*10)
	defer cancle()

	var wg sync.WaitGroup
	var syn sync.Mutex

	wg.Add(util.Total)
	for _, v := range clients {
		cli := v
		go func() {
			defer wg.Done()
			err := cli.InitSession(ctx)
			if err != nil {
				fmt.Println(err)
				return
			}
			if err:=cli.InitTerminal(util.Fd);err != nil{
				fmt.Println(err)
				return
			}
			syn.Lock()
			util.Ready++
			syn.Unlock()
		}()
		time.Sleep(time.Millisecond * 200)
		fmt.Printf("\r%s(%d/%d)>", util.Wait, util.Ready, util.Total)
	}
	wg.Wait()

	prompt := util.GetPrompt(fmt.Sprintf("\r%s(%d/%d)>", util.Wait, util.Ready, util.Total))
	tem := terminal.NewTerminal(os.Stdin, prompt)

	go func() {
		http.ListenAndServe("0.0.0.0:8081", nil)
	}()

	for {
		var ch=make(chan struct{})
		var cmdline string
		cmdline, err = tem.ReadLine()
		if err != nil {
			break
		}
		if cmdline == ""{
			continue
		}
		cmdline = strings.TrimSpace(cmdline)
		util.SetOldState()
		go func() {
			util.SendCtrl(clients,ch,true)
		}()
		util.ClintRunCmd(clients,cmdline,false)
		close(ch)
		util.SetRowState()
	}
	util.SetOldState()
	fmt.Println("")
}