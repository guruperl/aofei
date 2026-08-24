// Command config-preflight validates an Aofei configuration for production
// bid serving without connecting to MySQL, Redis, NATS, or another service.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/guruperl/aofei/dsp"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "s", os.Getenv("AOFEI"), "Aofei configuration path")
	flag.Parse()
	if err := run(configFile, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(configFile string, output io.Writer) error {
	if strings.TrimSpace(configFile) == "" {
		return fmt.Errorf("Aofei configuration path is required through -s or AOFEI")
	}
	config, err := dsp.NewConfig(configFile)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	if err := config.ValidateProduction(dsp.ConfigModeBid); err != nil {
		return fmt.Errorf("production configuration: %w", err)
	}
	if output == nil {
		return fmt.Errorf("preflight output is required")
	}
	_, err = fmt.Fprintln(output, "production_config_preflight=passed")
	return err
}
