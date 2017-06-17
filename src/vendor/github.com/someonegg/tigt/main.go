// Copyright 2016 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"
)

type strings []string

func (s strings) String() string {
	if len(s) == 0 {
		return ""
	}
	return fmt.Sprintf("%v", s)
}

func (s *strings) Set(v string) error {
	if len(v) > 0 {
		*s = append(*s, v)
	}
	return nil
}

var TmplF string
var ParamS strings

var USAGE = `Usage of tigt:
-p path
    	the parameter file, one or more
    	multi parameter will be merged to one
-t path
    	the go-template file
`

func parseConfig() bool {
	flag.StringVar(&TmplF, "t", "", "the go-template file")
	flag.Var(&ParamS, "p", "the parameter file, one or more")
	flag.Parse()
	return len(TmplF) > 0 && len(ParamS) > 0
}

func loadParam(path string) (map[string]interface{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	blob, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var param map[string]interface{}
	err = json.Unmarshal(blob, &param)
	if err != nil {
		return nil, err
	}

	return param, nil
}

func mergeObj(cur, inc map[string]interface{}) {
	for k, iv := range inc {
		if cv, exist := cur[k]; !exist {
			cur[k] = iv
		} else {
			cvo, cvok := cv.(map[string]interface{})
			ivo, ivok := iv.(map[string]interface{})
			if cvok && ivok {
				mergeObj(cvo, ivo)
			} else {
				cur[k] = iv
			}
		}
	}
}

func main() {
	if !parseConfig() {
		fmt.Fprintf(os.Stderr, USAGE)
		os.Exit(1)
	}

	tmpl, err := template.ParseFiles(TmplF)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse template file failed : %v\n", err)
		os.Exit(1)
	}
	tmpl.Option("missingkey=error")

	param := make(map[string]interface{})
	for _, paramF := range ParamS {
		p, err := loadParam(paramF)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Parse parameter file failed : %v,%v\n", paramF, err)
			os.Exit(1)
		}
		mergeObj(param, p)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, param)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Execute template failed : %v\n", err)
		os.Exit(1)
	}

	fmt.Fprint(os.Stdout, buf.String())
	return
}
