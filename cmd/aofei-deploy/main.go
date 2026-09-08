// Command aofei-deploy validates and operates one target-owned immutable
// Aofei/Pzdesign/Genelet deployment realization.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/guruperl/aofei/internal/deployment"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "aofei-deploy:", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("aofei-deploy", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	manifestPath := flags.String("manifest", "", "absolute target-owned environment manifest")
	unitTemplate := flags.String("unit-template", "", "absolute target-owned systemd unit template")
	historyDir := flags.String("history-dir", "", "absolute target-owned credential-free history directory")
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "usage: aofei-deploy [-manifest PATH -unit-template PATH -history-dir PATH] COMMAND [RELEASE]")
		fmt.Fprintln(flags.Output(), "commands: verify-release, validate, status, preflight, bootstrap, deploy")
	}
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	remaining := flags.Args()
	if len(remaining) == 0 {
		flags.Usage()
		return errors.New("command is required")
	}
	command := remaining[0]
	if command == "verify-release" {
		if len(remaining) != 2 {
			return errors.New("verify-release requires one release")
		}
		release, err := deployment.VerifyRelease(remaining[1], deployment.BuildInfoInspector{})
		if err != nil {
			return err
		}
		fmt.Printf("release_verification=passed release_id=%s\n", release.Manifest.ReleaseID)
		return nil
	}
	if *manifestPath == "" {
		return errors.New("manifest is required")
	}
	environment, err := deployment.LoadEnvironment(*manifestPath)
	if err != nil {
		return err
	}
	if command == "validate" {
		if len(remaining) != 1 {
			return errors.New("validate accepts no release")
		}
		fmt.Println("environment_manifest=passed")
		return nil
	}
	if *unitTemplate == "" || *historyDir == "" {
		return errors.New("unit-template and history-dir are required")
	}
	engine, err := deployment.NewEngine(environment, *unitTemplate, *historyDir)
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	switch command {
	case "status":
		if len(remaining) != 1 {
			return errors.New("status accepts no release")
		}
		return engine.Status(ctx)
	case "preflight", "bootstrap", "deploy":
		if len(remaining) != 2 {
			return fmt.Errorf("%s requires one release", command)
		}
		switch command {
		case "preflight":
			_, err = engine.Preflight(ctx, remaining[1])
			return err
		case "bootstrap":
			return engine.Bootstrap(ctx, remaining[1])
		default:
			return engine.Deploy(ctx, remaining[1])
		}
	default:
		flags.Usage()
		return fmt.Errorf("unknown command %q", command)
	}
}
