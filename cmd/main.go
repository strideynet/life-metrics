package main

import (
	"fmt"
	"github.com/AlekSi/applehealth"
	"github.com/AlekSi/applehealth/healthkit"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
)

const influxOrg = "org"
const influxBucket = "life_metrics"
const influxURL = "http://localhost:8086"

func run() error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	logger.Info("hello world")

	token := os.Getenv("INFLUXDB_TOKEN")
	client := influxdb2.NewClient(influxURL, token)
	logger.Info("InfluxDB client created")

	writeAPI := client.WriteAPI(influxOrg, influxBucket)
	defer client.Close()

	file := filepath.Join("export.zip")
	u, err := applehealth.NewUnmarshaler(file)
	if err != nil {
		return err
	}
	defer u.Close()

	// call Next() once to be able to read metadata
	var data healthkit.Data
	if data, err = u.Next(); err != nil {
		return err
	}

	logger.Info("Meta loaded", "meta", u.Meta())

	for {
		logger.Info("Data row", "data", data)
		switch data := data.(type) {
		case *healthkit.Record:
			tags := map[string]string{
				"type":        data.Type,
				"source_name": data.SourceName,
			}
			for _, md := range data.MetadataEntry {
				tags[fmt.Sprintf("metadata_%s", md.Key)] = md.Value
			}

			value, err := strconv.ParseFloat(data.Value, 64)
			if err != nil {
				return err
			}
			fields := map[string]interface{}{
				"value": value,
			}

			point := write.NewPoint(
				fmt.Sprintf("healthkit_%s", data.Type),
				tags,
				fields,
				data.EndDateTime(),
			)

			writeAPI.WritePoint(point)
		}

		if data, err = u.Next(); err != nil {
			break
		}
	}
	if err != io.EOF {
		return err
	}
	return nil
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
