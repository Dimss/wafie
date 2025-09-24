package plugin

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	cniv1 "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ns"
)

var logger = log.New(os.Stderr, "[wafie-cni] ", log.LstdFlags)

type Config struct {
	types.NetConf
	MyAwesomeFlag     bool   `json:"myAwesomeFlag"`
	AnotherAwesomeArg string `json:"anotherAwesomeArg"`
}

func parseConfig(stdin []byte) (*Config, error) {
	conf := Config{}
	if err := json.Unmarshal(stdin, &conf); err != nil {
		return nil, fmt.Errorf("failed to parse network configuration: %v", err)
	}
	if err := version.ParsePrevResult(&conf.NetConf); err != nil {
		return nil, fmt.Errorf("could not parse prevResult: %v", err)
	}
	if conf.AnotherAwesomeArg == "" {
		return nil, fmt.Errorf("anotherAwesomeArg must be specified")
	}
	logger.Println("CMD PARSE CONFIG CALLED")
	return &conf, nil
}

func CmdAdd(args *skel.CmdArgs) (err error) {
	conf, err := parseConfig(args.StdinData)
	if err != nil {
		return err
	}
	result := &cniv1.Result{}
	if conf.PrevResult != nil {
		prevResult, err := cniv1.GetResult(conf.PrevResult)
		if err != nil {
			return fmt.Errorf("failed to convert prevResult: %v", err)
		}
		logger.Println(result.IPs)
		result = prevResult
	} else {
		result = &cniv1.Result{
			CNIVersion: cniv1.ImplementedSpecVersion,
		}
	}
	if err := StartRelay(args.Netns); err != nil {
		logger.Printf("failed to start relay: %v", err)
	}
	logger.Println("Network namespace " + args.Netns)
	logger.Println("Ifname " + args.IfName)
	k8sArgs := parseCNIArgs(args.Args)
	logger.Printf("Pod: %s/%s\n", k8sArgs["K8S_POD_NAMESPACE"], k8sArgs["K8S_POD_NAME"])

	return types.PrintResult(result, conf.CNIVersion)
}

func CmdDel(args *skel.CmdArgs) (err error) {
	return nil
}

func CmdCheck(args *skel.CmdArgs) (err error) {
	return nil
}

func StartRelay(netnsPath string) error {
	targetNs, err := ns.GetNS(netnsPath)
	if err != nil {
		logger.Println("error getting netns", netnsPath)
		return err
	}

	return targetNs.Do(func(ns.NetNS) error {
		cmd := exec.Command("/opt/cni/bin/wafie-relay")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		return cmd.Start()
	})
}

func parseCNIArgs(args string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(args, ";")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}
	return result
}
