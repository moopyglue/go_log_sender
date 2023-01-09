package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
	"nhooyr.io/websocket"
)

var conf = make(map[string]string)
var termination = make(chan string, 10)
var configfile = "gls_client.yaml"
var timeout = 5000
var shortsleeploop = 500
var bufsize = 16000

func getConfig(filename string) {

	log.Print("config: ", filename)

	yconfigfile, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	err2 := yaml.Unmarshal(yconfigfile, &conf)
	if err2 != nil {
		log.Fatal(err2)
	}

	for k, v := range conf {
		if k == "flags" {
			flags := strings.Fields(v)
			for flag := range flags {
				conf["flag."+flags[flag]] = "true"
			}
			delete(conf, "flags")
		}
	}

	for k, v := range conf {
		log.Printf("    %s -> %s\n", k, v)
	}

}

var sessionid string = ""

func generateSessionID() {
	if sessionid == "" {
		// create a sessionid for this session
		currentTime := time.Now()
		hostname, _ := os.Hostname()
		user, _ := user.Current()
		auser := strings.Split(user.Username, "\\")
		sessionid = fmt.Sprintf(
			"%s-%s-%s-%d",
			currentTime.Format("20060102-150405"),
			auser[len(auser)-1],
			hostname,
			os.Getpid(),
		)
		log.Println("Session ID:", sessionid)
	}
}

func tailToServer(logfilename string, server string) {

	// Open the local log file
	loopcount := 0
	var f *os.File
	var err error
	for {
		loopcount++
		f, err = os.Open(logfilename)
		if err != nil {
			if loopcount == 1 {
				log.Printf("log file does not yet exist - waiting")
			}
			if len(termination) > 0 {
				// when termination requested
				log.Println(err, "normal ending while waiting for log creation")
				return
			}
			time.Sleep(time.Duration(shortsleeploop) * time.Millisecond)
		} else {
			break
		}
	}
	log.Printf("log file open for reading - %s", logfilename)
	defer f.Close()

	// create a timeout context
	ctx1, ctxreclaim1 := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer ctxreclaim1()

	// upgrade connection to websocket connection
	c, _, err := websocket.Dial(ctx1, server, nil)
	if err != nil {
		// connection issues get a longer retry loop
		// TODO - randomize for better graceful recovery
		log.Println(err)
		time.Sleep(time.Duration(5*timeout) * time.Millisecond)
		return
	}
	defer c.Close(websocket.StatusNormalClosure, "closing connection")

	// handshake by sending session ID and receiving remote
	// logfile length (for new file this will be "0")
	log.Printf("sending session id %s to server %s\n", sessionid, server)
	err6 := c.Write(ctx1, websocket.MessageText, []byte(sessionid))
	if err6 != nil {
		log.Println(err, "Write() failed, disconecting")
		c.Close(websocket.StatusInternalError, "websocket Read() failed")
	}
	mtype, str, err2 := c.Read(ctx1)
	if err2 != nil {
		log.Println(err, "Read() failed, disconecting")
		c.Close(websocket.StatusInternalError, "websocket Read() failed")
		return
	}
	if mtype != websocket.MessageText {
		log.Println("remote file size handshake not in text format")
		return
	}
	remoteSize, err3 := strconv.ParseInt(string(str), 10, 64)
	if err3 != nil {
		log.Println("remote provided file size not an integer: ", str, ":", err3)
		return
	}
	log.Printf("received file length %d, handshake complete", remoteSize)

	// tail from where server log file ends
	endptr, err4 := f.Seek(0, 2)
	if err4 != nil {
		log.Println(err, ": exiting tail loop on failed seek 1")
		return
	}
	if endptr > remoteSize {
		// we have local content which has not been reflected remotely
		// reset the read point to after the last byte sent
		endptr = remoteSize
		_, err := f.Seek(remoteSize, 0)
		if err != nil {
			log.Println(err, ": exiting tail loop on failed seek 2")
			return
		}
	}
	if endptr < remoteSize && conf["flag.restartwhenshrunk"] == "true" {
		// the remote file is larger than the local file
		// Norally we start from the end of local log file but if
		// flag 'restartwhenshrunk' is set then wind back to begining of the file
		endptr = 0
		_, err := f.Seek(0, 0)
		if err != nil {
			log.Println(err, ": exiting tail loop on failed seek 3")
			return
		}
	}

	// MAIN TAIL/SEND LOOP
	// watch logfile and send data to server
	log.Printf("moved to file position %d, entering read/send loop", endptr)
	for {

		// READ DATA from log file
		databytes := make([]byte, bufsize)
		nbytes, err := f.Read(databytes)
		if nbytes == 0 {
			// pointer is at end of file
			if len(termination) > 0 {
				// when termination requested
				log.Println(err, "normal ending while tailing")
				return
			}
			time.Sleep(time.Duration(shortsleeploop) * time.Millisecond)
			continue
		} else if err != nil {
			log.Println(err, ": disconnecting 3")
			return
		}

		// WRITE DATA to remote server
		ctx2, ctxreclaim2 := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
		err = c.Write(ctx2, websocket.MessageBinary, databytes[0:nbytes])
		fmt.Println(sessionid, "sent", nbytes, "bytes")
		ctxreclaim2()
		if err != nil {
			log.Println(err)
			return
		}

	}

}

func main() {

	generateSessionID()
	getConfig(configfile)

	// wipe logfile on startup if requested
	if conf["flag.removelogfile"] == "true" {
		log.Printf("removing log file")
		os.Remove(conf["logfile"])
	}
	t, _ := strconv.Atoi(conf["timeout"])
	if t > 0 {
		timeout = t
	}

	// tail file to remote server (in background)
	go func() {
		for {
			tailToServer(conf["logfile"], conf["server"])
			if len(termination) > 0 {
				<-termination
				break
			}
			time.Sleep(time.Duration(timeout) * time.Millisecond)
		}
	}()

	// execute command (in foreground)
	cmd := strings.Fields(conf["command"])
	log.Printf("running command: %#v", cmd)
	c := exec.Command(cmd[0], cmd[1:]...)
	err := c.Run()
	if err != nil {
		log.Println("Run() complete:", err)
	} else {
		log.Println("Run() complete: normal exit")
	}

	// send termination message and wait for response
	// or timeout
	termination <- "log file session ended"
	log.Println("Waiting for final log lines to be sent to server")
	countdown := timeout
	for {
		if countdown <= 0 {
			log.Println("Giving up")
			break
		}
		countdown -= shortsleeploop
		time.Sleep(time.Duration(shortsleeploop) * time.Millisecond)
		if len(termination) == 0 {
			log.Println("Successful compeltion of log sending")
			break
		}
		log.Println("waiting...")
	}
	log.Println("sender finished , exiting")

}

/*
	TODO:
	- review to avoid crash - as crash takes out the exec.Command()
    - FIX when waiting for Read or file to appear connection can go away and it is not known

*/
