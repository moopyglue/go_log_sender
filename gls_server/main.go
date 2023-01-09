package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
	"nhooyr.io/websocket"
)

//===========================================================

var conf = make(map[string]string)
var configfile = "gls_server.yaml"

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

func monitor() {
	for {
		time.Sleep(time.Minute)
		log.Print("monitor")
	}
}

func notFoundSession(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func recordSession(w http.ResponseWriter, r *http.Request) {

	// Upgrade normal connection to websocket
	c, accepterr := websocket.Accept(w, r, nil)
	if accepterr != nil {
		log.Println("connection Accept() failure:", accepterr)
		return
	}
	defer c.Close(websocket.StatusInternalError, "the sky is falling")

	// First (header) message should be text format
	// and contain session information
	log.Println("handshaking new connection request")
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
	defer cancel()
	mtype, str, err := c.Read(ctx)
	if err != nil {
		log.Println(err)
		c.Close(websocket.StatusNormalClosure, err.Error())
		return
	}
	if mtype != websocket.MessageText {
		log.Println("invalid remote greeting, closing connection")
		c.Close(websocket.StatusNormalClosure, "invalid remote greeting")
		return
	}
	header := strings.Split(strings.ReplaceAll(string(str), "\r\n", "\n"), "\n")
	logid := header[0]

	// CREATE AND OPEN log file, create empty if does not exist
	logfilename := conf["logdirectory"] + "/" + logid
	outfile, err3 := os.OpenFile(logfilename, os.O_WRONLY|os.O_CREATE, 0755)
	if err3 != nil {
		log.Println(logid, err3)
		return
	}
	defer outfile.Close()
	endptr, err4 := outfile.Seek(0, 2)
	if err4 != nil {
		log.Println(logid, err4)
		return
	}

	err = c.Write(ctx, websocket.MessageText, []byte(strconv.FormatInt(endptr, 10)))
	if err != nil {
		log.Println(logid, ": unable to complete handshake: ", err)
		return
	}
	log.Println(logid, ": handshake complete")
	log.Printf("opened log file %s at position %d", logfilename, endptr)

	cancel() // cancel timeout context

	// MAIN LOOP
	for {
		// READ DATA from incoming connection
		mtype, data, err := c.Read(context.TODO())
		if err != nil {
			log.Printf("%s: %s", logid, err)
			break
		}
		if mtype != websocket.MessageBinary {
			// expecting binary log data,
			log.Println(logid, ": Unexpected non-binary data encountered")
			break
		}

		//WRITE DATA to log file
		_, err2 := outfile.Write(data)
		if err2 != nil {
			log.Println(logid, "Write(): ", err2)
			return
		}
	}

	log.Println(logid, ": connection dropped")
	c.Close(websocket.StatusNormalClosure, "closing connection")

}

func connect_listen(port string, name string) {

	http.HandleFunc("/"+name, recordSession)
	http.HandleFunc("/", notFoundSession)

	log.Print("Opening Server on port: ", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))

}

func main() {

	// Read in yaml configuration
	getConfig(configfile)

	go connect_listen(conf["listenport"], conf["listenname"])
	go monitor()

	select {} // pause and just run daemon only

}

/*
   TODO

   - what else can be configureable via config file? (for client also)

*/
