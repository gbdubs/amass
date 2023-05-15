package amass

import (
	"github.com/gbdubs/attributions"
	"github.com/gbdubs/verbose"
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

func (r *GetRequest) Get() (*GetResponse, error) {
	return r.get()
}

type Amasser struct {
	TotalMaxConcurrentRequests int
	AllowedErrorProportion     float64
	verbose.Verbose
}

func (a *Amasser) GetAll(reqs []*GetRequest) ([]*GetResponse, error) {
	return a.getAll(reqs)
}

func AmasserForTests() *Amasser {
	return &Amasser{
		TotalMaxConcurrentRequests: 1,
		Verbose:                    verbose.Empty(),
		AllowedErrorProportion:     .01,
	}
}
