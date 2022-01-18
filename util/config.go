package util

import "github.com/spf13/viper"

type config struct {
	RsaHost    []rsaHost  `yaml:"rsahost"`
	PasswdHost []passHost `yaml:"passwdhost"`
}

type rsaHost struct {
	Host     string  `yaml:"host"`
	FilePath string  `yaml:"filepath"`
	User     string  `yaml:"user"`
	Port     int     `yaml:"port"`
}

type passHost struct {
	Host     string  `yaml:"host"`
	PassWd   string  `yaml:"passwd"`
	User     string  `yaml:"user"`
	Port     int     `yaml:"port"`
}

var Config = &config{}

func (this *config)Init(configFilePtr *string)  {
	this.loadConfig(configFilePtr)
}

func (this *config)loadConfig(configFilePtr *string)  {
	config:=viper.New()
	if configFilePtr == nil {
		config.SetConfigFile("./config.yaml")
	}else {
		config.SetConfigFile(*configFilePtr)
	}
	config.SetConfigType("yaml")
	if err:= config.ReadInConfig();err != nil {
		panic(err)
	}
	if err:=config.Unmarshal(&this);err != nil {
		panic(err)
	}
}

func (this *config)GetHostFromConfig() []*Cli {
	var clients []*Cli
	rsaHost := Config.RsaHost
	passwdHost := Config.PasswdHost

	if len(rsaHost) > 0 {
		for _, v := range rsaHost {
			cli, _ := New(v.Host, v.User, "", v.FilePath, v.Port)
			clients = append(clients, cli)
		}
	}

	if len(passwdHost) > 0 {
		for _, v := range passwdHost {
			cli, _ := New(v.Host, v.User, v.PassWd, "", v.Port)
			clients = append(clients, cli)
		}
	}
	return clients
}
