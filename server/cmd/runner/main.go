package main

import (
	"log"
	"os"
	"os/exec"
	"time"
)

func main() {
	// Step 1: Luôn install CLI mới nhất, bất kể kết quả build server.
	// Đây là điểm mấu chốt: CLI phải được cập nhật TRƯỚC khi server khởi động.
	log.Println("[Runner] Installing mangahub CLI...")
	installCmd := exec.Command("go", "install", "./cmd/mangahub")
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		log.Printf("[Runner] WARNING: CLI install failed: %v (continuing with old binary)", err)
	} else {
		log.Println("[Runner] ✓ mangahub CLI installed successfully")
	}

	// Step 2: Dọn dẹp tiến trình cũ nếu còn kẹt lại
	log.Println("[Runner] Cleaning up old server processes...")
	exec.Command("taskkill", "/F", "/IM", "api-server-dev.exe").Run()
	exec.Command("taskkill", "/F", "/IM", "tcp-server-dev.exe").Run()
	exec.Command("taskkill", "/F", "/IM", "udp-server-dev.exe").Run()
	exec.Command("taskkill", "/F", "/IM", "grpc-server-dev.exe").Run()
	time.Sleep(200 * time.Millisecond) // Cho OS thời gian giải phóng port

	// Step 3: Khởi động API Server
	log.Println("[Runner] Starting API Server...")
	apiCmd := exec.Command(".\\api-server-dev.exe")
	apiCmd.Stdout = os.Stdout
	apiCmd.Stderr = os.Stderr
	if err := apiCmd.Start(); err != nil {
		log.Fatalf("[Runner] Failed to start API Server: %v", err)
	}

	// Step 4: Khởi động TCP Server
	log.Println("[Runner] Starting TCP Server...")
	tcpCmd := exec.Command(".\\tcp-server-dev.exe")
	tcpCmd.Stdout = os.Stdout
	tcpCmd.Stderr = os.Stderr
	if err := tcpCmd.Start(); err != nil {
		log.Fatalf("[Runner] Failed to start TCP Server: %v", err)
	}

	// Step 5: Khởi động UDP Server
	log.Println("[Runner] Starting UDP Server...")
	udpCmd := exec.Command(".\\udp-server-dev.exe")
	udpCmd.Stdout = os.Stdout
	udpCmd.Stderr = os.Stderr
	if err := udpCmd.Start(); err != nil {
		log.Fatalf("[Runner] Failed to start UDP Server: %v", err)
	}

	// Step 6: Khởi động gRPC Server
	log.Println("[Runner] Starting gRPC Server...")
	grpcCmd := exec.Command(".\\grpc-server-dev.exe")
	grpcCmd.Stdout = os.Stdout
	grpcCmd.Stderr = os.Stderr
	if err := grpcCmd.Start(); err != nil {
		log.Fatalf("[Runner] Failed to start gRPC Server: %v", err)
	}

	// Giữ runner sống để Air có thể kill nó khi cần reload.
	// Nếu một trong các server thoát, log ra để dễ debug.
	done := make(chan error, 4)
	go func() { done <- apiCmd.Wait() }()
	go func() { done <- tcpCmd.Wait() }()
	go func() { done <- udpCmd.Wait() }()
	go func() { done <- grpcCmd.Wait() }()

	err := <-done
	log.Printf("[Runner] A server process exited: %v — runner will exit so Air can restart.", err)
}
