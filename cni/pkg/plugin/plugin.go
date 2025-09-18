package plugin

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Dimss/wafie/cni/pkg/plugin/nftables"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
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
	if conf.PrevResult == nil {
		return fmt.Errorf("must be called as chained plugin")
	}

	prevResult, err := current.GetResult(conf.PrevResult)
	if err != nil {
		return fmt.Errorf("failed to convert prevResult: %v", err)
	}
	if len(prevResult.IPs) == 0 {
		return fmt.Errorf("got no container IPs")
	}
	result := prevResult
	logger.Println(result.IPs)
	//args.ContainerID
	logger.Println("Network namespace " + args.Netns)
	file, err := os.OpenFile(args.Netns, os.O_RDONLY, 0)
	if err != nil {
		logger.Printf("Error opening netns file %s: %v\n", args.Netns, err)
	}
	defer file.Close()
	// The file descriptor is an integer handle to the namespace.
	fd := file.Fd()
	logger.Printf("Successfully got file descriptor %d for namespace at %s\n", fd, args.Netns)
	if err := nftables.Program(args.Netns); err != nil {
		logger.Printf("Error executing nftables program: %v", err)
		os.Exit(1)
	}
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
