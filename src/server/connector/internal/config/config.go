// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"

	. "common/config"
)

var conf string

func init() {
	const (
		defaultConf = "connector.conf"
		usage       = "the config file"
	)
	flag.StringVar(&conf, "conf", defaultConf, usage)
	flag.StringVar(&conf, "c", defaultConf, usage+" (shorthand)")
}

type ConnectSvcConfT struct {
	BDMsgSvcConfT
}

func (c *ConnectSvcConfT) Check() bool {
	return c.BDMsgSvcConfT.Check()
}

type ServiceSConfT struct {
	Debug   ServiceConfT    	`json:"debug"`
	Connect ConnectSvcConfT 	`json:"connect"`
}

type Mq struct {
	KafkaBrokers[] string 		`json:"kafkabrokers"`
	Topic string				`json:"topic"`
	ComputeBrokers[] string 	`json:computebrokers`
}

type Redis struct {
	MasterName string			`json:"masterName"`
	Addresses []string  		`json:"addresses"`
	Password  string    		`json:"password"`
	Db        int       		`json:"db"`
}

type RedisPubSub struct {
	Address string 				`json:"address"`
	Password string   			`json:"password"`
}

func (c *ServiceSConfT) Check() bool {
	// debug maybe unset.
	return c.Connect.Check()
}

type ManagerConfT struct {
}

func (c *ManagerConfT) Check() bool {
	return true
}

type ConfigT struct {
	Pidfile   string        	`json:"pidfile"`
	InfoFile  string        	`json:"logfile"`
	ErrorFile string        	`json:"logerrorfile"`
	ServiceS  ServiceSConfT 	`json:"service"`
	Manager   ManagerConfT  	`json:"manager"`
	Mq        Mq            	`json:"mq"`
	Redis	  Redis				`json:"redis"`
	RedisPubSub RedisPubSub		`jons:"redisPubSub"`
	IpResolver []string			`json:"ipResolver"`
}

func (c *ConfigT) Check() bool {
	return len(c.Pidfile) > 0 && len(c.InfoFile) > 0 && c.ServiceS.Check() && c.Manager.Check()
}

var Config *ConfigT = &ConfigT{}

func ParseConfig() error {
	flag.Parse()

	f, err := os.Open(conf)
	if err != nil {
		return err
	}
	defer f.Close()

	blob, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	err = json.Unmarshal(blob, Config)
	if err != nil {
		return err
	}

	if !Config.Check() {
		return ErrConfigContent
	}

	return nil
}
