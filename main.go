package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/oauth2"

	"github.com/minio/minio-go/v6/pkg/credentials"

	"github.com/dciangot/sts-wire/pkg/core"
	iamTmpl "github.com/dciangot/sts-wire/pkg/template"
	_ "github.com/go-bindata/go-bindata"
	"github.com/pkg/browser"
)

type RCloneStruct struct {
	Address  string
	Instance string
}
type IAMCreds struct {
	AccessToken  string
	RefreshToken string
}

func main() {

	credsIAM := IAMCreds{}
	sigint := make(chan int, 1)

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

	confDir := "." + instance

	_, err := os.Stat(confDir)
	if os.IsNotExist(err) {
		os.Mkdir(confDir, os.ModePerm)
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
		ConfDir:      confDir,
		ClientConfig: clientConfig,
		Scanner:      scanner,
		HTTPClient:   *httpClient,
	}

	endpoint, clientResponse, _, err := clientIAM.InitClient(instance)
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

		credsIAM.AccessToken = token
		credsIAM.RefreshToken = oauth2Token.Extra("refresh_token").(string)

		err = ioutil.WriteFile(".token", []byte(token), 0600)
		if err != nil {
			log.Println(fmt.Errorf("Could not save token file: %s", err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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

	confRClone := RCloneStruct{
		Address:  "https://131.154.97.121:9001",
		Instance: instance,
	}

	tmpl, err := template.New("client").Parse(iamTmpl.RCloneTemplate)
	if err != nil {
		panic(err)
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, confRClone)
	if err != nil {
		panic(err)
	}

	rclone := b.String()

	err = ioutil.WriteFile(confDir+"/"+"rclone.conf", []byte(rclone), 0600)
	if err != nil {
		panic(err)
	}

	go core.MountVolume(instance, remote, local, confDir)

	// TODO: start routine to keep token valid!

	for {
		v := url.Values{}

		v.Set("client_id", clientResponse.ClientID)
		v.Set("client_secret", clientResponse.ClientSecret)
		v.Set("grant_type", "refresh_token")
		v.Set("refresh_token", credsIAM.RefreshToken)

		url, err := url.Parse(endpoint + "/token" + "?" + v.Encode())

		req := http.Request{
			Method: "POST",
			URL:    url,
		}

		// TODO: retrieve token with https POST with t.httpClient
		r, err := httpClient.Do(&req)
		if err != nil {
			panic(err)
		}
		//fmt.Println(r.StatusCode, r.Status)

		var bodyJSON core.RefreshTokenStruct

		rbody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}

		//fmt.Println(string(rbody))
		err = json.Unmarshal(rbody, &bodyJSON)
		if err != nil {
			panic(err)
		}

		// TODO:
		//encrToken := core.Encrypt([]byte(bodyJSON.AccessToken, passwd)

		err = ioutil.WriteFile(".token", []byte(bodyJSON.AccessToken), 0600)
		if err != nil {
			panic(err)
		}

		time.Sleep(10 * time.Minute)
	}

}
