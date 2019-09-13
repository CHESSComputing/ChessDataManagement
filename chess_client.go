package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
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
	"syscall"
	"time"

	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/jcmturner/gokrb5.v7/client"
	"gopkg.in/jcmturner/gokrb5.v7/config"
	"gopkg.in/jcmturner/gokrb5.v7/credentials"
)

// helper function to get client/server certificates
func getCerticiates() (string, string, string) {
	ckey := os.Getenv("X509_USER_KEY")
	cert := os.Getenv("X509_USER_CERT")
	scrt := os.Getenv("X509_ROOT_CA")
	if ckey == "" || cert == "" {
		exit("Please setup your X509_USER_KEY and X509_USER_CERT environemt variables", nil)
	}
	return ckey, cert, scrt
}

// helper function to get https client
func httpClient(uckey, ucert, servercrt string) *http.Client {
	cert, err := tls.LoadX509KeyPair(ucert, uckey)
	if err != nil {
		log.Fatal(err)
	}

	var client *http.Client
	if servercrt != "" {
		caCert, err := ioutil.ReadFile(servercrt)
		if err != nil {
			exit("fail to read server certificates", nil)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:      caCertPool,
					Certificates: []tls.Certificate{cert},
				},
			},
		}
	} else {
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates: []tls.Certificate{cert},
				},
			},
		}
	}
	return client
}

// helper function to exit
func exit(msg string, err error) {
	log.Fatal(fmt.Sprintf("%s, error: %v", msg, err))
	os.Exit(1)
}

// helper function to return user and password
func userPassword() (string, string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("Enter Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println()
	password := string(bytePassword)

	return strings.TrimSpace(username), strings.TrimSpace(password)
}

// https://github.com/jcmturner/gokrb5/issues/7
func kuserFromCache(cacheFile string) (*credentials.Credentials, error) {
	cfg, err := config.Load("/etc/krb5.conf")
	ccache, err := credentials.LoadCCache(cacheFile)
	client, err := client.NewClientFromCCache(ccache, cfg)
	err = client.Login()
	if err != nil {
		return nil, err
	}
	return client.Credentials, nil

}

func userTicket() []byte {
	// get user login/password
	user, password := userPassword()
	fname := fmt.Sprintf("krb5_%d_%v", os.Getuid(), time.Now().Unix())
	tmpFile, err := ioutil.TempFile("/tmp", fname)
	if err != nil {
		exit("Unabel to get tmp file", err)
	}
	defer os.Remove(tmpFile.Name())

	cmd := exec.Command("kinit", "-c", tmpFile.Name(), user)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		msg := fmt.Sprintf("Fail to execute '%v'", cmd)
		exit(msg, err)
	}

	// start command execution
	err = cmd.Start()
	if err != nil {
		exit("unable to start kinit", err)
	}

	// write our input to our pipe for command
	io.WriteString(stdin, password)

	// explicitly close the input informing command that we're done
	stdin.Close()

	// wait for command to finish its execution
	cmd.Wait()

	// read tmp file content and return the ticket
	ticket, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil || len(ticket) == 0 {
		exit("unable to read kerberos credentials", err)
	}
	return ticket
}

// helper function to get kerberos ticket
func getKerberosTicket(krbFile string) []byte {
	if krbFile != "" {
		// read krbFile and check user credentials
		creds, err := kuserFromCache(krbFile)
		if err != nil {
			msg := fmt.Sprintf("unable to read %s", krbFile)
			exit(msg, err)
		}
		if creds.Expired() {
			exit("user credentials are expired, please obtain new/valid krb file", nil)
		}
		ticket, err := ioutil.ReadFile(krbFile)
		if err != nil {
			exit("unable to read kerberos credentials", err)
		}
		return ticket
	}
	return userTicket()
}

// helper function to get url.Values form populated with kerberos info
func getForm(krbFile string) url.Values {
	ticket := getKerberosTicket(krbFile)
	if ticket == nil {
		exit("unable to obtain valid kerberos ticket", nil)
	}
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
	//     ckey, cert, servercrt := getCerticiates()
	//     client := httpClient(ckey, cert, servercrt)
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
	//     ckey, cert, servercrt := getCerticiates()
	//     client := httpClient(ckey, cert, servercrt)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		exit("Fail to place request", err)
	}
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
	flag.StringVar(&uri, "uri", "http://chessdata.lns.cornell.edu:8243", "CHESS Data Management System URI")
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
