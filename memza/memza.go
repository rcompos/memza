package memza

// Memza will upload and download files to Memcache
// If the file is larger than 1MB, it is chunked
// When retrieved, the chunks are reconstituted
// Maximum file size is maxFileSize

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/bradfitz/gomemcache/memcache"
)

const devilsBytes int = 62
const valueSizeMax int = 1024*1024 - devilsBytes

var MemcachedServer string

//var HttpURL string

// RetrieveFile get file contents for given key filename
func RetrieveFile(f, mserver, outFile string, dbug bool) ([]byte, error) {

	// Get number of required chunks for file
	filehash, num, errGet := getter(mserver, f, dbug)
	if errGet != nil {
		return []byte{}, errGet
	}

	if dbug == true {
		fmt.Printf("RetrieveFile ->\n")
		fmt.Printf("Key: %s\n", f)
		fmt.Printf("Chunks: %v\n", num)
		fmt.Printf("Filehash: %x\n", string(filehash))
	}

	// Open file
	file, errCreate := os.Create(outFile)
	if errCreate != nil {
		return []byte{}, errCreate
	}
	defer file.Close()

	// Reconstitute
	for i := 1; i <= int(num); i++ {
		chunkKey := f + "-" + strconv.Itoa(i)
		// Get single chunk
		chunkItem, _, err := getter(mserver, chunkKey, dbug)
		if err != nil {
			return []byte{}, err
		}
		// Write file
		n2, werr := file.Write(chunkItem)
		if dbug == true {
			fmt.Printf("chunkKey: %s\n", chunkKey)
			fmt.Printf("\tchunk: %v\n", i)
			fmt.Printf("\tBytes written: %d\n", n2)

			if werr != nil {
				return []byte{}, werr
			}
		}
	}

	// Read newly created file
	data, errRead := ioutil.ReadFile(outFile)
	if errRead != nil {
		return []byte{}, errRead
	}

	// Delete file
	errRemove := os.Remove(outFile)
	if errRemove != nil {
		fmt.Printf("error: %v\n", errRemove)
	}

	// Hash the file and output results
	newHash := sha256.Sum256(data)

	if dbug == true {
		fmt.Printf("%s: sha256sum: %x\n", outFile, newHash)
	}

	//badHash := []byte{'1', '9', 'a', 'f'} // For TESTING ONLY

	compareResult := bytes.Compare(filehash[:], newHash[:])
	var err error
	if compareResult != 0 {
		err = errors.New("hash mismatch")
	}

	//fmt.Printf("%s", data)

	return data, err

}

// StoreFile key: filename, value: file contents
func StoreFile(f, mserver string, max int64, dbug, force bool) ([32]byte, error) {

	bufferSize := valueSizeMax - len(f)

	if _, err := os.Stat(f); os.IsNotExist(err) {
		// file does not exist
		return [32]byte{}, err
	}

	// Get number of required chunks for file
	num, shasum, err := numChunks(f, bufferSize, max, dbug)
	if err != nil {
		return [32]byte{}, err
	}

	if dbug == true {
		fmt.Printf("StoreFile->\n")
		fmt.Printf("\tKey: %s\n", f)
		fmt.Printf("\tValue: %x\n", shasum)
		fmt.Printf("chunks: %v\n", num)
		fmt.Printf("sha256sum: %x\n", shasum)
		fmt.Printf("Setting item:\n")
	}

	// Set key named after file with value of shasum
	errSetterFile := setter(mserver, f, shasum[:], uint32(num), 0, dbug, force)
	if errSetterFile != nil {
		return [32]byte{}, errSetterFile
	}

	// Open file
	file, errOpen := os.Open(f)
	if errOpen != nil {
		return [32]byte{}, errOpen
	}
	defer file.Close()

	buffer := make([]byte, bufferSize)
	var i int = 1
	for {
		bytesread, err := file.Read(buffer)
		if err != nil {
			if err != io.EOF {
				return [32]byte{}, err
			}
			break
		}
		buff := buffer[:bytesread]

		// Set file contents
		fileKey := f + "-" + strconv.Itoa(i)

		if dbug == true {
			fmt.Printf("\tChunk: %v\n", i)
			fmt.Println("\tBytes read: ", bytesread)
			fmt.Printf("\tKey: %v\n", fileKey)
		}

		errSet := setter(mserver, fileKey, buff, 0, 0, dbug, force)
		if errSet != nil {
			return [32]byte{}, err
		}

		i++
	}

	fmt.Printf("key: %s\n", f)
	fmt.Printf("sha256sum: %x\n", shasum)
	return shasum, err

}

