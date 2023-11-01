package udp

var Options = struct {
	acceptsCap   int
	ServerWorker int //UDP工作进程数量

}{
	acceptsCap:   1024,
	ServerWorker: 64,
}
