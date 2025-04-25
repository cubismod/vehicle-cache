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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metrics struct {
	refreshCounter *prometheus.CounterVec
	httpRequests   *prometheus.CounterVec
}

func NewMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		refreshCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vc_refreshes",
			Help: "s3 refreshes by key",
		}, []string{"key"}),
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vc_http_reqs",
			Help: "http requests",
		}, []string{"path", "method", "status", "useragent"}),
	}
	reg.MustRegister(m.refreshCounter)
	reg.MustRegister(m.httpRequests)
	return m
}

// downloads an object from S3
// returns boolean indicating if it changed, etag value, error
func downloadS3Object(client *minio.Client, bucket string, key string, ctx context.Context, etag string, metrics *metrics) (bool, string, error) {
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
	metrics.refreshCounter.With(prometheus.Labels{"key": key}).Inc()
	return true, stat.ETag, nil
}

func getFileContents(filename string, ctx context.Context, client *minio.Client, bucket string, metrics *metrics) string {
	updated, _, err := downloadS3Object(client, bucket, filename, ctx, "", metrics)
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

type fileDef struct {
	envName string
	local   string
}

func checkUpdates(data *haxmap.Map[string, string], prefix string, metrics *metrics) {
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
		log.Fatal().Err(err)
	}
	bucket := os.Getenv("VT_S3_BUCKET")

	files := []fileDef{
		{"shapes", "shapes.json"},
		{"alerts", "alerts.json"},
		{"vehicles", "vehicles.json"},
	}

	// Preload shapes file.
	shapesFile := fmt.Sprintf("%s%s", prefix, files[0].local)
	if _, _, err = downloadS3Object(client, bucket, shapesFile, ctx, "", metrics); err != nil {
		log.Fatal().Err(err)
	}
	shapes, err := os.ReadFile(shapesFile)
	if err != nil {
		log.Fatal().Err(err)
	}
	data.Set(shapesFile, string(shapes))

	runsSinceLastUpdate := 0
	for {
		for _, f := range files[1:] {
			filePath := fmt.Sprintf("%s%s", prefix, f.local)
			update := getFileContents(filePath, ctx, client, bucket, metrics)
			if update != "" {
				data.Set(filePath, update)
				runsSinceLastUpdate = 0
			}
		}
		if runsSinceLastUpdate >= 60 {
			// Reset vehicles JSON if needed.
			vehiclesPath := fmt.Sprintf("%s%s", prefix, files[2].local)
			data.Set(vehiclesPath, "{\"type\": \"FeatureCollection\", \"features\": []}")
			time.Sleep(time.Duration(runsSinceLastUpdate) * time.Second)
		}
		runsSinceLastUpdate++
		time.Sleep(2 * time.Second)
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
	return "" // default
}

func main() {
	flag.Parse()

	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	cleanup()
	debug := flag.Bool("debug", false, "sets log level to debug")

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	httpPort := os.Getenv("VT_HTTP_PORT")

	data := haxmap.New[string, string]()

	compress, err := httpcompression.DefaultAdapter()

	go checkUpdates(data, "", m)
	go checkUpdates(data, "dev_", m)

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
			log.Error().Err(err)
			m.httpRequests.With(prometheus.Labels{"path": r.URL.Path, "method": r.Method, "status": "404", "useragent": r.UserAgent()}).Inc()
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		hash := sha256.New()
		valReader := strings.NewReader(val)
		_, err = io.Copy(hash, valReader)
		if err != nil {
			log.Error().Err(err)
			m.httpRequests.With(prometheus.Labels{"path": r.URL.Path, "method": r.Method, "status": "500", "useragent": r.UserAgent()}).Inc()
			http.Error(w, errStr, http.StatusInternalServerError)
			return
		}
		etag := hash.Sum(nil)

		w.Header().Set("ETag", fmt.Sprintf("%x", etag))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		m.httpRequests.With(prometheus.Labels{"path": r.URL.Path, "method": r.Method, "status": "200", "useragent": r.UserAgent()}).Inc()
		_, err = io.WriteString(w, val)
		if err != nil {
			log.Error().Err(err)
			m.httpRequests.With(prometheus.Labels{"path": r.URL.Path, "method": r.Method, "status": "500", "useragent": r.UserAgent()}).Inc()
			http.Error(w, errStr, http.StatusInternalServerError)
			return
		}

	})

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	http.Handle("/", compress(handler))

	fmt.Printf("Server is running on port %s", httpPort)
	err = http.ListenAndServe(fmt.Sprintf(":%s", httpPort), nil)
	if err != nil {
		log.Fatal().Err(err)
	}
}
