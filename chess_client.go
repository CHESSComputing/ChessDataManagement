package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

// helper function to exit
func exit(msg string, err error) {
	log.Fatal(fmt.Sprintf("%s, error: %v", msg, err))
	os.Exit(1)
}

// helper function to get kerberos ticket
func getKerberosTicket(krbFile string) []byte {
	//     fmt.Printf("krbFile: '%s'\n", krbFile)
	if krbFile != "" {
		ticket, err := ioutil.ReadFile(krbFile)
		if err != nil {
			exit("unable to read kerberos credentials", err)
		}
		return ticket
	}
	fname := fmt.Sprintf("krb5_%d_%v", os.Getuid(), time.Now().Unix())
	tmpFile, err := ioutil.TempFile("/tmp", fname)
	if err != nil {
		exit("Unabel to get tmp file", err)
	}
	//     defer os.Remove(tmpFile.Name())

	// get user password
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your kerberos password: ")
	password, _ := reader.ReadString('\n')

	cname := fmt.Sprintf("KRB5CCNAME=%s", tmpFile.Name())
	cmd := exec.Command("kinit", "-c", cname)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		exit("prompt", err)
	}
	defer stdin.Close()
	err = cmd.Start()
	if err != nil {
		exit("unable to run kinit", err)
	}
	io.WriteString(stdin, password)
	cmd.Wait()
	ticket, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		exit("unable to read kerberos credentials", err)
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
		exit("unable to read config file", err)
	}
	form := getForm(krbFile)
	form.Add("config", string(config))
	url := fmt.Sprintf("%s/api", uri)
	req, err := http.NewRequest("POST", url, strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	client := http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		exit(fmt.Sprintf("requset fails with status: %v", resp.Status), nil)
	}
	response, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(response))
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
		exit("find records method fails", err)
	}
	client := http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		exit(fmt.Sprintf("requset fails with status: %v", resp.Status), nil)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		exit(fmt.Sprintf("read response body failure, error: %v", resp.Status), nil)
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
