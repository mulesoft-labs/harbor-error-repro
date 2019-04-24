package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/docker/distribution/registry/client/transport"
	"github.com/docker/docker/dockerversion"
	"github.com/docker/docker/registry"
	"github.com/docker/go-connections/sockets"
)

func doIt(ctx context.Context, regURL *url.URL, respond chan<- error) {
	direct := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}

	// TODO(dmcgowan): Call close idle connections when complete, use keep alive
	base := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		Dial:                direct.Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		// TODO(dmcgowan): Call close idle connections when complete and use keep alive
		DisableKeepAlives: true,
	}

	proxyDialer, err := sockets.DialerFromEnvironment(direct)
	if err == nil {
		base.Dial = proxyDialer.Dial
	}

	modifiers := registry.Headers(dockerversion.DockerUserAgent(ctx), nil)
	authTransport := transport.NewTransport(base, modifiers...)

	var transportOK bool
	_, foundVersion, err := registry.PingV2Registry(regURL, authTransport)
	if err != nil {
		transportOK = false
		if responseErr, ok := err.(registry.PingResponseError); ok {
			transportOK = true
			err = responseErr.Err
		}
	}

	if err != nil || !foundVersion {
		respond <- fmt.Errorf("error[%T / %v]; foundV2[%t]; transportOK[%t]", err, err, foundVersion, transportOK)
	}

	respond <- nil
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
