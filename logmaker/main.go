package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/natefinch/lumberjack"
)

func main() {

	flag.Parse()
	args := flag.Args()

	log.SetOutput(&lumberjack.Logger{
		Filename:   args[0],
		MaxBackups: 4,
		MaxSize:    1,
	})

	currentTime := time.Now()
	logid := fmt.Sprintf(
		"%s-%s-%s-%d",
		currentTime.Format("20060102-150405"),
		os.Getenv("USERNAME"),
		os.Getenv("COMPUTERNAME"),
		os.Getpid(),
	)

	log.Printf("logid: %s\n", logid)
	log.Printf("Hello from Go application!\n")

	rand.Seed(time.Now().UnixNano())

	linecount := 2
	padding := "882y4t8njfgiej"
	tmp := ""

	// generate 12-72 log lines, with 1-10 times padding string
	// this then pauses 30ms to avoid CPU soaking
	for {
		log.SetOutput(&lumberjack.Logger{
			Filename:   args[0],
			MaxBackups: 0,
			MaxSize:    100,
		})

		rand2 := int(rand.Intn(50) + 5)

		for i := 1; i <= rand2; i++ {
			linecount++
			rand1 := int(rand.Intn(10) + 1)
			tmp = fmt.Sprintf("linecount %d [%d] ", linecount, rand1)
			for j := 1; j <= rand1; j++ {
				tmp += padding
			}
			// write to log rolled outout using
			// lumberjack module
			log.Println(tmp)
		}
		fmt.Printf("linecount=%d\n", linecount)
		time.Sleep(1000 * time.Millisecond)

	}

}
