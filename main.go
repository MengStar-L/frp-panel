// Command frp-panel is a single-binary web panel for managing an frp server
// (frps) or client (frpc). On first run it guides the user through downloading
// the matching frp binary and choosing a role; afterwards it manages the frp
// process, its configuration, logs, live monitoring and updates.
//
// All state — panel.json, the generated frps.toml/frpc.toml and the downloaded
// frp binaries — lives in the same directory as this executable.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"frppanel/internal/config"
	"frppanel/internal/core"
	"frppanel/internal/server"
)

func main() {
	var dirFlag, addrFlag string
	flag.StringVar(&dirFlag, "dir", "", "工作目录 (默认与程序同级),配置与 frp 二进制都存放于此")
	flag.StringVar(&addrFlag, "addr", "", "监听地址,如 :8088 (覆盖配置文件)")
	flag.Parse()

	baseDir := dirFlag
	if baseDir == "" {
		baseDir = execDir()
	}

	store, err := config.NewStore(baseDir)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	app := core.New(store)
	web, err := webFS()
	if err != nil {
		log.Fatalf("加载前端资源失败: %v", err)
	}
	srv := server.New(app, web)

	// Reuse the saved configuration on subsequent launches.
	app.AutoStartIfNeeded()

	addr := store.Get().ListenAddr
	if addrFlag != "" {
		addr = addrFlag
	}
	if addr == "" {
		addr = ":8088"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		printBanner(addr, baseDir, store.IsConfigured())
		if err := srv.Run(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP 服务启动失败: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("正在关闭,停止 frp 进程…")
	app.Shutdown()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

// execDir returns the directory containing the running executable, resolving
// symlinks. It falls back to the working directory.
func execDir() string {
	exe, err := os.Executable()
	if err != nil {
		if wd, werr := os.Getwd(); werr == nil {
			return wd
		}
		return "."
	}
	if resolved, rerr := filepath.EvalSymlinks(exe); rerr == nil {
		exe = resolved
	}
	return filepath.Dir(exe)
}

func printBanner(addr, baseDir string, configured bool) {
	host := addr
	if strings.HasPrefix(addr, ":") {
		host = "localhost" + addr
	}
	url := "http://" + host
	fmt.Println()
	fmt.Println("  ╭───────────────────────────────────────────────╮")
	fmt.Println("  │            🍦  frp 管理面板  frp-panel          │")
	fmt.Println("  ╰───────────────────────────────────────────────╯")
	fmt.Printf("   ▸ 控制台地址 : %s\n", url)
	fmt.Printf("   ▸ 工作目录   : %s\n", baseDir)
	if configured {
		fmt.Println("   ▸ 状态       : 已配置,直接登录即可")
	} else {
		fmt.Println("   ▸ 状态       : 首次使用,请在浏览器中完成初始化向导")
	}
	fmt.Println()
}
