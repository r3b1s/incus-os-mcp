package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/gorilla/websocket"
	incusclient "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"golang.org/x/term"

	"incus-os-mcp/internal/incus"
)

type consoleTerminal struct {
	io.Reader
	io.Writer
}

func (consoleTerminal) Close() error { return nil }

func cmdConsole(args []string) error {
	fs := flag.NewFlagSet("console", flag.ContinueOnError)
	var f flags
	f.register(fs)
	force := fs.Bool("force", false, "attach even when an active console session exists")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: incus-os-mcp console [flags] <instance>")
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("console requires an interactive terminal on stdin and stdout")
	}

	cfg, err := f.loadConfig()
	if err != nil {
		return err
	}
	client, err := incus.New(cfg)
	if err != nil {
		return fmt.Errorf("target connection failed: %w", err)
	}
	server := client.Server.UseProject(client.Project(f.project))

	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("enable raw terminal mode: %w", err)
	}
	defer func() { _ = term.Restore(fd, state) }()

	width, height, err := term.GetSize(fd)
	if err != nil {
		return fmt.Errorf("read terminal size: %w", err)
	}

	disconnect := make(chan bool)
	var disconnectOnce sync.Once
	closeDisconnect := func() { disconnectOnce.Do(func() { close(disconnect) }) }
	defer closeDisconnect()
	control := func(conn *websocket.Conn) {
		resize := make(chan os.Signal, 1)
		signal.Notify(resize, syscall.SIGWINCH)
		defer signal.Stop(resize)
		for range resize {
			w, h, err := term.GetSize(int(os.Stdout.Fd()))
			if err != nil {
				continue
			}
			if err := conn.WriteJSON(api.InstanceExecControl{
				Command: "window-resize",
				Args: map[string]string{
					"width":  strconv.Itoa(w),
					"height": strconv.Itoa(h),
				},
			}); err != nil {
				return
			}
		}
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)
	go func() {
		<-interrupt
		closeDisconnect()
	}()

	name := fs.Arg(0)
	fmt.Fprint(os.Stdout, "Attached to serial console; press Ctrl-C to detach.\r\n")
	op, err := server.ConsoleInstance(name, api.InstanceConsolePost{
		Type:   "console",
		Width:  width,
		Height: height,
		Force:  *force,
	}, &incusclient.InstanceConsoleArgs{
		Terminal:          consoleTerminal{Reader: os.Stdin, Writer: os.Stdout},
		Control:           control,
		ConsoleDisconnect: disconnect,
	})
	if err != nil {
		return err
	}
	if err := op.Wait(); err != nil {
		return err
	}
	return nil
}
