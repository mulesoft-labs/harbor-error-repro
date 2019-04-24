package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

func doIt(ctx context.Context, regURL *url.URL, respond chan<- error) {
	regURL.Path = "/v2/"
	resp, err := http.Get(regURL.String())
	if err != nil {
		respond <- fmt.Errorf("failed to get: %s", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		respond <- nil
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		respond <- fmt.Errorf("unexpected code %d; failed to read body: %s", resp.StatusCode, err)
		return
	}
	respond <- fmt.Errorf("unexpected code %d; body: %q", resp.StatusCode, body)
}

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("%s <registry-url> <how-many>\n", os.Args[0])
		os.Exit(1)
	}

	regURL, err := url.Parse(os.Args[1])
	if err != nil {
		panic(err)
	}

	count, err := strconv.Atoi(os.Args[2])
	if err != nil {
		panic(err)
	}

	fmt.Printf("running %d pings to %s\n", count, regURL)

	ctx := context.Background()
	respond := make(chan error, count)

	for i := 0; i < count; i++ {
		go doIt(ctx, regURL, respond)
	}

	hasError := false
	for i := 0; i < count; i++ {
		err := <-respond
		if err != nil {
			hasError = true
			fmt.Println(err)
		}
		if i%5 == 4 {
			fmt.Println(i+1, "done")
		}
	}

	if hasError {
		os.Exit(1)
	}
	fmt.Println("ALL GOOD")
}
