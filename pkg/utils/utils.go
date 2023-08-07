package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/mod/sumdb/dirhash"
)

// Creates the required file structure in the users Home directory
func CreateFolders() {
	os.MkdirAll(GetReleasesFolder(), os.FileMode(0755))
}

// ValidateName ensures that the name for a resource is within certain boundaries
// Valid characters: [a-z] [A-Z] _ - [0-9]
// Max length: 128
func ValidateName(name string) (bool, error) {
	// check the length
	if len(name) > 128 {
		return false, NameExceedsMaxLengthError
	}

	r := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	ok := r.MatchString(name)
	if !ok {
		return false, NameContainsInvalidCharactersError
	}

	return true, nil
}

// ReplaceNonURIChars replaces any characters in the resrouce name which
// can not be used in a URI
func ReplaceNonURIChars(s string) (string, error) {
	reg, err := regexp.Compile(`[^a-zA-Z0-9\-\.]+`)
	if err != nil {
		return "", err
	}

	return reg.ReplaceAllString(s, "-"), nil
}

// FQDN generates the full qualified name for a container
func FQDN(name, module, typeName string) string {
	fqdn := fmt.Sprintf("%s.%s.jumppad.dev", name, typeName)
	if module != "" {
		fqdn = fmt.Sprintf("%s.%s.%s.jumppad.dev", name, module, typeName)
	}

	// ensure that the name is valid for URI schema
	cleanName, err := ReplaceNonURIChars(fqdn)
	if err != nil {
		panic(err)
	}

	return cleanName
}

// FQDNVolumeName creates a full qualified volume name
func FQDNVolumeName(name string) string {
	// ensure that the name is valid for URI schema
	cleanName, err := ReplaceNonURIChars(name)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s.volume.jumppad.dev", cleanName)
}

// CreateKubeConfigPath creates the file path for the KubeConfig file when
// using Kubernetes cluster
func CreateKubeConfigPath(name string) (dir, filePath string, dockerPath string) {
	dir = filepath.Join(JumppadHome(), "/config/", name)
	filePath = filepath.Join(dir, "/kubeconfig.yaml")
	dockerPath = filepath.Join(dir, "/kubeconfig-docker.yaml")

	// create the folders
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		panic(err)
	}

	return
}

// HomeFolder returns the users homefolder this will be $HOME on windows and mac and
// USERPROFILE on windows
func HomeFolder() string {
	return os.Getenv(HomeEnvName())
}

// HomeEnvName returns the environment variable used to store the home path
func HomeEnvName() string {
	if runtime.GOOS == "windows" {
		return "USERPROFILE"
	}

	return "HOME"
}

// JumppadHome returns the location of the shipyard
// folder, usually $HOME/.shipyard
func JumppadHome() string {
	return filepath.Join(HomeFolder(), "/.jumppad")
}

// ShipyardTemp returns a temporary folder
func ShipyardTemp() string {
	dir := filepath.Join(JumppadHome(), "/tmp")
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		panic(err)
	}

	return dir
}

// StateDir returns the location of the shipyard
// state, usually $HOME/.shipyard/state
func StateDir() string {
	return filepath.Join(JumppadHome(), "/state")
}

// CertsDir returns the location of the certificates for the given resource
// used to secure the Shipyard ingress, usually rooted at $HOME/.shipyard/certs
func CertsDir(name string) string {
	certs := filepath.Join(JumppadHome(), "/certs", name)
	certs = filepath.FromSlash(certs)

	// create the folder if it does not exist
	os.MkdirAll(certs, os.ModePerm)
	return certs
}

// LogsDir returns the location of the logs
// used to secure the Shipyard ingress, usually $HOME/.shipyard/logs
func LogsDir() string {
	logs := filepath.Join(JumppadHome(), "/logs")

	os.MkdirAll(logs, os.ModePerm)
	return logs
}

// StatePath returns the full path for the state file
func StatePath() string {
	return filepath.Join(StateDir(), "/state.json")
}

// ImageCacheLog returns the location of the image cache log
func ImageCacheLog() string {
	return fmt.Sprintf("%s/images.log", JumppadHome())
}

// IsLocalFolder tests if the given path is a localfolder and can
// exist in the current filesystem
// TODO make more robust with error messages
// to improve UX
func IsLocalFolder(path string) bool {
	path, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	f, err := os.Stat(path)
	if err != nil || f == nil {
		return false
	}

	return true
}

