package amass

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func (h *GetResponse) AsDocument() *goquery.Document {
	res, err := goquery.NewDocumentFromReader(strings.NewReader(h.ResponseBody))
	if err != nil {
		panic(err)
	}
	return res
}

func (h *GetResponse) AsXMLObject(i interface{}) error {
	asBytes := []byte(h.ResponseBody)
	return xml.Unmarshal(asBytes, i)
}

func (r *GetRequest) SetRoundTripData(i interface{}) {
	b, err := xml.MarshalIndent(i, "", "  ")
	if err != nil {
		panic(fmt.Errorf("Couldn't marshal rtd %v: %v", i, err))
	}
	r.RoundTripData = b
}

func (r *GetResponse) GetRoundTripData(i interface{}) {
	err := xml.Unmarshal(r.RoundTripData, i)
	if err != nil {
		panic(fmt.Errorf("Couldn't unmarshal rtd %s: %v", string(r.RoundTripData), err))
	}
}
