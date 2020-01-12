package main

import (
	"TL-Data-Collector/entity"
	"TL-Data-Collector/log"
	"TL-Data-Collector/proto/gateway"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const (
	// Rfc3339Milli unified time format, refer to time.RFC3339Nano
	Rfc3339Milli = "2006-01-02T15:04:05.999Z07:00"
)

var (
	// arrary of export types, currently only "data" is supported, in future there might be DOM
	exportTypes = []string{"data"}
)

// report data by the grpc connection
func (p *Program) report(token string, v interface{}) error {
	// marshal the value to json
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	// context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// prepare the request
	request := gateway.ReportRequest{
		Token: token,
		Data:  data,
	}
	if _, err := p.reportClient.Report(ctx, &request); err != nil {
		return err
	}

	return nil
}

// load login id and token from files every time
func (p *Program) load() (string, string, string) {
	// read credential
	credential, err := ioutil.ReadFile(p.settings.App.BaseDir + credPath)
	if err != nil {
		log.Errorf("read credential file: %v", err)
		runtime.Goexit()
	}

	// read token file
	token, err := ioutil.ReadFile(p.settings.App.BaseDir + tokenPath)
	if err != nil {
		log.Errorf("read token file: %v", err)
		runtime.Goexit()
	}

	// read uuid file
	uuid, err := ioutil.ReadFile(p.settings.App.BaseDir + uuidPath)
	if err != nil {
		log.Errorf("read uuid file: %v", err)
		runtime.Goexit()
	}

	return string(credential), string(token), string(uuid)
}

// write a heartbeat message to gateway
func (p *Program) heartbeat() {
	log.Info("Heartbeat report - START")

	// load the login id, uuid and token every time
	login, token, uuid := p.load()

	// create the heartbeat data
	hearbeat := entity.Heartbeat{
		Kind:   "data_heartbeat",
		Action: "insert",
		UserID: login,
		Source: uuid,
		Path:   "&&heartbeat",
		Data: entity.HeartbeatData{
			Status: "OK",
		},
		Timestamp: time.Now().Format(Rfc3339Milli),
	}

	// send the heartbeat to gateway
	if err := p.report(token, &hearbeat); err != nil {
		log.Errorf("report %v: %v", hearbeat, err)
	}
	log.Info("Heartbeat report - END")
}

// whether the slice contains the target string or not
func contains(ss []string, target string) bool {
	b := false
	for _, s := range ss {
		if s == target {
			b = true
			break
		}
	}
	return b
}

func (p *Program) dataFolder() (string, error) {
	// parse the data folder path
	folder, err := ioutil.ReadFile(p.settings.App.BaseDir + dataPath)
	if err != nil {
		return "", err
	}
	return string(folder), nil
}

// parse the valid data files
func (p *Program) parse(folder string) ([]string, error) {
	// collect the names of json files in data folder
	files, err := ioutil.ReadDir(folder)
	if err != nil {
		return nil, err
	}
	jsons := make([]string, 0, maxNumOfFiles)
	for _, file := range files {
		name := file.Name()
		if path.Ext(name) == ".json" {
			jsons = append(jsons, name)
		}
	}

	// whether the lock is for all
	lock := false
	// the types that have been locked
	lockTypes := make([]string, 0, len(exportTypes))
	// the other json files
	dataFiles := make([]string, 0, maxNumOfFiles)
	for _, name := range jsons {
		fn := strings.Split(name, ".")[0]
		// check if it's lock file
		if strings.HasPrefix(fn, "lock") {
			locks := strings.Split(fn, "-")
			if len(locks) == 1 {
				lock = true // lock file
			} else {
				// lock-xx.json
				if len(locks) == 2 {
					l := strings.ToLower(locks[1])
					// lock-data.json
					if contains(exportTypes, l) {
						// append it to lock types only if it is a supported export type
						lockTypes = append(lockTypes, l)
					}
				}
			}
		} else {
			pieces := strings.Split(fn, "_")
			// a valid name should be split into 4 pieces by underscore sign
			// <export>_<topic>_<app>_<unique timestamp>.json
			if len(pieces) == 4 {
				dataFiles = append(dataFiles, name)
			}
		}
	}
	if lock == true {
		log.Info("lock.json is in the data folder")
		return nil, nil

	}

	// get valid json files for processing, filter by export type and lock
	vfs := make([]string, 0, maxNumOfFiles)
	// <export>_<topic>_<app>_<unique timestamp>.json
	for _, f := range dataFiles {
		// convert to lower case
		export := strings.ToLower(strings.Split(f, "_")[0])
		if !contains(exportTypes, export) {
			log.Infof("%s is skipped. It's export type is not supported", f)
		} else if contains(lockTypes, export) {
			log.Infof("%s is skipped. It's export type is locked", f)
		} else {
			// if the export is a supported type and not locked by lock-xx.json, then append it to valid file list
			vfs = append(vfs, f)
		}
	}

	return vfs, nil
}

func validate(m *entity.Message) error {
	if m.Kind == "" {
		return errors.New("dtype is empty")
	}
	if m.Action != "insert" && m.Action != "update" {
		return fmt.Errorf("invalid action: %v", m.Action)
	}
	if m.Source == "" {
		return errors.New("source is empty")
	}
	if m.Path == "" {
		return errors.New("path is empty")
	}
	if m.Time == "" {
		return errors.New("time is empty")
	}
	if len(m.Data) == 0 {
		return errors.New("data is empty")
	}
	if json.Valid([]byte(m.Data)) == false {
		return errors.New("data is invalid json")
	}

	return nil
}

// transfer the data
func (p *Program) transfer(name string, data []byte, login string, token string) error {
	var msgs []entity.Message

	for index, msg := range msgs {
		// check whether data field is a valid json string
		if err := validate(&msg); err != nil {
			log.Errorf("message is invalid, file: %s index: %d, data: %s, error: %v", name, index, msg, err)
			continue
		}

		// these two fields are provided by collector
		msg.UserID = login
		msg.Timestamp = time.Now().Format(Rfc3339Milli)

		// send the data to gateway
		if err := p.report(token, &msg); err != nil {
			log.Errorf("send data to gateway, file: %s index: %d, data: %v, error: %v", name, index, msg, err)
			continue
		}
	}

	return nil
}

// process the data file
func (p *Program) process(folder string, name string, login string, token string) {
	// concatenate data folder and json file name
	fullpath := folder + "\\" + name
	pathPtr, err := syscall.UTF16PtrFromString(fullpath)
	if err != nil {
		return
	}

	// try to open this file exclusively by windows api, see https://golang.org/src/syscall/syscall_windows.go#L248
	handle, err := windows.CreateFile(pathPtr, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		log.Infof("%s in use by another process", name)
	} else {
		log.Infof("%s can be opened in exclusive mode", name)

		var done uint32
		data := make([]byte, maxFileSize)
		// read from the file
		windows.ReadFile(handle, data, &done, nil)
		// trim slice to the number of bytes that have been read
		data = data[:done]

		// transfer the messages to gateway
		err = p.transfer(name, data, login, token)

		// close the file first
		windows.CloseHandle(handle)

		if err != nil {
			log.Errorf("write error occurs in processing file %s: %v", name, err)
		} else {
			// delete this file after processing successfully(no kafka write error occurs)
			// leave it for next time otherwise
			windows.DeleteFile(pathPtr)
		}
	}
}

// collecting messages from data folder and sending to gateway
func (p *Program) collect() {
	// load the login id, uuid and token every time
	login, token, _ := p.load()

	log.Info("Data collecting - START")

	// retrieve the data folder
	folder, err := p.dataFolder()
	if err != nil {
		log.Errorf("parse to retrieve files: %v", err)
		return
	}

	// parse the valid data files
	files, err := p.parse(folder)
	if err != nil {
		log.Errorf("parse to retrieve files: %v", err)
		return
	}
	if len(files) == 0 {
		log.Info("no files to process")
		return
	}

	// handle the data files one bye one
	for _, file := range files {
		p.process(folder, file, login, token)
	}

	// handle the files
	log.Info("Data collecting - END")
}
