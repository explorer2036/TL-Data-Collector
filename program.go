package main

import (
	"TL-Data-Collector/config"
	"TL-Proto/gateway"
	"fmt"
	"os"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/kardianos/service"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
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

// Program define Start and Stop methods.
type Program struct {
	logFile      *os.File                    // file handler for logger
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

// initialize the logger file and set log level
func (p *Program) initLogger() {
	// e.g. logs\2019-06\2019-06-19.txt
	day := time.Now().Format("2006-01-02")
	dir := fmt.Sprintf("%s\\logs", p.settings.BaseDir)

	// create directoy if not exists
	if !exists(dir) {
		if err := os.Mkdir(dir, 0660); err != nil {
			panic(err)
		}
	}

	name := fmt.Sprintf("%s\\%s", dir, day)
	// open the log file
	file, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		panic(err)
	}
	p.logFile = file

	log.SetOutput(file)
	log.SetLevel(log.InfoLevel)
}

// init the report service client
func (p *Program) initReportClient() {
	// set up a connection to the server.
	conn, err := grpc.Dial(
		p.settings.Gateway,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		panic(err)
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

	// init the logger
	program.initLogger()

	// init the grpc connection and report client
	program.initReportClient()

	// lock file name
	program.lockFile = settings.BaseDir + "\\tmp.lock"

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

	c := cron.New()
	heartbeat := fmt.Sprintf("%d", p.settings.Heartbeat) + "m"
	c.AddFunc("@every "+heartbeat, func() {
		p.heartbeat()
	})
	collect := fmt.Sprintf("%d", p.settings.Collect) + "m"
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

	// close the log file
	if p.logFile != nil {
		p.logFile.Close()
	}

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
