package main

func main() {
	//var (
	//	cgroupPath = flag.String("cgroup", "/sys/fs/cgroup", "Path to cgroup v2 mount")
	//)
	//flag.Parse()
	//
	//// Create sockmap manager
	//manager, err := sockmap.NewSockmapManager(*cgroupPath)
	//if err != nil {
	//	log.Fatalf("Failed to create sockmap manager: %v", err)
	//}
	//defer manager.StopSpec()
	//
	//// Create context for graceful shutdown
	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()
	//
	//// StartSpec the manager
	//if err := manager.StartSpec(ctx); err != nil {
	//	log.Fatalf("Failed to start sockmap manager: %v", err)
	//}
	//
	//// Example: Register a service manually for testing
	//// In production, this would be handled by K8s service discovery
	//exampleService := sockmap.Service{
	//	Name:      "example-service",
	//	Namespace: "default",
	//	ServiceIP: net.ParseIP("10.96.0.1"),
	//	Port:      80,
	//	Strategy:  sockmap.RoundRobin,
	//	Endpoints: []sockmap.Container{
	//		{
	//			PodIP:     net.ParseIP("10.244.0.10"),
	//			Namespace: "default",
	//			PodName:   "pod-1",
	//			Port:      8080,
	//		},
	//		{
	//			PodIP:     net.ParseIP("10.244.0.11"),
	//			Namespace: "default",
	//			PodName:   "pod-2",
	//			Port:      8080,
	//		},
	//	},
	//}
	//
	//if err := manager.RegisterService(exampleService); err != nil {
	//	log.Printf("Warning: Failed to register example service: %v", err)
	//}
	//
	//log.Println("Sockmap redirector is running...")
	//log.Println("TCP connections to registered services will be redirected to backend containers")
	//
	//// Wait for shutdown signal
	//sigChan := make(chan os.Signal, 1)
	//signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	//<-sigChan
	//
	//log.Println("Shutting down sockmap redirector...")
	//cancel()
}
