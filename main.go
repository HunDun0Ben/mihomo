package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	mihomocmd "github.com/metacubex/mihomo/cmd"
	"github.com/metacubex/mihomo/common/cmd"
	"github.com/metacubex/mihomo/component/age"
	"github.com/metacubex/mihomo/component/geodata"
	"github.com/metacubex/mihomo/component/updater"
	"github.com/metacubex/mihomo/config"
	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/hub"
	"github.com/metacubex/mihomo/hub/executor"
	"github.com/metacubex/mihomo/log"
	_ "go.uber.org/automaxprocs/maxprocs"
)

var (
	testConfig                    bool
	geodataMode                   bool
	homeDir                       string
	configFile                    string
	configString                  string
	configBytes                   []byte
	ageSecretKey                  string
	externalUI                    string
	externalController            string
	externalControllerTLS         string
	externalControllerUnix        string
	externalControllerPipe        string
	externalControllerRoutingMark int
	secret                        string
	postUp                        string
	postDown                      string
)

func main() {
	if err := executeCLI(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func executeCLI() error {
	return mihomocmd.Execute(func(options mihomocmd.Options) {
		testConfig = options.TestConfig
		geodataMode = options.GeodataMode
		homeDir = options.HomeDir
		configFile = options.ConfigFile
		configString = options.ConfigString
		ageSecretKey = options.AgeSecretKey
		externalUI = options.ExternalUI
		externalController = options.ExternalController
		externalControllerTLS = options.ExternalControllerTLS
		externalControllerUnix = options.ExternalControllerUnix
		externalControllerPipe = options.ExternalControllerPipe
		externalControllerRoutingMark = options.ExternalControllerRoutingMark
		secret = options.Secret
		postUp = options.PostUp
		postDown = options.PostDown
		runMihomo()
	})
}

func runMihomo() {
	// Defensive programming: panic when code mistakenly calls net.DefaultResolver
	net.DefaultResolver.PreferGo = true
	net.DefaultResolver.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		//panic("should never be called")
		buf := make([]byte, 1024)
		for {
			n := runtime.Stack(buf, true)
			if n < len(buf) {
				buf = buf[:n]
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		fmt.Fprintf(os.Stderr, "panic: should never be called\n\n%s", buf) // always print all goroutine stack
		os.Exit(2)
		return nil, nil
	}

	// _, _ = maxprocs.Set(maxprocs.Logger(func(string, ...any) {}))

	if homeDir != "" {
		if !filepath.IsAbs(homeDir) {
			currentDir, _ := os.Getwd()
			homeDir = filepath.Join(currentDir, homeDir)
		}
		C.SetHomeDir(homeDir)
	}

	if geodataMode {
		geodata.SetGeodataMode(true)
	}

	if ageSecretKey != "" {
		if err := age.VeritySecretKeys(ageSecretKey); err != nil {
			log.Errorln("Parse age-secret-key error: %s", err.Error())
		}
		age.SetGlobalSecretKeys(ageSecretKey)
	}

	if configString != "" {
		var err error
		configBytes, err = base64.StdEncoding.DecodeString(configString)
		if err != nil {
			log.Fatalln("Initial configuration error: %s", err.Error())
			return
		}
	} else if configFile == "-" {
		var err error
		configBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalln("Initial configuration error: %s", err.Error())
			return
		}
	} else {
		if configFile != "" {
			if !filepath.IsAbs(configFile) {
				currentDir, _ := os.Getwd()
				configFile = filepath.Join(currentDir, configFile)
			}
		} else {
			configFile = filepath.Join(C.Path.HomeDir(), C.Path.Config())
		}
		C.SetConfig(configFile)

		if err := config.Init(C.Path.HomeDir()); err != nil {
			log.Fatalln("Initial configuration directory error: %s", err.Error())
		}
	}

	if testConfig {
		if len(configBytes) != 0 {
			if _, err := executor.ParseWithBytes(configBytes); err != nil {
				log.Errorln(err.Error())
				fmt.Println("configuration test failed")
				os.Exit(1)
			}
		} else {
			if _, err := executor.Parse(); err != nil {
				log.Errorln(err.Error())
				fmt.Printf("configuration file %s test failed\n", C.Path.Config())
				os.Exit(1)
			}
		}
		fmt.Printf("configuration file %s test is successful\n", C.Path.Config())
		return
	}

	var options []hub.Option
	if externalUI != "" {
		options = append(options, hub.WithExternalUI(externalUI))
	}
	if externalController != "" {
		options = append(options, hub.WithExternalController(externalController))
	}
	if externalControllerTLS != "" {
		options = append(options, hub.WithExternalControllerTLS(externalControllerTLS))
	}
	if externalControllerUnix != "" {
		options = append(options, hub.WithExternalControllerUnix(externalControllerUnix))
	}
	if externalControllerPipe != "" {
		options = append(options, hub.WithExternalControllerPipe(externalControllerPipe))
	}
	if externalControllerRoutingMark != 0 {
		options = append(options, hub.WithExternalControllerRoutingMark(externalControllerRoutingMark))
	}
	if secret != "" {
		options = append(options, hub.WithSecret(secret))
	}

	if err := hub.Parse(configBytes, options...); err != nil {
		log.Fatalln("Parse config error: %s", err.Error())
	}

	if updater.GeoAutoUpdate() {
		updater.RegisterGeoUpdater()
	}

	if postDown != "" {
		defer func() {
			if _, err := cmd.ExecShell(postDown); err != nil {
				log.Errorln("post-down script error: %s", err.Error())
			}
		}()
	}
	if postUp != "" {
		if _, err := cmd.ExecShell(postUp); err != nil {
			log.Fatalln("post-up script error: %s", err.Error())
		}
	}

	defer executor.Shutdown()

	termSign := make(chan os.Signal, 1)
	hupSign := make(chan os.Signal, 1)
	signal.Notify(termSign, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(hupSign, syscall.SIGHUP)
	for {
		select {
		case <-termSign:
			return
		case <-hupSign:
			if err := hub.Parse(configBytes, options...); err != nil {
				log.Errorln("Parse config error: %s", err.Error())
			}
		}
	}
}
