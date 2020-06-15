package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/dciangot/sts-wire/pkg/core"
	_ "github.com/go-bindata/go-bindata"
)

func main() {

	inputReader := *bufio.NewReader(os.Stdin)
	scanner := core.GetInputWrapper{
		Scanner: inputReader,
	}

	instance := os.Args[1]
	if instance == "-h" {
		fmt.Println("sts-wire <instance name> <s3 endpoint> <rclone remote path> <local mount point>")
		return
	}

	if len(os.Args) < 4 {
		fmt.Println("Invalid instance name, please type 'sts-wire -h'  for help")
		return
	}

	confDir := "." + instance

	_, err := os.Stat(confDir)
	if os.IsNotExist(err) {
		os.Mkdir(confDir, os.ModePerm)
	}

	s3Endpoint := os.Args[2]

	remote := os.Args[3]

	local := os.Args[4]
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
		panic(err)
	}

	if os.Getenv("REFRESH_TOKEN") != "" {
		clientResponse.ClientID = "8bed0f49-168f-4c0d-8862-b11af06f2916"
		clientResponse.ClientSecret = "G7yrGjR_qGcLWS44MadMUMj5xA9_bV1yRcFSdicUx9D0SeJVsGFfk0v5R0MPNT28gQWZ1QStwDe1r_8_xkeyDg"
	}

	fmt.Println(clientResponse.Endpoint)

	server := core.Server{
		Client:     clientIAM,
		Instance:   instance,
		S3Endpoint: s3Endpoint,
		RemotePath: remote,
		LocalPath:  local,
		Endpoint:   endpoint,
		Response:   clientResponse,
	}

	err = server.Start()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Server started and volume mounted in %s", local)
	fmt.Printf("To unmount you can see you PID in mount.pid file and kill it.")

}
