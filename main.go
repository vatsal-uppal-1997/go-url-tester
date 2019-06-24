package main

import (
	"fmt"
	"flag"
	"regexp"
	"net/url"
	"net/http"
	"crypto/tls"
	"io/ioutil"
	"strings"
	"unicode/utf8"
	"sync"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func processFile(filename string) (valid [][]string, invalid []string) {
	fixUtf := func(r rune) rune {
    			if r == utf8.RuneError {
        			return -1
    			}
    			return r
		}
	dat, err := ioutil.ReadFile(filename)
	check(err)
	re := regexp.MustCompile("[\\r\\n]+")
	urls := re.Split(string(dat), -1)
	for i := range urls {
		urls[i] = strings.Map(fixUtf, urls[i])
		urls[i] = strings.ToLower(strings.TrimSpace(urls[i]))
		if parsed, err := url.Parse(urls[i]); err != nil {
			invalid = append(invalid, urls[i])
		} else {
			parsed.Scheme = "https"
			temp := []string{}
			temp = append(temp, parsed.String())
			parsed.Scheme = "http"
			temp = append(temp, parsed.String())
			valid = append(valid, temp)
		}
	}
	return
}

func openUrl(url string) bool {
	var netClient = &http.Client{
 		 Timeout: time.Second * 300,
	}
	resp, err := netClient.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return true
}

func workers(input <- chan []string, canOpen chan <- string, cannotOpen chan <- string, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := range input {
		https := openUrl(i[0])
		if (https == true) {
			canOpen <- i[0]
			fmt.Println("[+]", i[0], "is accessible")
			continue
		}
		http := openUrl(i[1])
		if (http == true) {
			canOpen <- i[1]
			fmt.Println("[+]", i[1], "is accessible")
			continue
		}
		cannotOpen <- i[0]
		fmt.Println("[-]", i[1], "is not accessible")
	}
}

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	inputFile := flag.String("input", "urls.txt", "Input File Name")
	successFile := flag.String("success", "canOpen", "URL's that can be opened")
	failedFile := flag.String("failed", "cannotOpen", "URL's that can't be opened")
	threads :=  flag.Int("threads", 4, "Number of threads")
	flag.Parse()
	valid, invalid := processFile(*inputFile)
	fmt.Println("Invalid:",invalid)
	var wg sync.WaitGroup
	inputChannel := make(chan []string, len(valid))
	canOpenChannel := make(chan string, len(valid))
	cannotOpenChannel := make(chan string, len(valid))
	for i:=0; i<*threads; i++ {
		wg.Add(1)
		go workers(inputChannel, canOpenChannel, cannotOpenChannel, &wg)
	}
	for i := range valid {
		inputChannel <- valid[i]
	}
	close(inputChannel)
	wg.Wait()
	close(canOpenChannel)
	close(cannotOpenChannel)
	fmt.Println("Dumping Output")
	var canOpen strings.Builder
	var cannotOpen strings.Builder
	for i := range canOpenChannel {
		canOpen.WriteString(i)
		canOpen.WriteString("\n")
	}
	for i:= range cannotOpenChannel {
		cannotOpen.WriteString(i)
		cannotOpen.WriteString("\n")
	}
	bytes := []byte(canOpen.String())
	err := ioutil.WriteFile(*successFile, bytes, 0644)
	check(err)
	bytes = []byte(cannotOpen.String())
	err = ioutil.WriteFile(*failedFile, bytes, 0644)
	check(err)
}
