package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/jcmturner/gokrb5.v7/client"
	"gopkg.in/jcmturner/gokrb5.v7/config"
	"gopkg.in/jcmturner/gokrb5.v7/credentials"
)

// define if we use dev mode
var devMode bool

// helper function to get ROOT certificate
func getCertificate() string {
	scrt := os.Getenv("X509_ROOT_CA")
	//     if scrt == "" {
	//         exit("X509_ROOT_CA environemt is not set", nil)
	//     }
	return scrt
}

// helper function to get https client
// for chess internal usage we'll not provide client's cert and
// we will not verify server certificate
func httpClient(servercrt string) *http.Client {
	var client *http.Client
	client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				ClientAuth:         tls.NoClientCert, // we will not send client's cert
				InsecureSkipVerify: true,             // we will not verify server cert
			},
		},
	}
	return client
}

// helper function to get https client, this is more secure version
// which will verify server certificate
func httpClient_orig(servercrt string) *http.Client {
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
					RootCAs: caCertPool,
				},
			},
		}
	} else {
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{},
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
		exit("Unable to get tmp file", err)
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
	if devMode {
		return []byte("krb5-token-dev")
	}
	if krbFile != "" {
		// read krbFile and check user credentials
		creds, err := kuserFromCache(krbFile)
		if err != nil {
			msg := fmt.Sprintf("getKerberosTicket unable to read %s", krbFile)
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
func placeRequest(schemaName, uri, fileName, krbFile string, verbose int) error {

	// if we'll pass yaml file we'll need to convert it to json
	// if we'll pass json data we should probably read it via
	// json.Unmarshal and use appropriate type structure from server
	record, err := ioutil.ReadFile(fileName)
	if err != nil {
		exit("unable to read config file", err)
	}
	form := getForm(krbFile)
	form.Add("record", string(record))
	form.Add("SchemaName", schemaName)
	rurl := fmt.Sprintf("%s/api", uri)
	req, err := http.NewRequest("POST", rurl, strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	//     req.Header.Add("Content-Type", "multipart/form-data")
	if verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		log.Printf("http request %+v, rurl %v, dump %v, error %v\n", req, rurl, string(dump), err)
	}
	servercrt := getCertificate()
	client := httpClient(servercrt)
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if verbose > 1 {
		if resp != nil {
			dump, err := httputil.DumpResponse(resp, true)
			log.Printf("http response rurl %v, dump %v, error %v\n", rurl, string(dump), err)
		}
	}
	if resp.StatusCode != 200 {
		exit(fmt.Sprintf("request fails with status: %v", resp.Status), nil)
	}
	response, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(response))
	return err
}

// helper function to look-up records in chess data management system
func findRecords(uri, query, krbFile string, verbose int) {
	form := getForm(krbFile)
	form.Add("query", string(query))
	form.Add("client", "cli")
	rurl := fmt.Sprintf("%s/search", uri)
	req, err := http.NewRequest("POST", rurl, strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		exit("find records method fails", err)
	}
	if verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		log.Printf("http request %+v, rurl %v, dump %v, error %v\n", req, rurl, string(dump), err)
	}
	servercrt := getCertificate()
	client := httpClient(servercrt)
	resp, err := client.Do(req)
	if err != nil {
		exit("Fail to place request", err)
	}
	defer resp.Body.Close()
	if verbose > 1 {
		if resp != nil {
			dump, err := httputil.DumpResponse(resp, true)
			log.Printf("http response rurl %v, dump %v, error %v\n", rurl, string(dump), err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		exit(fmt.Sprintf("request fails with status: %v", resp.Status), nil)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		exit(fmt.Sprintf("read response body failure, error: %v", resp.Status), nil)
	}
	fmt.Println(string(data))
}

// helper function to look-up records in chess data management system
func findFiles(uri string, did int64, krbFile string, verbose int) {
	form := getForm(krbFile)
	form.Add("did", fmt.Sprintf("%d", did))
	rurl := fmt.Sprintf("%s/files", uri)
	req, err := http.NewRequest("POST", rurl, strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		exit("find records method fails", err)
	}
	if verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		log.Printf("http request %+v, rurl %v, dump %v, error %v\n", req, rurl, string(dump), err)
	}
	servercrt := getCertificate()
	client := httpClient(servercrt)
	resp, err := client.Do(req)
	if err != nil {
		exit("Fail to place request", err)
	}
	defer resp.Body.Close()
	if verbose > 1 {
		if resp != nil {
			dump, err := httputil.DumpResponse(resp, true)
			log.Printf("http response rurl %v, dump %v, error %v\n", rurl, string(dump), err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		exit(fmt.Sprintf("request fails with status: %v", resp.Status), nil)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		exit(fmt.Sprintf("read response body failure, error: %v", resp.Status), nil)
	}
	fmt.Println(string(data))
}

func info() string {
	goVersion := runtime.Version()
	tstamp := time.Now()
	return fmt.Sprintf("git={{VERSION}} go=%s date=%s", goVersion, tstamp)
}

func main() {
	var schema string
	flag.StringVar(&schema, "schema", "", "schema name for your data")
	var query string
	flag.StringVar(&query, "query", "", "query string to look-up your data")
	var did int64
	flag.Int64Var(&did, "did", 0, "show files for given dataset-id")
	var record string
	flag.StringVar(&record, "insert", "", "insert record to the server")
	var krbFile string
	flag.StringVar(&krbFile, "krbFile", "", "kerberos file")
	flag.BoolVar(&devMode, "devMode", false, "run dev mode")
	var uri string
	flag.StringVar(&uri, "uri", "https://chessdata.classe.cornell.edu:8243", "CHESS Data Management System URI")
	var verbose int
	flag.IntVar(&verbose, "verbose", 0, "verbosity level")
	var version bool
	flag.BoolVar(&version, "version", false, "Show version")
	flag.Usage = func() {
		client := "chess_client"
		msg := fmt.Sprintf("\nCommand line interface to CHESS Data Management System\n")
		msg = fmt.Sprintf("%s\nObtain kerberos ticket:\nkinit -c krb5_ccache <username>\n", msg)
		msg = fmt.Sprintf("%s\nOptions:\n", msg)
		fmt.Fprintf(os.Stderr, msg)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n\n# inject new record into the system using lite schema")
		fmt.Fprintf(os.Stderr, "\n%s -krbFile krb5cc_ccache -insert record.json -schema lite", client)
		fmt.Fprintf(os.Stderr, "\n\n# look-up data from the system using free text-search")
		fmt.Fprintf(os.Stderr, "\n%s -krbFile krb5cc_ccache -query=\"search words\"", client)
		fmt.Fprintf(os.Stderr, "\n\n# look-up data from the system using keyword search")
		fmt.Fprintf(os.Stderr, "\n%s -krbFile krb5cc_ccache -query=\"proposal:123\"", client)
		fmt.Fprintf(os.Stderr, "\n\n# look-up files for specific dataset-id")
		fmt.Fprintf(os.Stderr, "\n%s -krbFile krb5cc_ccache -did=1570563920579312510\n", client)
	}
	flag.Parse()
	if version {
		fmt.Println("chess_client version:", info())
		return
	}
	// we only allow devMode for localhost testing
	if devMode {
		if !strings.HasPrefix(uri, "http://localhost") {
			msg := fmt.Sprintf("In dev mode we only allow access to localhost while uri=%s", uri)
			exit(msg, errors.New("invalid set of parameters"))
		}
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if krbFile == "" {
		ccname := os.Getenv("KRB5CCNAME")
		if ccname != "" {
			krbFile = strings.Replace(ccname, "FILE:", "", -1)
		}
	}
	if did > 0 {
		findFiles(uri, did, krbFile, verbose)
		return
	}
	if query != "" {
		findRecords(uri, query, krbFile, verbose)
		return
	}
	placeRequest(schema, uri, record, krbFile, verbose)
}
