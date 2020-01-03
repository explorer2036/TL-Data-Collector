package main

import (
	"TL-Data-Collector/config"
	"flag"

	"github.com/kardianos/service"
	log "github.com/sirupsen/logrus"
)

var (
	action = flag.String("action", "", "Service action for collector: start, stop, restart, install, uninstall")
)

func main() {
	flag.Parse()

	// init the settings
	var settings config.Config
	// parse the config file
	if err := config.ParseYamlFile("config.yml", &settings); err != nil {
		panic(err)
	}

	// define the service config for program
	options := make(service.KeyValue)
	options["Restart"] = "on-success"
	options["SuccessExitStatus"] = "1 2 8 SIGKILL"
	serviceConfig := &service.Config{
		Name:         "TraderLinkedDataCollector",
		DisplayName:  "Trader Linked Data Collector Service",
		Description:  "The service that collects information from trading platforms",
		Dependencies: []string{},
		Option:       options,
	}

	prog := NewProgram(&settings)
	// new the service for program
	s, err := service.New(prog, serviceConfig)
	if err != nil {
		panic(err)
	}

	// if it's the service action
	if len(*action) > 0 {
		if err := service.Control(s, *action); err != nil {
			panic(err)
		}
		return
	}

	// start the service
	if err := s.Run(); err != nil {
		log.Error(err)
	}
}

// // parse the names of json files
// func parseNames(names []string) (bool, []string, []string) {
// 	// whether the lock is for all
// 	lock := false
// 	// the types that have been locked
// 	lockTypes := make([]string, 0, len(exportTypes))
// 	// the other json files
// 	otherFiles := make([]string, 0, maxNumOfFiles)
// 	for _, name := range names {
// 		fn := strings.Split(name, ".")[0]
// 		if strings.HasPrefix(fn, "lock") {
// 			locks := strings.Split(fn, "-")
// 			if len(locks) == 1 {
// 				lock = true
// 			} else {
// 				// lock-xx.json
// 				if len(locks) == 2 {
// 					l := strings.ToLower(locks[1])
// 					if contains(exportTypes, l) {
// 						// append it to lock types only if it is a supported export type
// 						lockTypes = append(lockTypes, l)
// 					}
// 				}
// 			}
// 		} else {
// 			pieces := strings.Split(fn, "_")
// 			// a valid name should be split into 4 pieces by underscore sign
// 			// <export>_<topic>_<app>_<unique timestamp>.json
// 			if len(pieces) == 4 {
// 				otherFiles = append(otherFiles, name)
// 			}
// 		}
// 	}
// 	return lock, lockTypes, otherFiles
// }

// // get valid json files for processing, filter by export type and lock
// func getValidFiles(fs []string, lockTypes []string) []string {
// 	vfs := make([]string, 0, maxNumOfFiles)
// 	for _, f := range fs {
// 		// convert to lower case
// 		export := strings.ToLower(strings.Split(f, "_")[0])
// 		if !contains(exportTypes, export) {
// 			log.Infof("%s is skipped. Its export type is not supported", f)
// 		} else if contains(lockTypes, export) {
// 			log.Infof("%s is skipped. Its export type is locked", f)
// 		} else {
// 			// if the export is a supported type and not locked by lock-xx.json, then append it to valid file list
// 			vfs = append(vfs, f)
// 		}
// 	}
// 	return vfs
// }

// // user the 1st and 2nd piece as export and topic
// func getExportAndTopic(name string) (export string, topic string) {
// 	pieces := strings.Split(name, "_")
// 	// convert to lower case
// 	return strings.ToLower(pieces[0]), strings.ToLower(pieces[1])
// }

// process each valid json file
// func processFile(df string, name string, token string, key string, uuid string) {
// 	// concatenate data folder and json file name
// 	fullpath := df + "\\" + name
// 	pathp, err := syscall.UTF16PtrFromString(fullpath)
// 	if err != nil {
// 		return
// 	}
// 	// try to open this file exclusively by windows api, see https://golang.org/src/syscall/syscall_windows.go#L248
// 	handle, err := windows.CreateFile(pathp, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
// 	if err != nil {
// 		log.Infof("%s in use by another process, skipped", name)
// 	} else {
// 		log.Infof("%s can be opened in exclusive mode", name)
// 		var done uint32
// 		data := make([]byte, maxFileSize)
// 		windows.ReadFile(handle, data, &done, nil)
// 		// trim slice to the number of bytes that have been read
// 		data = data[:done]
// 		err = doProcess(name, data, token, key, uuid)
// 		windows.CloseHandle(handle)
// 		if err != nil {
// 			log.Errorf("Kafka write error occurs in processing file %s", name)
// 			log.Errorf("%v", err)
// 		} else {
// 			// delete this file after processing successfully(no kafka write error occurs)
// 			// leave it for next time otherwise
// 			windows.DeleteFile(pathp)
// 		}
// 	}
// }

// do the actual processing
// func doProcess(name string, data []byte, token string, key string, uuid string) error {
// 	var msgs []entity.Message
// 	err := json.Unmarshal(data, &msgs)
// 	if err != nil {
// 		// log the decoding exception
// 		log.Errorf("Error parsing json file %s", name)
// 		log.Errorf("%v", err)
// 		// but no error is returned (this file will be deleted later)
// 		return nil
// 	}
// 	ops := make([]interface{}, 0, len(msgs))
// 	timestamp := time.Now().Format(constant.Rfc3339Milli)
// 	for index, msg := range msgs {
// 		// check whether data field is a valid json string
// 		if json.Valid([]byte(msg.Data)) {
// 			op := entity.Output{
// 				Key:   key,
// 				UUID:  uuid,
// 				Value: msg,
// 				Time:  timestamp,
// 			}
// 			ops = append(ops, op)
// 		} else {
// 			// if data filed is not a valid json string, skip processing this element in the array
// 			log.Errorf("The data field is not a valid json string, file: %s index: %d, data: %s", name, index, msg.Data)
// 		}
// 	}
// 	return produce(token, ops)
// }

// returns the error in WriteMessages
// func produce(token, objs []interface{}) error {
// 	for _, obj := range objs {
// 		// buf := new(bytes.Buffer)
// 		// encoder := json.NewEncoder(buf)
// 		// // avoid the escaping of '&' used in heart beat message
// 		// encoder.SetEscapeHTML(false)
// 		// err := encoder.Encode(obj)
// 		// if err != nil {
// 		// 	log.Errorf("Error occurrred in json encoding %v", err)
// 		// 	continue
// 		// }
// 		// msg := kafka.Message{Value: buf.Bytes()}
// 		// msgs = append(msgs, msg)
// 	}
// 	wer := w.WriteMessages(context.Background(), msgs...)
// 	w.Close()
// 	return wer
// }
