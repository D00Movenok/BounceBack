package ipapicom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

var (
	ErrReservedRange = errors.New("reserved range")
	ErrPrivateRange  = errors.New("private range")
)

type Client interface {
	GetLocationForIP(ctx context.Context, ip string) (*Location, error)
}

// StandardURL is the primary URL.
const StandardURL = "http://ip-api.com"

// ProURL is the pro URL for api key.
const ProURL = "https://pro.ip-api.com"

func NewClient() Client {
	return &client{
		FmtURL:     fmt.Sprintf("%s/json/%%s", StandardURL),
		HTTPClient: &http.Client{},
	}
}

func NewClientWithAPIKey(apiKey string) Client {
	return &client{
		FmtURL:     fmt.Sprintf("%s/json/%%s?key=%s", ProURL, apiKey),
		HTTPClient: &http.Client{},
	}
}

// Location contains all the relevant data for an IP.
type Location struct {
	As          string  `json:"as"`
	City        string  `json:"city"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Isp         string  `json:"isp"`
	Lat         float32 `json:"lat"`
	Lon         float32 `json:"lon"`
	Message     string  `json:"message"`
	Org         string  `json:"org"`
	Query       string  `json:"query"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	Status      string  `json:"status"`
	Timezone    string  `json:"timezone"`
	Zip         string  `json:"zip"`
}

type client struct {
	FmtURL     string
	HTTPClient *http.Client
}

// GetLocationForIp retrieves the supplied IP address's location information.
func (c *client) GetLocationForIP(
	ctx context.Context,
	ip string,
) (*Location, error) {
	return getLocation(ctx, c.FmtURL, ip, c.HTTPClient)
}

func getLocation(
	ctx context.Context,
	fmtURL string,
	ip string,
	httpClient *http.Client,
) (*Location, error) {
	url := fmt.Sprintf(fmtURL, ip)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("can't create http request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't make http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read http response body: %w", err)
	}

	var l Location
	err = json.Unmarshal(body, &l)
	if err != nil {
		return nil, fmt.Errorf("can't parse json answer \"%s\": %w", body, err)
	}

	if resp.StatusCode != http.StatusOK || l.Status != "success" {
		switch l.Message {
		case ErrReservedRange.Error():
			return nil, fmt.Errorf(
				"can't catch ip geolocation: %w",
				ErrReservedRange,
			)
		case ErrPrivateRange.Error():
			return nil, fmt.Errorf(
				"can't catch ip geolocation: %w",
				ErrPrivateRange,
			)
		default:
			return nil, fmt.Errorf(
				"can't catch ip geolocation: %s",
				l.Message,
			)
		}
	}

	return &l, nil
}
