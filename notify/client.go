/*
Copyright © 2022 Kyle Chadha @kylechadha
*/
package notify

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/inconshreveable/log15"
)

const baseURL = "https://www.recreation.gov/api"

type Client struct {
	client *http.Client
	log    log15.Logger
}

func NewClient(log log15.Logger) *Client {
	return &Client{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: log,
	}
}

func (c *Client) Do(path string, queryParams url.Values) (*http.Response, error) {
	url := baseURL + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = queryParams.Encode()

	// Need to spoof the user agent or CloudFront blocks us.
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.138 Safari/537.36")
	c.log.Debug("request is", "req", req)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("%s: %s", resp.Status, string(bytes))
	}
	return resp, nil
}

const searchEndpoint = "/search/suggest"

type SearchResponse struct {
	Campgrounds []Campground `json:"inventory_suggestions"`
}

func (c *Client) Search(query string) ([]Campground, error) {
	c.log.Debug("Searching for campgrounds", "query", query)

	qp := url.Values{}
	qp.Add("q", query)
	qp.Add("geocoder", "true")

	resp, err := c.Do(searchEndpoint, qp)
	if err != nil {
		return nil, fmt.Errorf("Search request failed: %w", err)
	}
	defer resp.Body.Close()

	var sr SearchResponse
	err = json.NewDecoder(resp.Body).Decode(&sr)
	if err != nil {
		return nil, err
	}

	// Filter non-campgrounds from the search results.
	end := 0
	for i, c := range sr.Campgrounds {
		if c.EntityType == "campground" {
			sr.Campgrounds[end] = sr.Campgrounds[i]
			sr.Campgrounds[end].Name = strings.Title(strings.ToLower(c.Name))
			end++
		}
	}

	sr.Campgrounds = sr.Campgrounds[:end]
	return sr.Campgrounds, nil
}

type GetResponse struct {
	Campgrounds []Campground `json:"results"`
}

//const getCampgroundEndpoint = "https://www.recreation.gov/api/search?fq=entity_type:campground&fq=entity_id:232890
func (c *Client) getCampground(id string) (*Campground, error) {
	c.log.Debug("Getting campground details", "id", id)

	qp := url.Values{}
	qp.Add("fq", "entity_type:campground")
	qp.Add("fq", "entity_id:"+id)

	resp, err := c.Do("/search", qp)
	if err != nil {
		return nil, fmt.Errorf("get request failed.", "err", err)
	}
	defer resp.Body.Close()

	var sr GetResponse
	err = json.NewDecoder(resp.Body).Decode(&sr)
	if err != nil {
		return nil, err
	}

	for i, c := range sr.Campgrounds {
		if c.EntityType == "campground" {
			return &sr.Campgrounds[i], nil
		}
	}
	return nil, nil
}

const availEndpoint = "/camps/availability/campground/%s/month"

type AvailabilityResponse struct {
	Campsites map[string]Campsite `json:"campsites"`
	Count     int                 `json:"count"`
}

func (c *Client) Availability(campgroundID string, month string) (map[string]Campsite, error) {
	c.log.Debug("Checking campground availability", "campgroundID", campgroundID, "month", month)

	path := fmt.Sprintf(availEndpoint, campgroundID)
	qp := url.Values{}
	qp.Add("start_date", month+"-01T00:00:00.000Z")

	resp, err := c.Do(path, qp)
	if err != nil {
		return nil, fmt.Errorf("Search request failed: %w", err)
	}
	defer resp.Body.Close()

	var ar AvailabilityResponse
	err = json.NewDecoder(resp.Body).Decode(&ar)
	if err != nil {
		return nil, err
	}

	return ar.Campsites, nil
}
