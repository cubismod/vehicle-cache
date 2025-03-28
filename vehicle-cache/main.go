package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/CAFxX/httpcompression"
	"github.com/alphadose/haxmap"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
)

// downloads an object from S3
// returns boolean indicating if it changed, etag value, error
func download_s3_object(client *minio.Client, bucket string, key string, ctx context.Context, etag string) (bool, string, error) {
	reader, err := client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return false, "", err
	}
	defer reader.Close()

	stat, err := reader.Stat()
	if err != nil {
		log.Fatalln(err)
	}

	sf, err := os.Stat(key)
	if err == nil {
		if etag == stat.ETag || sf.ModTime() == stat.LastModified || sf.Size() == stat.Size {
			return false, "", nil
		}
	}
	if !errors.Is(err, fs.ErrNotExist) {
		err = os.Remove(key)
		if err != nil {
			return false, "", err
		}
	}

	f, err := os.Create(key)
	if err != nil {
		return false, "", err
	}
	defer f.Close()

	if _, err := io.CopyN(f, reader, stat.Size); err != nil {
		log.Fatalln(err)
	}

	log.Printf("%s downloaded, %d bytes\n", key, stat.Size)
	return true, stat.ETag, nil
}

func get_file_contents(filename string, ctx context.Context, client *minio.Client, bucket string) string {
	updated, _, err := download_s3_object(client, bucket, filename, ctx, "")
	if err != nil {
		log.Fatal(err)
	}
	if updated {
		fileContents, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		return string(fileContents)
	}
	return ""
}

func check_updates(data *haxmap.Map[string, string]) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	endpoint := os.Getenv("AWS_ENDPOINT_URL_S3")
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true,
	})

	if err != nil {
		log.Fatal(err)
	}
	objEtag := ""

	bucket := os.Getenv("VT_S3_BUCKET")
	_, _, err = download_s3_object(client, bucket, "shapes.json", ctx, objEtag)
	if err != nil {
		log.Fatal(err)
	}
	shapes, err := os.ReadFile("shapes.json")
	if err != nil {
		log.Fatal(err)
	}
	data.Set("shapes", string(shapes))
	runsSinceLastUpdate := 0
	for {
		vehicleUpdate := get_file_contents("vehicles.json", ctx, client, bucket)
		if vehicleUpdate != "" {
			data.Set("vehicles", string(vehicleUpdate))
			runsSinceLastUpdate = 0
		} else {
			runsSinceLastUpdate++
		}
		alertsUpdate := get_file_contents("alerts.json", ctx, client, bucket)
		if alertsUpdate != "" {
			data.Set("alerts", string(alertsUpdate))
		}

		if runsSinceLastUpdate >= 60 {
			data.Set("vehicles", "{\"type\": \"FeatureCollection\", \"features\": []}")
			time.Sleep(time.Duration(runsSinceLastUpdate) * time.Second)
		}
		time.Sleep(time.Second * 1)
	}
}

func main() {
	http_port := os.Getenv("VT_HTTP_PORT")

	data := haxmap.New[string, string]()

	compress, err := httpcompression.DefaultAdapter()

	go check_updates(data)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for name, values := range r.Header {
			for _, value := range values {
				log.Printf("{'%s': '%s'}", name, value)
			}
		}
		log.Printf("Received request: %s %s from %s\n",
			r.Method, r.URL.Path, r.RemoteAddr)

		key := "vehicles"
		if r.URL.Path == "/shapes" {
			key = "shapes"
		}

		if r.URL.Path == "/alerts" {
			key = "alerts"
		}

		val, ok := data.Get(key)
		if !ok {
			log.Fatalf("unable to load %s", key)
		}

		hash := sha256.New()
		valReader := strings.NewReader(val)
		_, err = io.Copy(hash, valReader)
		if err != nil {
			log.Fatal(err)
		}
		etag := hash.Sum(nil)

		w.Header().Set("ETag", fmt.Sprintf("%x", etag))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		_, err = io.WriteString(w, val)
		if err != nil {
			log.Fatal(err)
		}

	})

	http.Handle("/", compress(handler))

	fmt.Println(fmt.Sprintf("Server is running on port %s", http_port))
	err = http.ListenAndServe(fmt.Sprintf(":%s", http_port), nil)
	if err != nil {
		log.Fatal(err)
	}
}
