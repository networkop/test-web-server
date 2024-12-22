package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type payload struct {
	Hostname string `json:"hostname,omitempty"`
	Server   string `json:"server,omitempty"`
	IP       string `json:"ip"`
	Private  string `json:"private,omitempty"`
	ClientIP string `json:"clientIP,omitempty"`
}

const (
	maxUploadFileSize = 1024 * 1024 * 1024 // 1GB
	downloadSizeParam = "sizeMB"
	crawlerList       = "crawl"
)

var (
	response       payload
	getPublicIPURL = "http://ifconfig.co/json"
	output         = `%s
	Hostname: %s
	Public IP: %s
	Private IP: %s
	Client IP: %s`
	listenAddress = flag.String("listen", "0.0.0.0:8080", "echo server listening port")
	payloadSize   = flag.Int("payload", 1024, "Size of download payload in KB (default: 1MB")
	filename      = "test-download"
)

func downloadBytes(k int) int {
	return 1024 * k
}

func getPublicIP() {

	resp, err := http.Get(getPublicIPURL)
	if err != nil {
		log.Printf("Failed to GET ifconfig.co/json: %s", err)
		response.IP = ""
		return
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %s", err)
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Printf("Failed to unmarshall response: %s", err)
	}

}

func getPrivateIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return ""
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String()
		}
	}
	return ""
}

func handler(w http.ResponseWriter, r *http.Request) {

	log.Printf("Received request from %s for %s", r.RemoteAddr, r.Host)

	switch r.Method {
	case "GET":

		toCrawl := r.URL.Query().Get(crawlerList)
		if toCrawl != "" {
			data, err := base64.StdEncoding.DecodeString(toCrawl)
			if err != nil {
				fmt.Println(err)
				return
			}
			for _, host := range strings.Split(string(data), ",") {

				u := url.URL{
					Scheme: "https",
					Host:   host,
				}

				urlString, _ := url.PathUnescape(u.String())
				urlString = strings.TrimSuffix(urlString, "\n")

				log.Printf("Crawling %s", urlString)
				client := http.Client{
					Timeout: 1 * time.Second,
				}
				_, err := client.Get(urlString)
				if err != nil {
					log.Printf("Failed to crawl %s: %s", urlString, err)
					continue
				}

			}
		}
		switch r.Header.Get("Accept") {

		case "application/json":
			response.ClientIP = r.RemoteAddr
			j, _ := json.Marshal(response)
			w.Write(j)

		default:
			w.Write([]byte(fmt.Sprintf(output, response.Server, response.Server, response.IP, response.Private, r.RemoteAddr)))
		}

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

	go getPublicIP()

}

func jsonHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("Received request from %s for %s/json", r.RemoteAddr, r.Host)

	switch r.Method {
	case "GET":
		{
			response.ClientIP = r.RemoteAddr
			j, _ := json.Marshal(response)
			w.Write(j)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

	go getPublicIP()

}

func downloader(i int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {
		case "GET":

			sizeMB := r.URL.Query().Get(downloadSizeParam)
			if sizeMB != "" {
				s, err := strconv.Atoi(sizeMB)
				if err != nil {
					http.Error(w, fmt.Sprintf("Incorrect value of 'sizeMB' param: %s", sizeMB), http.StatusBadRequest)
					return
				}
				i = s * 1024
			}

			size := downloadBytes(i)
			file := make([]byte, size)
			rand.Read(file)
			log.Printf("Downloading file of size %d MB", size/1024/1024)

			FileContentType := http.DetectContentType(file)

			w.Header().Set("Content-Disposition", "attachment; filename="+filename)
			w.Header().Set("Content-Type", FileContentType)
			w.Header().Set("Content-Length", strconv.Itoa(size))
			binary.Write(w, binary.BigEndian, file)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}

	}
}

func uploader(i int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			http.Error(w, "Incorrect combination of path and method: /upload and GET", http.StatusBadRequest)
			return
		case "POST":
			r.Body = http.MaxBytesReader(w, r.Body, int64(maxUploadFileSize))
			if err := r.ParseMultipartForm(int64(maxUploadFileSize)); err != nil {
				http.Error(w, "The uploaded file is too big. Please choose an file that's less than 1GB in size", http.StatusBadRequest)
				return
			}
			// FormFile returns the first file for the given key `myFile`
			// it also returns the FileHeader so we can get the Filename,
			// the Header and the size of the file
			file, handler, err := r.FormFile(filename)
			if err != nil {
				fmt.Println("Error Retrieving the File")
				fmt.Println(err)
				return
			}
			defer file.Close()
			fmt.Printf("File Size (MB): %+v\n", handler.Size/1024/1024)
		}
	}
}

func main() {
	log.Print("Starting Echo web server on " + *listenAddress)
	flag.Parse()

	host, err := os.Hostname()
	if err != nil {
		host = "Failed to read hostname"
	}
	log.Printf("Hostname: %s", host)
	response.Server = host
	response.Private = getPrivateIP()
	log.Printf("Server IP: %s", response.Private)

	go getPublicIP()

	http.HandleFunc("/", handler)
	http.HandleFunc("/json", jsonHandler)
	http.HandleFunc("/download", downloader(*payloadSize))
	http.HandleFunc("/upload", uploader(*payloadSize))

	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
