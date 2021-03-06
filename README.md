# Memza

Library that will accept a large file (somewhere between 5MB and 50MB) as an input and store it in Memcache.  The library will also retrieve the file from Memcache and return it.

HTTP API (REST, JSON, etc) to accept and store files posted from an HTTP Client (browser, curl). HTTP API will retrieve files with a followup request.

Requirement: Memcached server

```
⇒ docker container run -p 11211:11211 -d memcached  -m 1000
```

Requirement: Must set Memcached server name:port as command-line arg (-s) or environmental variable MEMCACHED\_SERVER\_URL.  If no Memcache server is specified, the local host is used.

```
⇒ export MEMCACHED_SERVER_URL=0.0.0.0:11211
	or
⇒ export MEMCACHED_SERVER_URL=192.168.1.128:11211
```

# Memza CLI

CLI Usage:

```
⇒ go run cmd/memza/main.go -h
  -d	Debug mode
  -f	Force key overwrite
  -g string
    	File to get
  -o string
    	Output file for retrieval (default "out.dat")
  -p string
    	File to put
  -s string
    	memcached_server:port (default "localhost:11211")
  -t	Check memcached server
```

Run on command-line without building binary:
Store a file on local Memcached:

```
⇒ go run cmd/memza-cli/main.go -p myfile.dat
```

Store a file on remote Memcached

```
⇒ go run cmd/memza-cli/main.go -s 192.168.1.128:11211 -p myfile.dat
```

Retrieve a file:

```
⇒ go run cmd/memza-cli/main.go -g myfile.dat

```

Alternatley, build a binary:

```
⇒ cd cmd/memza-cli
⇒ go build -o memza-cli

⇒ ./memza-cli -h
```

# Memza API

Build API service binary and run service in debugging mode (-d) with a server specifcation (-s).  

```
⇒ cd cmd/memza
⇒ go build -o memza
⇒ ./memza -d -s 192.168.1.128:11211
```

Alternately, run API service without building binary

```
⇒ go run cmd/memza-api/main.go
```

Endpoint for web browser file upload

```
http://localhost:8080/upload
```

Endpoint for web browser file download

```
http://localhost:8080/download
```

Curl command to post file:

```
⇒ curl -F 'file=@/path/to/myfile.dat' http://localhost:8080/receive
```

Curl command to get file

```
⇒ curl "http://localhost:8080/retrieve?key=/path/to/myfile.dat" --output out.dat
```

# Memza Docker

Build API service binary for Linux operating systems on AMD64 architecture.

Build Docker image from repo top-level directory

```
⇒ docker build -t memza .
```

Run Docker image with Memcached server as envvar

```
⇒ docker container run -d -p 8080:8080 --env MEMCACHED_SERVER_URL=192.168.1.128:11211 memza
```
