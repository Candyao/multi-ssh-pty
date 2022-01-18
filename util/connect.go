package util

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gookit/color"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"sync"
	"time"
)

type Cli struct {
	client  *ssh.Client
	Session *ssh.Session
	Stdout  *bytes.Buffer
	Stdin   io.WriteCloser
	LastLen int
	Color   color.Color256
	WriteAble bool
	*Term
	*ConnMsg
	*MsgChan
}

type ConnMsg struct {
	IP       string
	Username string
	Password string
	Port     int
	rsaFile  string
}

type Term struct {
	TermWidth  int
	TermHeight int
	HasPty     bool
}

type MsgChan struct {
	In  chan string
	Out chan string
}

func New(ip string, username string, password, rsaFile string, port int) (*Cli, error) {
	cli := new(Cli)
	if err := cli.InitConnMsg(ip, username, password, rsaFile, port); err != nil {
		return nil, err
	}
	if err := cli.InitMsgChan(); err != nil {
		return nil, err
	}
	cli.Stdout = bytes.NewBuffer(make([]byte, 0))
	cli.MsgChan = &MsgChan{
		In:  make(chan string, 1),
		Out: make(chan string, 0),
	}
	cli.Color = color.C256(RandColor(), false)
	cli.Term = new(Term)
	cli.HasPty = false
	cli.WriteAble=true
	return cli, nil

}

func (c *Cli) InitConnMsg(ip string, username string, password, rsaFile string, port int) error {
	connMsg := new(ConnMsg)
	connMsg.IP = ip
	connMsg.Username = username
	connMsg.Password = password
	connMsg.rsaFile = rsaFile
	if port == 0 {
		connMsg.Port = 22
	} else {
		connMsg.Port = port
	}
	c.ConnMsg = connMsg
	return nil
}

func (c *Cli) InitMsgChan() error {
	msgChan := new(MsgChan)
	msgChan.In = make(chan string)
	msgChan.Out = make(chan string)
	c.MsgChan = msgChan
	return nil
}

func (c *Cli) connection(ctx context.Context, wd bool) error {
	var err error
	var sshClient *ssh.Client
	var ch = make(chan struct{})
	var auth []ssh.AuthMethod
	if !wd {
		privateKeyBytes, err := ioutil.ReadFile(c.rsaFile)
		if err != nil {
			return err
		}
		key, err := ssh.ParsePrivateKey(privateKeyBytes)
		if err != nil {
			return err
		}
		auth = []ssh.AuthMethod{ssh.PublicKeys(key)}
	} else {
		auth = []ssh.AuthMethod{ssh.Password(c.Password)}
	}
	config := &ssh.ClientConfig{
		User: c.Username,
		Auth: auth,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 10 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", c.IP, c.Port)
	go func() {
		sshClient, err = ssh.Dial("tcp", addr, config)
		if err != nil {
			ch <- struct{}{}
		} else {
			c.client = sshClient
			ch <- struct{}{}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
		return err
	}
}

func (c *Cli) InitSession(ctx context.Context) error {
	var err error
	var IsPassword = false
	if c.Password != "" {
		IsPassword = true
	}
	if err = c.connection(ctx, IsPassword); err != nil {
		return err
	}
	if c.Session, err = c.client.NewSession(); err != nil {
		return err
	}
	stdin, err := c.Session.StdinPipe()
	if err != nil {
		return err
	}
	c.Stdin = stdin
	c.Session.Stdout = c.Stdout
	return nil
}

func (c *Cli) InitTerminal(fd int) error {
	var err error
	c.Term.TermWidth, c.Term.TermHeight, err = terminal.GetSize(fd)
	if err != nil {
		return err
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4k
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4k
	}
	if !c.HasPty {
		err := c.Session.RequestPty("xterm-256color", c.TermHeight, c.TermWidth, modes)
		if err != nil {
			return err
		}
		if err := c.Session.Shell(); err != nil {
			return err
		}
		c.HasPty = true
	}
	ch := make(chan struct{}, 0)
	go func() {
		_ = c.Session.Wait()
	}()

	go func() {
		_ = ClearBuf(c.Stdout, ch, GetEndByte(c.Username))
	}()
	<-ch
	return nil
}

func (c *Cli) getResMsg() {
	var msg string
	var timer=time.NewTimer(time.Millisecond)
	var Mu sync.Mutex
	buf := make([]byte, 1024)
	<-timer.C
	timer.Reset(time.Second*2)
	BufWait(c.Stdout)
	for len([]byte(msg)) < MaxSize {
		n, err := c.Stdout.Read(buf)
		if err != nil && err != io.EOF {
			break
		}
		if n < 1024 {
			msg += string(buf[:n])
			index := strings.LastIndexByte(msg, GetEndByte(c.Username))
			if index+2 >= n {
				break
			} else {
				BufWait(c.Stdout)
				select {
				case <-timer.C:
					Mu.Lock()
					TermOutPut(msg,c.IP,c.Color)
					Mu.Unlock()
					c.WriteAble=false
					timer.Reset(time.Millisecond*10)
				default:
					continue
				}
			}
		} else {
			msg += string(buf)
		}
	}
	c.Out <- msg
	return
}

func (c *Cli) RunCMD(shell string,isctrl bool) string {
	var wg sync.WaitGroup
	var msg string
	c.In <- shell
	wg.Add(1)
	go func() {
		shell := <-c.In
		_, _ = c.Stdin.Write([]byte(fmt.Sprintf("%s\r", shell)))
		defer wg.Done()
	}()
	wg.Wait()
	go func() {
		c.getResMsg()
	}()
	if !isctrl{
		msg = <-c.Out
	}
	return msg
}
