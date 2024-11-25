# Echo web server


## Actions

1. curl -H http://{ip}:8080/

Returns standard server metadata, including:

  * hostname 
  * public IP (discovered via `ifconfig.co`)
  * private IP 
  * client IP (as seen by the web server)

2. curl -H 'Accept: application/json' http://{ip}:8080/

Returns the same server metadata in JSON format

3. curl "http://{ip}:8080/json"

Same as #2

4. curl "http://{ip}:8080/?crawl=$(echo 'google.com,aws.amazon.com' | base64)"

Same as #1 but also instructs the web server to crawl the list of hostnames, encoded as comma-separated base64 string. 

5. curl -O "http://{ip}:8080/download"

Download a 1MB file (with randomly generated content)

6. curl -O "http://{ip}:8080/download?sizeMB=1000"

Download a file of arbitrary size (max size is 1GB)

7. curl -v -F test-download=@download http://localhost:8080/upload

Upload a file to the webserver (the content is discarded)