// IsHCLFile tests if the given path resolves to a HCL config file
func IsHCLFile(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}

	if s.IsDir() {
		return false
	}

	if filepath.Ext(s.Name()) != ".hcl" {
		return false
	}

	return true
}

func sanitizeBlueprintFolder(blueprint string) string {
	blueprint = strings.ReplaceAll(blueprint, "//", "/")
	blueprint = strings.ReplaceAll(blueprint, "?", "/")
	blueprint = strings.ReplaceAll(blueprint, "&", "/")
	blueprint = strings.ReplaceAll(blueprint, "=", "/")

	return blueprint
}

// GetBlueprintFolder parses a blueprint uri and returns the top level
// blueprint folder
// if the URI is not a blueprint will return an error
func GetBlueprintFolder(blueprint string) (string, error) {
	// get the folder for the blueprint
	parts := strings.Split(blueprint, "//")

	if parts == nil || len(parts) != 2 {
		return "", InvalidBlueprintURIError
	}

	return sanitizeBlueprintFolder(parts[1]), nil
}

// GetBlueprintLocalFolder returns the full storage path
// for the given blueprint URI
func GetBlueprintLocalFolder(blueprint string) string {
	// we might have a querystring reference such has github.com/abc/cds?ref=dfdf&dfdf
	// replace these separators with /
	blueprint = sanitizeBlueprintFolder(blueprint)

	return filepath.Join(JumppadHome(), "blueprints", blueprint)
}

// GetHelmLocalFolder returns the full storage path
// for the given blueprint URI
func GetHelmLocalFolder(chart string) string {
	chart = sanitizeBlueprintFolder(chart)

	return filepath.Join(JumppadHome(), "helm_charts", chart)
}

// GetReleasesFolder return the path of the Shipyard releases
func GetReleasesFolder() string {
	return filepath.Join(JumppadHome(), "releases")
}

// GetDataFolder creates the data directory used by the application
func GetDataFolder(p string, perms os.FileMode) string {
	data := filepath.Join(JumppadHome(), "data", p)

	// create the folder if it does not exist
	os.MkdirAll(data, perms)
	os.Chmod(data, perms)

	return data
}

// GetLibraryFolder creates the library directory used by the application
func GetLibraryFolder(p string, perms os.FileMode) string {
	data := filepath.Join(JumppadHome(), "library", p)

	// create the folder if it does not exist
	os.MkdirAll(data, perms)
	os.Chmod(data, perms)

	return data
}

// GetDockerHost returns the location of the Docker API depending on the platform
func GetDockerHost() string {
	if dh := os.Getenv("DOCKER_HOST"); dh != "" {
		return dh
	}

	return "/var/run/docker.sock"
}

// GetDockerIP returns the location of the Docker Server IP address
func GetDockerIP() string {
	if dh := os.Getenv("DOCKER_HOST"); dh != "" {
		if strings.HasPrefix(dh, "tcp://") {
			u, err := url.Parse(dh)
			if err == nil {
				host := strings.Split(u.Host, ":")[0]
				ip, err := net.LookupHost(host)
				if err == nil && len(ip) > 0 {
					return ip[0]
				}
			}
		}
	}

	sp, _ := GetLocalIPAndHostname()

	return sp
}

// GetConnectorPIDFile returns the connector PID file used by the connector
func GetConnectorPIDFile() string {
	return filepath.Join(JumppadHome(), "connector.pid")
}

// GetConnectorLogFile returns the log file used by the connector
func GetConnectorLogFile() string {
	return filepath.Join(LogsDir(), "connector.log")
}

func compileShipyardBinary(path string) error {
	maxLevels := 10
	currentLevel := 0

	// we are running from a test so compile the binary
	// and returns its path
	dir, _ := os.Getwd()

	// walk backwards until we find the go.mod
	for {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return err
		}

		for _, f := range files {
			if strings.HasSuffix(f.Name(), "go.mod") {
				fp, _ := filepath.Abs(dir)

				// found the project root
				file := filepath.Join(fp, "main.go")
				tmpBinary := path

				// if windows append the exe extension
				if runtime.GOOS == "windows" {
					tmpBinary = tmpBinary + ".exe"
				}

				os.RemoveAll(tmpBinary)

				outwriter := bytes.NewBufferString("")
				cmd := exec.Command("go", "build", "-o", tmpBinary, file)
				cmd.Stderr = outwriter
				cmd.Stdout = outwriter

				err := cmd.Run()
				if err != nil {
					fmt.Println("Error building temporary binary:", cmd.Args)
					fmt.Println(outwriter.String())
					panic(fmt.Errorf("unable to build connector binary: %s", err))
				}

				return nil
			}
		}

		// check the parent
		dir = filepath.Join(dir, "../")
		fmt.Println(dir)
		currentLevel++
		if currentLevel > maxLevels {
			panic("unable to find go.mod")
		}
	}
}

