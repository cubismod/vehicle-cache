package main

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

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
		log.Error().Err(err)
		return false, "", err
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
			log.Error().Err(err)
			return false, "", err
		}
	}

	f, err := os.Create(key)
	if err != nil {
		return false, "", err
	}
	defer f.Close()

	if _, err := io.CopyN(f, reader, stat.Size); err != nil {
		log.Error().Err(err)
		return false, "", err
	}

	log.Info().Msgf("%s downloaded, %d bytes\n", key, stat.Size)
	return true, stat.ETag, nil
}

func get_file_contents(filename string, ctx context.Context, client *minio.Client, bucket string) string {
	updated, _, err := download_s3_object(client, bucket, filename, ctx, "")
	if err != nil {
		log.Error().Err(err)
		return ""
	}
	if updated {
		fileContents, err := os.ReadFile(filename)
		if err != nil {
			log.Error().Err(err)
			return ""
		}
		return string(fileContents)
	}
	return ""
}

func check_updates(data *haxmap.Map[string, string], prefix string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shapesFile := fmt.Sprintf("%sshapes.json", prefix)
	alertsFile := fmt.Sprintf("%salerts.json", prefix)
	vehiclesFile := fmt.Sprintf("%svehicles.json", prefix)

	endpoint := os.Getenv("AWS_ENDPOINT_URL_S3")
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true,
	})

	if err != nil {
		log.Fatal().Err(err)
	}
	objEtag := ""

	bucket := os.Getenv("VT_S3_BUCKET")
	_, _, err = download_s3_object(client, bucket, shapesFile, ctx, objEtag)
	if err != nil {
		log.Fatal().Err(err)
	}
	shapes, err := os.ReadFile(shapesFile)
	if err != nil {
		log.Fatal().Err(err)
	}
	data.Set(shapesFile, string(shapes))

	runsSinceLastUpdate := 0
	for {
		vehicleUpdate := get_file_contents(vehiclesFile, ctx, client, bucket)
		if vehicleUpdate != "" {
			data.Set(vehiclesFile, string(vehicleUpdate))
			runsSinceLastUpdate = 0
		} else {
			runsSinceLastUpdate++
		}

		alertsUpdate := get_file_contents(alertsFile, ctx, client, bucket)
		if alertsUpdate != "" {
			data.Set(alertsFile, string(alertsUpdate))
		}

		if runsSinceLastUpdate >= 60 {
			data.Set(vehicleUpdate, "{\"type\": \"FeatureCollection\", \"features\": []}")
			time.Sleep(time.Duration(runsSinceLastUpdate) * time.Second)
		}
		time.Sleep(time.Second * 1)
	}
}

// cleans up all the .json files in local dir
func cleanup() {
	jsonFiles, err := filepath.Glob("*.json")
	if err != nil {
		return
	}
	for _, e := range jsonFiles {
		_ = os.Remove(e)
	}
}

func resolveKey(urlPath string) string {
	var routeMap = map[string]string{
		"/alerts":   "alerts",
		"/shapes":   "shapes",
		"/":         "vehicles",
		"/vehicles": "vehicles",
	}
	isDev := strings.HasPrefix(urlPath, "/dev")
	path := urlPath
	if isDev {
		// Remove "/dev" prefix for lookup
		path = strings.TrimPrefix(urlPath, "/dev")
		if path == "" {
			path = "/vehicles"
		}
	}
	if key, ok := routeMap[path]; ok {
		if isDev {
			return "dev_" + key
		}
		return key
	}
	return "vehicles" // default
}

func main() {
	flag.Parse()

	cleanup()
	debug := flag.Bool("debug", false, "sets log level to debug")

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	httpPort := os.Getenv("VT_HTTP_PORT")

	data := haxmap.New[string, string]()

	compress, err := httpcompression.DefaultAdapter()

	go check_updates(data, "")
	go check_updates(data, "dev_")

	errStr := "unable to load data"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for name, values := range r.Header {
			for _, value := range values {
				log.Debug().Msgf("{'%s': '%s'}", name, value)
			}
		}
		log.Info().Msgf("Received request: %s %s from %s\n",
			r.Method, r.URL.Path, r.RemoteAddr)

		key := resolveKey(r.URL.Path)
		val, ok := data.Get(fmt.Sprintf("%s.json", key))
		if !ok {
			log.Error().Msgf("unable to load %s", key)
			http.Error(w, "unable to load data", http.StatusInternalServerError)
			return
		}
		hash := sha256.New()
		valReader := strings.NewReader(val)
		_, err = io.Copy(hash, valReader)
		if err != nil {
			log.Error().Msgf("unable to load %s", key)
			http.Error(w, errStr, http.StatusInternalServerError)
			return
		}
		etag := hash.Sum(nil)

		w.Header().Set("ETag", fmt.Sprintf("%x", etag))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		_, err = io.WriteString(w, val)
		if err != nil {
			log.Error().Err(err)
			http.Error(w, errStr, http.StatusInternalServerError)
			return
		}

	})

	http.Handle("/", compress(handler))

	fmt.Printf("Server is running on port %s", httpPort)
	err = http.ListenAndServe(fmt.Sprintf(":%s", httpPort), nil)
	if err != nil {
		log.Fatal().Err(err)
	}
}
