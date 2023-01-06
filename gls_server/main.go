package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
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
	return
}

func avaloqSession(w http.ResponseWriter, r *http.Request) {

	c, accepterr := websocket.Accept(w, r, nil)
	defer c.Close(websocket.StatusInternalError, "the sky is falling")

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
	defer cancel()

	logid := ""
	if accepterr == nil {

		// First (header) message should be text format
		// and contain session information
		log.Println("new connection request")
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
		logid = header[0]

		err = c.Write(ctx, websocket.MessageText, []byte("0"))
		if err != nil {
			log.Println(logid, ": unable to complete handshake: ", err)
			return
		}

		log.Println(logid, ": handshake complete")

	} else {
		log.Println(accepterr)
		return
	}
	cancel() // cancel timeout context

	for {
		mtype, data, err := c.Read(context.TODO())
		if err != nil {
			log.Printf("%s: %s", logid, err)
			break
		}
		if mtype != websocket.MessageBinary {
			log.Println(logid, ": Unexpected non-binary data encountered")
			// expecting binary log data,
			break
		}
		fmt.Printf("%s", string(data))
	}

	log.Println(logid, ": connection dropped")
	c.Close(websocket.StatusNormalClosure, "closing connection")
	return

}

func connect_listen(port string) {

	http.HandleFunc("/avaloq/", avaloqSession)
	http.HandleFunc("/", notFoundSession)

	log.Print("Opening Server on port: ", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))

}

func main() {

	// Read in yaml configuration
	getConfig(configfile)

	go connect_listen(conf["listenport"])
	go monitor()

	select {} // pause and just run daemon only

}
