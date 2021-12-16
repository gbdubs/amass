package amass

import (
	"github.com/gbdubs/attributions"
)

type GetRequest struct {
	Site                      string
	RequestKey                string
	MinVersion                int
	URL                       string
	SiteMaxConcurrentRequests int
	Attribution               attributions.Attribution
	RoundTripData             []byte
}

type GetResponse struct {
	Site          string
	RequestKey    string
	Version       int
	URL           string
	ResponseBody  string
	StatusCode    int
	Status        string
	Attribution   attributions.Attribution
	RoundTripData []byte
}

type Amasser struct {
	TotalMaxConcurrentRequests int
	Verbose                    bool
	AllowedErrorProportion     float64
}

// To call just once, use *GetRequest.Get() => *GetResponse, error

func (a *Amasser) GetAll(reqs []*GetRequest) ([]*GetResponse, error) {
	return a.getAll(reqs)
}
