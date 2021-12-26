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

func (r *GetRequest) Get() (*GetResponse, error) {
	return r.get()
}

type Amasser struct {
	TotalMaxConcurrentRequests int
	Verbose                    bool
	AllowedErrorProportion     float64
}

func (a *Amasser) GetAll(reqs []*GetRequest) ([]*GetResponse, error) {
	return a.getAll(reqs)
}

func AmasserForTests() *Amasser {
	return &Amasser{
		TotalMaxConcurrentRequests: 1,
		Verbose:                    false,
		AllowedErrorProportion:     .01,
	}
}
