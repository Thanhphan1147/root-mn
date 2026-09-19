package main

import (
	"flag"
	"fmt"
	"net/http"
)

// cmdServe serves the static analysis tool. The engine runs client-side as
// WebAssembly and the game state lives in the browser's localStorage, so this
// is only needed for local preview or a container; the site can be hosted as
// plain static files (e.g. GitHub Pages).
func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "listen address")
	dir := fs.String("web", "docs", "static site directory")
	fs.Parse(args)
	fmt.Printf("ROOT analysis tool (static) on http://localhost%s  serving %s\n", *addr, *dir)
	if err := http.ListenAndServe(*addr, http.FileServer(http.Dir(*dir))); err != nil {
		fmt.Println("server error:", err)
	}
}
