package main

import (
	"TL-Data-Collector/config"
	"TL-Data-Collector/log"
	"TL-Data-Collector/proto/gateway"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"time"

	"github.com/kardianos/service"
	"github.com/robfig/cron"
	"golang.org/x/sys/windows"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

var (
	// assume that there will be 100 json files at most
	maxNumOfFiles = 100
	// assume that the file size won't exceed 1MB
	maxFileSize = 1024 * 1024
	// data file path
	dataPath = "\\conf\\appdata.txt"
	// credential file path
	credPath = "\\conf\\credential"
	// token file path
	tokenPath = "\\conf\\token"
	// uuid file path
	uuidPath = "\\conf\\uuid"
)

const (
	// GRPCClientDialTimeout - the timeout for client to dial the server
	GRPCClientDialTimeout = 5
	// GRPCClientKeepaliveTime - After a duration of this time if the client doesn't see any activity it
	// pings the server to see if the transport is still alive.
	GRPCClientKeepaliveTime = 15
	// GRPCClientKeepaliveTimeout - After having pinged for keepalive check, the client waits for a duration
	// of Timeout and if no activity is seen even after that the connection is closed.
	GRPCClientKeepaliveTimeout = 5

	// DefaultHeartbeatInterval - the default interval time for heartbeat jobs
	DefaultHeartbeatInterval = 10
	// DefaultCollectInterval - the default interval time for collect jobs
	DefaultCollectInterval = 30
)

// Program define Start and Stop methods.
type Program struct {
	lockFile     string                      // lock file name
	settings     *config.Config              // the settings for program
	exit         chan struct{}               // exit signal
	conn         *grpc.ClientConn            // grpc connection
	reportClient gateway.ReportServiceClient // report client
}

// check if the path exists or not
func exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

// prepare the files for certification
func (p *Program) createCredentials() credentials.TransportCredentials {
	cert, err := tls.LoadX509KeyPair(p.settings.TLS.Perm, p.settings.TLS.Key)
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(p.settings.TLS.Ca)
	if err != nil {
		panic(err)
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		panic("append certs from pem")
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   "collector",
		RootCAs:      certPool,
	})
}

// init the report service client
func (p *Program) initReportClient() {
	// prepare the dial options for grpc client
	opts := []grpc.DialOption{}
	opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:    GRPCClientKeepaliveTime * time.Second,
		Timeout: GRPCClientKeepaliveTimeout * time.Second,
	}))
	if p.settings.TLS.Switch {
		// prepare the credentials with the ca files
		creds := p.createCredentials()
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	// set up a connection to the server.
	conn, err := grpc.Dial(p.settings.App.Gateway, opts...)
	if err != nil {
		panic(fmt.Sprintf("grpc conn: %v", err))
	}
	// update the connection
	p.conn = conn

	// new report service clint
	p.reportClient = gateway.NewReportServiceClient(conn)
}

// NewProgram create a program for crob jobs
func NewProgram(settings *config.Config) *Program {
	program := &Program{
		settings: settings,
		exit:     make(chan struct{}),
	}

	// init the grpc connection and report client
	program.initReportClient()

	// lock file name
	program.lockFile = settings.App.BaseDir + "\\tmp.lock"

	return program
}

// Start the service
func (p *Program) Start(s service.Service) error {
	if service.Interactive() {
		log.Info("Running in terminal.")
	} else {
		log.Info("Running under service manager.")
	}

	// do the actual work async.
	go p.run()

	return nil
}

// run the service logic
func (p *Program) run() error {
	// check if the program is already running
	handle, err := p.hasLockfile()
	if err != nil {
		log.Info("A collector process is already running")
		os.Exit(1)
	}

	// try to start a cron job for data collecting
	log.Info("Cron job for data collecting preparing")

	if p.settings.App.Heartbeat < DefaultHeartbeatInterval {
		p.settings.App.Heartbeat = DefaultHeartbeatInterval
	}
	if p.settings.App.Collect < DefaultCollectInterval {
		p.settings.App.Collect = DefaultCollectInterval
	}

	c := cron.New()
	heartbeat := fmt.Sprintf("%d", p.settings.App.Heartbeat) + "s"
	c.AddFunc("@every "+heartbeat, func() {
		p.heartbeat()
	})
	collect := fmt.Sprintf("%d", p.settings.App.Collect) + "s"
	c.AddFunc("@every "+collect, func() {
		p.collect()
	})
	c.Start()

	log.Info("Cron job for data collecting started")

	// block until receive a exit signal
	for {
		select {
		case <-p.exit:
			// stop the cron job
			c.Stop()

			// close the lock file
			windows.CloseHandle(handle)

			log.Info("Cron job for data collecting ended")
			return nil
		}
	}
}

// Stop the service
func (p *Program) Stop(s service.Service) error {
	// signal the program to exit
	close(p.exit)

	// close the grpc connection
	if p.conn != nil {
		p.conn.Close()
	}

	return nil
}

func (p *Program) hasLockfile() (windows.Handle, error) {
	if !exists(p.lockFile) {
		// create the lock file if it doesn't exist
		f, err := os.Create(p.lockFile)
		if err != nil {
			return windows.InvalidHandle, err
		}
		f.Close()
	}
	filePath, err := syscall.UTF16PtrFromString(p.lockFile)
	if err != nil {
		return windows.InvalidHandle, err
	}
	// try to read this file - no sharing
	return windows.CreateFile(filePath, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
}
