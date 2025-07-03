package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"iter"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

var (
	port   *uint   = flag.Uint("port", 8080, "port to serve traffic on")
	host   *string = flag.String("unifi-host", "192.168.0.1", "host to query")
	apiKey *string = flag.String("api-key", "", "api key")
)

func main() {
	flag.Parse()

	d := &Discovery{Client: &unifi{
		host:   *host,
		apiKey: *apiKey,
	}}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), d))
}

// Discovery is an [http.Handler] that queries the Unifi Network API to list a set of targets for Grafana Alloy's [discovery.http] component.
//
// [discovery.http]: https://grafana.com/docs/alloy/latest/reference/components/discovery/discovery.http/
type Discovery struct {
	Client *unifi
}

// ServeHTTP serves the list of scrape targets.
func (d *Discovery) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var tt []target
	for s, err := range GetList[site](r.Context(), d.Client, d.Client.URL(path.Join("v1", "sites")).String()) {
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for dv, err := range GetList[device](r.Context(), d.Client, d.Client.URL(path.Join("v1", "sites", s.ID, "devices")).String()) {
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			tt = append(tt, target{
				Targets: []string{dv.IPAddress},
				Labels: map[string]string{
					"device_id":    dv.ID,
					"device_name":  dv.Name,
					"device_model": dv.Model,
				},
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(tt); err != nil {
		panic(err)
	}
}

// unifi API client.
type unifi struct {
	host   string
	apiKey string
}

// RoundTrip implements [http.RoundTripper].
func (u *unifi) RoundTrip(r *http.Request) (*http.Response, error) {
	inner := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if u.apiKey != "" {
		r.Header.Set("X-Api-Key", u.apiKey)
	}

	return inner.RoundTrip(r)
}

// target definition for Grafana Alloy
type target struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

// GetJSON an HTTP client for the Unifi Network API
func (u *unifi) Client() *http.Client {
	return &http.Client{
		Transport: u,
	}
}

// Do the HTTP request.
func (u *unifi) Do(req *http.Request) (*http.Response, error) {
	return u.Client().Do(req)
}

// URL creates an [url.URL] for the given endpoint.
func (u *unifi) URL(endpoint string) *url.URL {
	b := &url.URL{
		Scheme: "https",
		Host:   u.host,
		Path:   "/proxy/network/integration/",
	}

	return b.JoinPath(endpoint)
}

// GetList provides an iterator over endpoints that return lists.
func GetList[T any](ctx context.Context, client *unifi, u string) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var (
			offset, count, totalCount int
		)
		for {
			u2, err := url.Parse(u)
			if err != nil {
				var z T
				yield(z, err)
				return
			}
			q := u2.Query()
			q.Set("offset", strconv.Itoa(offset))
			u2.RawQuery = q.Encode()

			res, err := GetJSON[listResponse[[]T]](ctx, client, u2.String())
			if err != nil {
				var z T
				yield(z, err)
				return
			}

			offset, count, totalCount = res.Offset, res.Count, res.TotalCount
			for _, s := range res.Data {
				if !yield(s, nil) {
					return
				}
			}

			if (offset + count) >= totalCount {
				return
			}
			offset = offset + count
		}
	}
}

// GetJSON fetches a JSON-formatted response.
func GetJSON[T any](ctx context.Context, client *unifi, u string) (T, error) {
	var o T

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return o, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return o, fmt.Errorf("%s %s: %w", req.Method, u, err)
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&o); err != nil {
		return o, fmt.Errorf("decoding response: %w", err)
	}

	return o, nil
}

// listResponse is the common list output for the Unifi API.
type listResponse[T any] struct {
	Offset     int `json:"offset"`
	Limit      int `json:"limit"`
	Count      int `json:"count"`
	TotalCount int `json:"totalCount"`
	Data       T   `json:"data"`
}

// site response.
type site struct {
	ID string `json:"id"`
}

// device response.
type device struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Model      string `json:"model"`
	MacAddress string `json:"macAddress"`
	State      string `json:"state"`
	IPAddress  string `json:"ipAddress"`
}
