package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"text/template"

	iamTmpl "github.com/dciangot/sts-wire/pkg/template"
)

type InitClientConfig struct {
	ClientConfig IAMClientConfig
	Scanner      GetInputWrapper
	HTTPClient   http.Client
}

func (t *InitClientConfig) InitClient(instance string) (endpoint string, clientResponse ClientResponse, err error) {
	rbody, err := ioutil.ReadFile("." + instance + ".json")
	if err != nil {

		tmpl, err := template.New("client").Parse(iamTmpl.ClientTemplate)
		if err != nil {
			panic(err)
		}

		var b bytes.Buffer
		err = tmpl.Execute(&b, t.ClientConfig)
		if err != nil {
			panic(err)
		}

		request := b.String()

		fmt.Println(request)

		contentType := "application/json"
		//body := url.Values{}
		//body. Set(request)

		endpoint, err = t.Scanner.GetInputString("Insert the IAM endpoint for the instance: ", "https://iam-demo.cloud.cnaf.infn.it")
		if err != nil {
			panic(err)
		}

		r, err := t.HTTPClient.Post(endpoint+"/register", contentType, strings.NewReader(request))
		if err != nil {
			panic(err)
		}

		fmt.Println(r.StatusCode, r.Status)

		rbody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(rbody))

		err = json.Unmarshal(rbody, &clientResponse)
		if err != nil {
			panic(err)
		}

		clientResponse.Endpoint = endpoint

		passwd, err := t.Scanner.GetPassword("Insert pasword for secrets encryption: ")
		if err != nil {
			panic(err)
		}
		dumpClient := Encrypt(rbody, passwd)

		err = ioutil.WriteFile("."+instance+".json", dumpClient, 0600)
		if err != nil {
			panic(err)
		}
	} else {
		passwd, err := t.Scanner.GetPassword("Insert pasword for secrets decryption: ")
		if err != nil {
			panic(err)
		}

		err = json.Unmarshal(Decrypt(rbody, passwd), &clientResponse)
		if err != nil {
			panic(err)
		}
		endpoint = strings.Split(clientResponse.Endpoint, "/register")[0]
	}

	if endpoint == "" {
		panic("Something went wrong. No endpoint selected")
	}

	return endpoint, clientResponse, nil
}
