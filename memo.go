package amass

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func (req *GetRequest) readMemo() (resp *GetResponse, isMissing bool, err error) {
	resp = &GetResponse{}
	mp := req.memoPath()
	_, err = os.Stat(mp)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			isMissing = true
			err = nil
		}
		return
	}
	f, err := ioutil.ReadFile(mp)
	if err != nil {
		return
	}
	err = xml.Unmarshal(f, resp)
	if err != nil {
		return
	}
	if req.MinVersion > resp.Version {
		err = os.Remove(mp)
		if err != nil {
			return
		}
		isMissing = true
		return
	}
	return
}

func (resp *GetResponse) writeMemo() error {
	b, err := xml.MarshalIndent(resp, "", "  ")
	if err != nil {
		return err
	}
	mp := resp.memoPath()
	err = os.MkdirAll(filepath.Dir(mp), 0777)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(mp, b, 0777)
}

func (r *GetRequest) memoPath() string {
	return memoPath(r.Site, r.RequestKey)
}

func (r *GetResponse) memoPath() string {
	return memoPath(r.Site, r.RequestKey)
}

func memoPath(site string, requestKey string) string {
	if site == "" {
		panic(fmt.Errorf("Expected Site to be set"))
	}
	if requestKey == "" {
		panic(fmt.Errorf("Expected RequestKey to be set"))
	}
	return fmt.Sprintf("/memo/%s/%s.xml", site, requestKey)
}
