package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/devices/bmxx80"
	"periph.io/x/periph/host"
)

var addr = flag.String("listen-address", ":9529", "The address to listen on for HTTP requests.")
var dev *bmxx80.Dev

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	// Prepare metrics
	temperatureGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "bme280_temperature",
		Help: "Temperature in degrees Celsius",
	})
	humidityGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "bme280_humidity",
		Help: "Relative humidity %",
	})
	pressureGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "bme280_pressure",
		Help: "Pressure in hPa",
	})

	registry := prometheus.NewRegistry()
	registry.MustRegister(temperatureGauge)
	registry.MustRegister(humidityGauge)
	registry.MustRegister(pressureGauge)

	// Collect metrics
	var env physic.Env
	if err := dev.Sense(&env); err != nil {
		log.Fatal(err)
	}
	temperatureGauge.Set(env.Temperature.Celsius())
	humidityGauge.Set(float64(env.Humidity) / float64(physic.PercentRH))
	pressureGauge.Set(float64(env.Pressure) / float64(100*physic.Pascal))

	// Respond
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func main() {
	// Load all the drivers:
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Open a handle to the first available I²C bus:
	bus, err := i2creg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer bus.Close()

	// Open a handle to a bme280/bmp280 connected on the I²C bus using default
	// settings:
	dev, err = bmxx80.NewI2C(bus, 0x76, &bmxx80.DefaultOpts)
	if err != nil {
		log.Fatal(err)
	}
	defer dev.Halt()

	flag.Parse()

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metricsHandler(w, r)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>
	<head><title>BME280 Exporter</title></head>
	<body>
	<h1>BME280 Exporter</h1>
	<p>
	  <a href="/metrics">Metrics</a>
	</p>
	</body>
	</html>
	`))
	})

	log.Printf("Listening on %s\n", *addr)
	// The Handler function provides a default handler to expose metrics
	// via an HTTP server. "/metrics" is the usual endpoint for that.
	// http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(*addr, nil))
}
