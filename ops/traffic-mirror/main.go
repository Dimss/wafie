package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	startCmd.PersistentFlags().IntP("port", "", 8080, "listening port")
	viper.BindPFlag("port", startCmd.PersistentFlags().Lookup("port"))
	rootCmd.AddCommand(startCmd)
}

var rootCmd = &cobra.Command{
	Use:   "traffic-mirror",
	Short: "Wafie Traffic Mirror server",
}
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts traffic mirror server",
	Run: func(cmd *cobra.Command, args []string) {
		http.HandleFunc("/", handler)
		listenAddr := fmt.Sprintf(":%d", viper.GetInt("port"))
		log.Printf("starting traffic mirror server on port %s\n", listenAddr)
		go func() {
			log.Fatal(http.ListenAndServe(listenAddr, nil))
		}()
		// handle interrupts
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		s := <-sigCh
		log.Printf("signal received: %s, shutting down\n", s.String())
		log.Println("bye bye 👋")
		os.Exit(0)
	},
}

// handler is the function that processes incoming HTTP requests.
func handler(w http.ResponseWriter, r *http.Request) {
	// Log the incoming request details
	log.Printf("Received request: Method=%s, URL=%s, RemoteAddr=%s", r.Method, r.URL.Path, r.RemoteAddr)

	// You can access request details like:
	// r.Method (GET, POST, etc.)
	// r.URL.Path (the requested path)
	// r.URL.Query() (query parameters for GET requests)
	// r.Body (request body for POST/PUT requests)
	// r.Header (request headers)

	// Example: Respond with a simple message including the requested path
	//fmt.Fprintf(w, "Hello from Go HTTP server! You requested: %s\n", r.URL.Path)

	w.WriteHeader(http.StatusOK) // This is the default if not explicitly set
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
