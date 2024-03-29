package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

type ConfigInfo struct {
	CmdLine     []string
	Params      []ParamInfo
	EndPoints   []EndPointInfo
	Connections []ConnectInfo
}

type ParamInfo struct {
	Name, Desc string
}

type EndPointInfo struct {
	Name, Kind, Config, Depends string
}

type ConnectInfo struct {
	From, To string
	Filters  []FilterInfo
}

type FilterInfo struct {
	Name, Config string
}

var (
	epRegexp     = regexp.MustCompile(`\s*([\w]+)\s*=\s*([\w]+)\s*\((.*)\)(\s*depends\s+([\w]+))*`)
	filterRegexp = regexp.MustCompile(`\s*([\w]+)\((.*)\)\s*`)
	paramRegexp  = regexp.MustCompile(`\s*([\w]+)\s+(.+)`)
)

func processParam(config *ConfigInfo, line string) error {
	parts := paramRegexp.FindStringSubmatch(line)
	if len(parts) != 3 {
		return fmt.Errorf("Invalid Param Definition: %v" + line)
	}

	pinfo := ParamInfo{parts[1], parts[2]}
	config.Params = append(config.Params, pinfo)

	return nil

}

func locateParamIndex(config *ConfigInfo, paramName string) (int, error) {
	for n := 0; n < len(config.Params); n++ {
		if paramName == config.Params[n].Name {
			return n, nil
		}
	}

	return 0, fmt.Errorf("param: %v  not Found", paramName)
}

func replaceParams(config *ConfigInfo, line string) (string, error) {
	for n := 0; n < len(config.Params); n++ {
		if len(config.CmdLine) <= n {
			return line, fmt.Errorf("no value provided for param: %v", config.Params[n].Name)
		}

		line = strings.Replace(line, "[$"+config.Params[n].Name+"$]", config.CmdLine[n], -1)
	}

	return line, nil
}

func processEndPoint(config *ConfigInfo, line string) error {

	var err error
	line, err = replaceParams(config, line)
	if err != nil {
		return err
	}

	parts := epRegexp.FindStringSubmatch(line)
	if len(parts) == 4 {
		config.EndPoints = append(config.EndPoints, EndPointInfo{parts[1], parts[2], parts[3], ""})
		return nil
	}

	if len(parts) == 6 {
		config.EndPoints = append(config.EndPoints, EndPointInfo{parts[1], parts[2], parts[3], parts[5]})
		return nil
	}

	return fmt.Errorf("invalid Endpoint Definition: %v", line)
}

func processConnect(config *ConfigInfo, line string) error {

	var err error
	line, err = replaceParams(config, line)
	if err != nil {
		return err
	}

	parts := strings.Split(line, "->")

	if len(parts) < 2 {
		return fmt.Errorf("invalid Connection: %v", line)
	}

	var ci ConnectInfo
	ci.From = strings.TrimSpace(parts[0])
	ci.To = strings.TrimSpace(parts[len(parts)-1])

	for i := 1; i < (len(parts) - 1); i++ {
		filter := filterRegexp.FindStringSubmatch(parts[i])
		if len(filter) == 3 {
			ci.Filters = append(ci.Filters, FilterInfo{filter[1], filter[2]})
		} else {
			return fmt.Errorf("invalid Filter: %v", parts[i])
		}
	}

	config.Connections = append(config.Connections, ci)

	return nil
}

func printHelp(w io.Writer, config *ConfigInfo) {
	if len(config.Params) > 0 {
		fmt.Println(" Param Usage:")
		for n := 0; n < len(config.Params); n++ {
			fmt.Printf("   %10s: %s\n", config.Params[n].Name, config.Params[n].Desc)
		}
	}
}

func ReadConfig(fname string) (*ConfigInfo, error) {

	var config ConfigInfo
	lineCount := 0

	if len(os.Args) > 2 {
		config.CmdLine = append(config.CmdLine, os.Args[2:]...)
	}

	err := ReadFile(fname, func(l string) error {
		lineCount++
		line := strings.TrimSpace(l)

		if len(line) == 0 {
			return nil
		}

		var err error
		var ok bool

		if strings.HasPrefix(line, "#") {
			// # is a comment ignore line
		} else if line, ok = TryTrimPrefix(line, "endpoint"); ok {
			err = processEndPoint(&config, line)
		} else if line, ok = TryTrimPrefix(line, "connect"); ok {
			err = processConnect(&config, line)
		} else if line, ok = TryTrimPrefix(line, "param"); ok {
			err = processParam(&config, line)
		} else {
			err = fmt.Errorf("unrecognised Line: [%v]", l)
		}
		return err
	})

	if err != nil {
		err = fmt.Errorf("error On Line: %d :%w", lineCount, err)
	}

	printHelp(os.Stdout, &config)

	//	fmt.Println(config.Params)
	//	fmt.Println(config.CmdLine)

	return &config, err
}
