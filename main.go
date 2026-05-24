package main

import (
	"fmt"
	"os"
)

func main() {
	cfg := LoadConfig()
	state := LoadState(cfg.StatePath)
	cfg = ResolveConfig(cfg, state)
	cmd, err := ParseCommand(os.Args[1:], cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "diffy:", err)
		fmt.Fprintln(os.Stderr, "Run `diffy --help` for usage.")
		os.Exit(2)
	}

	if cmd.Options.Help {
		fmt.Print(Help(cmd))
		if len(Help(cmd)) == 0 || Help(cmd)[len(Help(cmd))-1] != '\n' {
			fmt.Println()
		}
		return
	}

	controller := Controller{Git: Git{}}

	if cmd.Options.Summary {
		out, err := controller.Summary(cmd)
		if err != nil {
			fmt.Fprintln(os.Stderr, "diffy:", err)
			os.Exit(1)
		}
		fmt.Print(out)
		return
	}

	if cmd.Options.Raw {
		out, err := controller.Raw(cmd)
		if err != nil {
			fmt.Fprintln(os.Stderr, "diffy:", err)
			os.Exit(1)
		}
		fmt.Print(out)
		return
	}

	if err := RunTUI(cmd, cfg, state); err != nil {
		fmt.Fprintln(os.Stderr, "diffy:", err)
		os.Exit(1)
	}
}
