package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"
)

func main() {

	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	padding := "88ijyn5498yun89u89uc5ythju4yth78gy8fjuy458ky9v8ek8y56y7i4ythk7v457kyn7k4vt0u450umnv845u85vun805y0c,4xi5is395iv9nbnyu0v6598uldifu408p6yu35v9numih,d4w9,uyw508uybv8m6;ucd4luymml049mvl0u4yb95uy49lpcul48u8oyunv3o8luo82uc45yu48u5y8,54uyp96uykn0o078olk0pt,i6np.vumjocms2y4t8njfgiej"
	linecount := 0

	f, err := os.OpenFile(flag.Args()[0], os.O_CREATE|os.O_APPEND, 755)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(f)

	for {
		rand2 := int(rand.Intn(50) + 5)
		for i := 1; i <= rand2; i++ {
			linecount++
			rand1 := int(rand.Intn(100) + 20)
			log.Printf("linecount %d [%03d] %s", linecount, rand1, padding[0:rand1])
		}
		fmt.Printf("linecount=%d\n", linecount)
		time.Sleep(1000 * time.Millisecond)

	}

}
