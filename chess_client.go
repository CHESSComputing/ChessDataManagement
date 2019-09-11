package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// helper function to get kerberos ticket
func getKerberosTicket(krbFile string) []byte {
	ticket, err := ioutil.ReadFile(krbFile)
	if err != nil {
		msg := fmt.Sprintf("unable to read kerberos credentials, error: %v", err)
		log.Fatal(msg)
		os.Exit(1)
	}
	return ticket
}

// helper function to get url.Values form populated with kerberos info
func getForm(krbFile string) url.Values {
	ticket := getKerberosTicket(krbFile)
	arr := strings.Split(krbFile, "/")
	fname := arr[len(arr)-1]
	form := url.Values{}
	form.Add("ticket", string(ticket))
	form.Add("name", fname)
	return form
}

// helper function to place request to chess data management system
func placeRequest(uri, configFile, krbFile string) error {

	// if we'll pass yaml file we'll need to convert it to json
	// if we'll pass json data we should probably read it via
	// json.Unmarshal and use appropriate type structure from server
	config, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}
	form := getForm(krbFile)
	form.Add("config", string(config))
	url := fmt.Sprintf("%s/api", uri)
	req, err := http.NewRequest("POST", url, strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	client := http.Client{}
	resp, err := client.Do(req)
	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("requset fails with status: %v", resp.Status)
		log.Fatal(msg)
		os.Exit(1)
	}
	return err
}

// helper function to look-up records in chess data management system
func findRecords(uri, query, krbFile string) {
	form := getForm(krbFile)
	form.Add("query", string(query))
	url := fmt.Sprintf("%s/search", uri)
	req, err := http.NewRequest("POST", url, strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		msg := fmt.Sprintf("find records method fails with error: %v", err)
		log.Fatal(msg)
		os.Exit(1)
	}
	client := http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("requset fails with status: %v", resp.Status)
		log.Fatal(msg)
		os.Exit(1)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg := fmt.Sprintf("read response body failure, error: %v", resp.Status)
		log.Fatal(msg)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func main() {
	var query string
	flag.StringVar(&query, "query", "", "query string to look-up your data")
	var jsonConfig string
	flag.StringVar(&jsonConfig, "json", "", "json configuration file to inject")
	var krbFile string
	flag.StringVar(&krbFile, "krbFile", "", "kerberos file")
	var uri string
	flag.StringVar(&uri, "uri", "https://chessdata.lns.cornell.edu:8243", "CHESS Data Management System URI")
	flag.Usage = func() {
		client := "chess_client"
		msg := fmt.Sprintf("\nCommand line interface to CHESS Data Management System\nOptions:\n")
		fmt.Fprintf(os.Stderr, msg)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n\n# inject new configuration into the system")
		fmt.Fprintf(os.Stderr, "\n%s -krbFile /tmp/krb5cc_%d -json config.json", client, os.Getuid())
		fmt.Fprintf(os.Stderr, "\n\n# look-up some data from the system")
		fmt.Fprintf(os.Stderr, "\n%s -krbFile /tmp/krb5cc_%d -query \"search words\"\n", client, os.Getuid())
	}
	flag.Parse()
	if query != "" {
		findRecords(uri, query, krbFile)
		return
	}
	placeRequest(uri, jsonConfig, krbFile)
}
