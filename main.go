package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"

	"github.com/minio/minio-go/v6/pkg/credentials"

	"github.com/dciangot/sts-wire/pkg/core"
	_ "github.com/go-bindata/go-bindata"
	"github.com/pkg/browser"
)

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func DownloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func main() {

	sigint := make(chan int, 1)
	// get rclone from bindata
	//_, err := Asset("data/rclone")
	//if err != nil {
	//	panic(err)
	//}
	inputReader := *bufio.NewReader(os.Stdin)
	scanner := core.GetInputWrapper{
		Scanner: inputReader,
	}

	if len(os.Args) < 4 {
		fmt.Println("Invalid instance name, please type 'sts-wire -h'  for help")
		return
	}

	instance := os.Args[1]
	if instance == "-h" {
		fmt.Println("sts-wire <instance name> <rclone remote path> <local mount point>")
		return
	}

	remote := os.Args[2]

	local := os.Args[3]
	//fmt.Println(instance)

	// if instance == "" {
	// 	instance, err := scanner.GetInputString("Insert a name for the instance: ", "")
	// 	if err != nil {
	// 		panic(err)
	// 	} else if instance == "" {
	// 		panic(fmt.Errorf("Please insert a valid name."))
	// 	}
	// }

	clientConfig := core.IAMClientConfig{
		Port:       3128,
		ClientName: "oidc-client",
	}

	// Create a CA certificate pool and add cert.pem to it
	//caCert, err := ioutil.ReadFile("MINIO.pem")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//caCertPool := x509.NewCertPool()
	//caCertPool.AppendCertsFromPEM(caCert)

	// Create the TLS Config with the CA pool and enable Client certificate validation
	cfg := &tls.Config{
		//ClientCAs: caCertPool,
		InsecureSkipVerify: true,
	}
	//cfg.BuildNameToCertificate()

	tr := &http.Transport{
		TLSClientConfig: cfg,
	}

	httpClient := &http.Client{
		Transport: tr,
	}

	clientIAM := core.InitClientConfig{
		ClientConfig: clientConfig,
		Scanner:      scanner,
		HTTPClient:   *httpClient,
	}

	endpoint, clientResponse, err := clientIAM.InitClient(instance)
	if err != nil {
		log.Fatal(err)
	}

	//fmt.Println(clientResponse.ClientID)
	//fmt.Println(clientResponse.ClientSecret)

	ctx := context.Background()

	config := oauth2.Config{
		ClientID:     clientResponse.ClientID,
		ClientSecret: clientResponse.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  endpoint + "/authorize",
			TokenURL: endpoint + "/token",
		},
		RedirectURL: fmt.Sprintf("http://localhost:%d/oauth2/callback", clientConfig.Port),
		Scopes:      []string{"address", "phone", "openid", "email", "profile", "offline_access"},
	}

	state := core.RandomState()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//log.Printf("%s %s", r.Method, r.RequestURI)
		if r.RequestURI != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, config.AuthCodeURL(state), http.StatusFound)
	})

	http.HandleFunc("/oauth2/callback", func(w http.ResponseWriter, r *http.Request) {
		//log.Printf("%s %s", r.Method, r.RequestURI)
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}

		oauth2Token, err := config.Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "cannot get token", http.StatusBadRequest)
			return
		}
		if !oauth2Token.Valid() {
			http.Error(w, "token expired", http.StatusBadRequest)
			return
		}

		token := oauth2Token.Extra("access_token").(string)

		//fmt.Println(token)

		//sts, err := credentials.NewSTSWebIdentity("https://131.154.97.121:9001/", getWebTokenExpiry)
		providers := []credentials.Provider{
			&core.IAMProvider{
				StsEndpoint: "https://131.154.97.121:9001",
				Token:       token,
				HTTPClient:  httpClient,
			},
		}

		sts := credentials.NewChainCredentials(providers)
		if err != nil {
			log.Println(fmt.Errorf("Could not set STS credentials: %s", err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		creds, err := sts.Get()
		if err != nil {
			log.Println(fmt.Errorf("Could not get STS credentials: %s", err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		//fmt.Println(creds)

		response := make(map[string]interface{})
		response["credentials"] = creds
		c, err := json.MarshalIndent(response, "", "\t")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(c)

		sigint <- 1

	})

	address := fmt.Sprintf("localhost:3128")
	urlBrowse := fmt.Sprintf("http://%s/", address)
	log.Printf("listening on http://%s/", address)
	err = browser.OpenURL(urlBrowse)
	if err != nil {
		panic(err)
	}

	srv := &http.Server{Addr: address}

	idleConnsClosed := make(chan struct{})
	go func() {
		<-sigint

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed

	//log.Fatal(http.ListenAndServe(address, nil))

	err = core.MountVolume(instance, remote, local, "/home/dciangot/.config/rclone/rclone.conf")
	if err != nil {
		panic(err)
	}

}
