package util

import (
	"bytes"
	"fmt"
	"github.com/gookit/color"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	Prefix  string = "GOLYSH"
	Wait    string = "Waiting"
	MaxSize int    = 2 << 20
)

var (
	Ready        int
	Total        int
	OldTermState *terminal.State
	RawState     *terminal.State
	Fd           int
)
var CtrlMsg = map[string]string{"c": "\x03", "z": "\x1A"}

func GetPrompt(s string) string {
	var waitLen = len([]byte(s))
	h := strings.Repeat(" ", waitLen)
	fmt.Printf("\r%s", h)
	prompt := fmt.Sprintf("\r%s(%d/%d)>", Prefix, Ready, Total)
	return prompt
}

func ClearBuf(out *bytes.Buffer, ch chan struct{}, c byte) error {
	buf := make([]byte, 1024)
	for {
		_, err := out.Read(buf)
		if err == io.EOF {
			time.Sleep(time.Millisecond * 10)
		}
		if err != nil && err != io.EOF {
			break
		}
		endIndex := bytes.LastIndexByte(buf, c)
		if endIndex > 0 {
			break
		}
	}
	ch <- struct{}{}
	return nil
}

func SendCtrl(c []*Cli,ch chan  struct{},isctrl bool){
	sig:=make(chan os.Signal)
	signal.Notify(sig,syscall.SIGINT,syscall.SIGTSTP)
	var cmdline string
	select {
	case val:=<-sig:
		switch val {
		case syscall.SIGINT:
			cmdline=CtrlMsg["c"]
		case syscall.SIGSTOP:
			cmdline=CtrlMsg["z"]
		}
		ClintRunCmd(c,cmdline,isctrl)
	case <-ch:
	}
}

func ClintRunCmd(c []*Cli,cmdline string,isctrl bool)  {
	var wg sync.WaitGroup
	var syn sync.Mutex
	for _,v:=range c{
		cli:=v
		if cli.HasPty{
			wg.Add(1)
			go func() {
				res:=cli.RunCMD(cmdline,isctrl)
				if isctrl {
					cli.WriteAble=true
				}
				syn.Lock()
				if cli.WriteAble {
					TermOutPut(res,cli.IP,cli.Color)
				}
				syn.Unlock()
				defer wg.Done()
			}()
			cli.WriteAble=true
		}
	}
	wg.Wait()
	return
}

func RandColor() uint8 {
	rand.Seed(time.Now().UnixNano() / 256)
	i := rand.Intn(256)
	return uint8(i)
}

func GetEndByte(s string) byte {
	if s == "root" {
		return '#'
	}
	return '$'
}

func BufWait(buf *bytes.Buffer) {
	var timer = time.NewTimer(time.Millisecond)
	var nowReader []byte
	<-timer.C
	timer.Reset(time.Millisecond * 100)
	for {
		lastReader := buf.Bytes()
		if len(lastReader) == 0 {
			timer.Reset(time.Millisecond * 100)
		}
		<-timer.C
		nowReader = buf.Bytes()
		if len(nowReader) == 0 {
			continue
		}
		if len(nowReader) == len(lastReader) {
			break
		} else {
			timer.Reset(time.Millisecond * 100)
			continue
		}
	}
}

func TermOutPut(s, ip string, c color.Color256) {
	resList := strings.Split(s, "\r")
	lenList := len(resList)
	for i := 0; i < lenList-1; i++ {
		if resList[i] != "" {
			last := strings.ReplaceAll(resList[i], "\n", "")
			fmt.Printf("%s%s\n", c.Sprintf("%s : ", ip), last)
		}
	}
}

func InitTerm()  {
	var err error
	//fmt.Printf("\n%s CLI - version 1.0 \n\n", Prefix)
	Fd = int(os.Stdin.Fd())
	OldTermState, err = terminal.MakeRaw(Fd)
	if err != nil {
		fmt.Println(err)
		return
	}
	RawState, err = terminal.GetState(Fd)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func SetRowState()  {
	_=terminal.Restore(Fd,RawState)
	return
}

func SetOldState()  {
	_=terminal.Restore(Fd,OldTermState)
	return
}