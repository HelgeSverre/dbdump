package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := "8000"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	fs := http.FileServer(http.Dir("."))
	http.Handle("/", fs)

	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("\n🌐 dbdump website server\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Server running at: http://localhost%s\n", addr)
	fmt.Printf("Press Ctrl+C to stop\n\n")

	log.Fatal(http.ListenAndServe(addr, nil))
}