// setter is for setting mcache values
func setter(mserver, key string, val []byte, fla uint32, exp int32, dbug, force bool) error {
	mc := memcache.New(mserver)

	// Check for pre-existing key
	_, _, errGet := getter(mserver, key, dbug)
	if errGet == nil && force != true {
		return errors.New("key exists")
	}

	// Set key
	//fmt.Printf("Set key -> %s\tvalue: %s\tflag: %d\texp: %d\n", key, val, fla, exp)
	err := mc.Set(&memcache.Item{Key: key, Value: val, Flags: fla, Expiration: exp})
	if dbug == true {
		fmt.Printf("SETTER> %v\n", err)
	}
	return err
}

// getter is for getting mcache values
func getter(mserver, key string, dbug bool) ([]byte, uint32, error) {
	mc := memcache.New(mserver)
	// Get key
	if dbug == true {
		fmt.Printf("Get key -> %s\n", key)
	}
	myitem, err := mc.Get(key)
	if err != nil {
		return []byte{}, 0, err
	}
	return myitem.Value, myitem.Flags, err
}

// CheckServer memcached server status
func CheckServer(memcachedServer string) error {

	fmt.Println("Memza->CheckServer->")

	//mc := memcache.New("10.0.0.1:11211", "10.0.0.2:11211", "10.0.0.3:11212")
	mc := memcache.New(memcachedServer)

	// Check connection to memcached server
	fmt.Printf("Ping memcached server\n")
	errPing := mc.Ping()
	if errPing != nil {
		fmt.Printf("ping failed!\n")
		fmt.Printf("ERROR: %v", errPing)
	}
	fmt.Printf("ping successfull!\n")

	// Set key
	keyIn := "foo"
	valIn := "baarrr"
	fmt.Printf("Set Item\n")
	fmt.Printf("Set key -> %s\tvalue: %s\n", keyIn, valIn)
	mc.Set(&memcache.Item{Key: keyIn, Value: []byte(valIn)})

	// Get key
	fmt.Printf("Get key ->\n")
	myitem, err := mc.Get("foo")
	if err != nil {
		fmt.Printf("ERROR: %v", err)
	}
	fmt.Printf("item: %v\n", myitem)
	fmt.Printf("key: %v\n", myitem.Key)
	fmt.Printf("value: %v\n", string(myitem.Value))
	fmt.Printf("flags: %v\n", myitem.Flags)
	fmt.Printf("expiration: %v\n", myitem.Expiration)

	return err

}

// numChunks determine number of chunks needed
func numChunks(fileName string, chunksize int, max int64, dbug bool) (int, [32]byte, error) {

	sizeBytes, errFS := fileSize(fileName)
	if errFS != nil {
		return 0, [32]byte{}, errFS
	}

	// Empty file check
	if sizeBytes == 0 {
		return 0, [32]byte{}, errors.New("zero file size")
	}

	// Max file size check
	if sizeBytes > max {
		fmt.Printf("Max size: %d\n", max)
		errMsg := fmt.Sprintf("ERROR: File too large: %d\n", sizeBytes)
		return 0, [32]byte{}, errors.New(errMsg)
	}

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return 0, [32]byte{}, err
	}

	fileSHA256 := sha256.Sum256(data)

	// Figure out how many 1MB chunks
	floatChunks := float64(sizeBytes) / float64(chunksize)

	intChunks := int(floatChunks)
	if floatChunks > float64(intChunks) {
		intChunks++
	}

	if dbug == true {
		fmt.Printf("File: %s\n", fileName)
		fmt.Printf("Size (bytes): %d\n", sizeBytes)
		fmt.Printf("SHA-256: %x\n", fileSHA256)
		fmt.Printf("Chunks (1MB) Float: %f\n", floatChunks)
		fmt.Printf("Chunks (1MB) Int: %d\n", intChunks)
	}

	return intChunks, fileSHA256, err

}

// HelpMe provides help usage message and exits
// Used by CLI
func HelpMe(msg string) {
	if msg != "" {
		fmt.Printf("%s\n\n", msg)
	}
	fmt.Println("Store file in memcached")
	fmt.Println("Supply file name i.e. /path/to/myfile.txt")
	flag.PrintDefaults()
	os.Exit(1)
}

// fileSize checks file size
func fileSize(f string) (int64, error) {
	fi, err := os.Stat(f)
	return fi.Size(), err
}

// Info handler displays http header
func Info(w http.ResponseWriter, r *http.Request) {
	//fmt.Fprintf(w, "URL.Path = %q\n", r.URL.Path)
	fmt.Fprintf(w, "%s %s %s\n", r.Method, r.URL, r.Proto)
	for k, v := range r.Header {
		fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
	}
	fmt.Fprintf(w, "Host = %q\n", r.Host)
	fmt.Fprintf(w, "RemoteAddr = %q\n", r.RemoteAddr)
	fmt.Fprintf(w, "MemcachedServer= %q\n", MemcachedServer)
	if err := r.ParseForm(); err != nil {
		log.Print(err)
	}
	for k, v := range r.Form {
		fmt.Fprintf(w, "Form[%q] = %q\n", k, v)
	}
}
