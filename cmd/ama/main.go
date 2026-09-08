package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/svalentino/ama/internal/app"
	"github.com/svalentino/ama/internal/llm/factory"
)

func main() {
	application, err := app.New(factory.New, os.Stdout, buildVersion())
	if err != nil {
		fmt.Fprintf(os.Stderr, "ama: %v\n", err)
		os.Exit(1)
	}
	if err := application.Run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ama: %v\n", err)
		os.Exit(1)
	}
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
