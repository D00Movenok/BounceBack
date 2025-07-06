package ipapico

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	ErrReservedRange = errors.New("reserved ip address")
)

type Client interface {
	GetLocationForIP(ctx context.Context, ip string) (*Location, error)
}

// StandardURL is the primary URL.
const StandardURL = "https://ipapi.co"

func NewClient() Client {
	return &client{
		FmtURL:     fmt.Sprintf("%s/%%s/json/", StandardURL),
		HTTPClient: &http.Client{},
	}
}

func NewClientWithAPIKey(apiKey string) Client {
	return &client{
		FmtURL:     fmt.Sprintf("%s/%%s/json/?key=%s", StandardURL, apiKey),
		HTTPClient: &http.Client{},
	}
}

// Location contains all the relevant data for an IP.
type Location struct {
	IP                 string  `json:"ip"`
	City               string  `json:"city"`
	Region             string  `json:"region"`
	RegionCode         string  `json:"region_code"`
	Country            string  `json:"country"`
	CountryName        string  `json:"country_name"`
	ContinentCode      string  `json:"continent_code"`
	InEu               bool    `json:"in_eu"`
	Postal             string  `json:"postal"`
	Latitude           float32 `json:"latitude"`
	Longitude          float32 `json:"longitude"`
	Timezone           string  `json:"timezone"`
	UtcOffset          string  `json:"utc_offset"`
	CountryCallingCode string  `json:"country_calling_code"`
	Currency           string  `json:"currency"`
	Languages          string  `json:"languages"`
	Asn                string  `json:"asn"`
	Org                string  `json:"org"`
	IsError            bool    `json:"error"`
	Reason             string  `json:"reason"`
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

	// for unknown reason they bans default useragent
	req.Header.Set("User-Agent", "ipapi.co/#go")

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

	if resp.StatusCode != http.StatusOK || l.IsError {
		switch strings.ToLower(l.Reason) {
		case ErrReservedRange.Error():
			return nil, fmt.Errorf(
				"can't catch ip geolocation: %w",
				ErrReservedRange,
			)
		default:
			return nil, fmt.Errorf(
				"can't catch ip geolocation: %s",
				l.Reason,
			)
		}
	}

	return &l, nil
}
