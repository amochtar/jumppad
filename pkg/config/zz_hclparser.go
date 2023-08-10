package config

import (
	"os"

	"github.com/jumppad-labs/hclconfig"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// registeredTypes is a static list of types that can be used by the parser
// it is the responsibility of the type to register itself with the parser
var registeredTypes map[string]types.Resource

// registeredProvider is a static list of providers that can be used by the parser
// it is the responsibility of the type to register itself with the parser
var registeredProviders map[string]Provider

func init() {
	registeredTypes = map[string]types.Resource{}
	registeredProviders = map[string]Provider{}
}

// RegisterResource allows a resource to register itself with the parser
func RegisterResource(name string, r types.Resource, p Provider) {
	if r != nil {
		registeredTypes[name] = r
	}

	if p != nil {
		registeredProviders[name] = p
	}
}

// setupHCLConfig configures the HCLConfig package and registers the custom types
func NewParser(callback hclconfig.ProcessCallback, variables map[string]string, variablesFiles []string) *hclconfig.Parser {
	cfg := hclconfig.DefaultOptions()
	cfg.ParseCallback = callback
	cfg.Variables = variables
	cfg.VariablesFiles = variablesFiles

	p := hclconfig.NewParser(cfg)

	// Register the types
	for k, v := range registeredTypes {
		p.RegisterType(k, v)
	}

	// Register the custom functions
	p.RegisterFunction("jumppad", customHCLFuncJumppad)
	p.RegisterFunction("docker_ip", customHCLFuncDockerIP)
	p.RegisterFunction("docker_host", customHCLFuncDockerHost)
	p.RegisterFunction("data", customHCLFuncDataFolder)
	p.RegisterFunction("data_with_permissions", customHCLFuncDataFolderWithPermissions)

	return p
}

func customHCLFuncJumppad() (string, error) {
	return utils.JumppadHome(), nil
}

// returns the docker host ip address
func customHCLFuncDockerIP() (string, error) {
	return utils.GetDockerIP(), nil
}

func customHCLFuncDockerHost() (string, error) {
	return utils.GetDockerHost(), nil
}

func customHCLFuncDataFolderWithPermissions(name string, permissions int) (string, error) {
	perms := os.FileMode(permissions)
	return utils.GetDataFolder(name, perms), nil
}

func customHCLFuncDataFolder(name string) (string, error) {
	perms := os.FileMode(0775)
	return utils.GetDataFolder(name, perms), nil
}