var buildSync = sync.Once{}

// GetShipyardBinaryPath returns the path to the running Shipyard binary
func GetShipyardBinaryPath() string {
	if strings.HasSuffix(os.Args[0], "jumppad") || strings.HasSuffix(os.Args[0], "jumppad-dev") || strings.HasSuffix(os.Args[0], "jumppad.exe") || strings.HasSuffix(os.Args[0], "jp") {
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}

		return ex
	}

	tmpBinary := filepath.Join(os.TempDir(), "jumppad-dev")
	buildSync.Do(func() {
		compileShipyardBinary(tmpBinary)
	})

	return tmpBinary
}

// GetHostname returns the hostname for the current machine
func GetHostname() string {
	hn, err := os.Hostname()
	if err != nil {
		return ""
	}

	return hn
}

// GetLocalIPAddress returns a list of ip addressses for the local machine
func GetLocalIPAddresses() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return []string{}
	}

	addresses := []string{}
	for _, a := range addrs {
		ip, _, err := net.ParseCIDR(a.String())
		if err == nil {
			addresses = append(addresses, fmt.Sprintf("%s", ip))
		}
	}

	return addresses
}

// GetLocalIPAndHostname returns the IP Address of the machine
func GetLocalIPAndHostname() (string, string) {
	netInterfaceAddresses, err := net.InterfaceAddrs()
	if err != nil {
		return "", ""
	}

	for _, netInterfaceAddress := range netInterfaceAddresses {
		networkIP, ok := netInterfaceAddress.(*net.IPNet)
		if ok && !networkIP.IP.IsLoopback() && networkIP.IP.To4() != nil {
			ip := networkIP.IP.String()
			return ip, GetHostname()
		}
	}

	return "127.0.0.1", "localhost"
}

// HTTPProxyAddress returns the default HTTPProxy used by
// Nomad and Kubernetes clusters unless the environment variable
// HTTP_PROXY is set when it returns this value
func HTTPProxyAddress() string {
	if p := os.Getenv("HTTP_PROXY"); p != "" {
		return p
	}

	return shipyardProxyAddress
}

// HTTPSProxyAddress returns the default HTTPProxy used by
// Nomad and Kubernetes clusters unless the environment variable
// HTTPS_PROXY is set when it returns this value
func HTTPSProxyAddress() string {
	if p := os.Getenv("HTTPS_PROXY"); p != "" {
		return p
	}

	return shipyardProxyAddress
}

// get all ipaddresses in a subnet
func SubnetIPs(subnet string) ([]string, error) {
	_, ipnet, _ := net.ParseCIDR(subnet)

	var ipList []string
	ip := ipnet.IP
	for ; ipnet.Contains(ip); ip = incIP(ip) {
		ipList = append(ipList, ip.String())
	}

	return ipList, nil
}

// HashDir generates a hash of the given directory
func HashDir(dir string) (string, error) {
	return dirhash.HashDir(dir, "", dirhash.DefaultHash)
}

// HashFile returns a sha256 hash of the given file
func HashFile(file string) (string, error) {
	r, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer r.Close()

	hf := sha256.New()
	_, err = io.Copy(hf, r)

	if err != nil {
		return "", err
	}

	return "h1:" + base64.StdEncoding.EncodeToString(hf.Sum(nil)), nil
}

func incIP(ip net.IP) net.IP {
	// allocate a new IP
	newIp := make(net.IP, len(ip))
	copy(newIp, ip)

	byteIp := []byte(newIp)
	l := len(byteIp)
	var i int
	for k, _ := range byteIp {
		// start with the rightmost index first
		// increment it
		// if the index is < 256, then no overflow happened and we increment and break
		// else, continue to the next field in the byte
		i = l - 1 - k
		if byteIp[i] < 0xff {
			byteIp[i]++
			break
		} else {
			byteIp[i] = 0
		}
	}
	return net.IP(byteIp)
}
