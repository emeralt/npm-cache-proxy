package proxy

import (
	"compress/gzip"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// GetCachedPath returns cached upstream response for a given url path.
func (proxy Proxy) GetCachedPath(path string, request *http.Request) ([]byte, error) {
	options, err := proxy.GetOptions()
	if err != nil {
		return nil, err
	}

	key := options.DatabasePrefix + path

	// get package from database
	pkg, err := proxy.Database.Get(key)

	// either package doesn't exist or there's some other problem
	if err != nil {

		// check if error is caused by nonexistend package
		// if no, return error
		if err.Error() != "redis: nil" {
			return nil, err
		}

		// error is caused by nonexistent package
		// fetch package
		req, err := http.NewRequest("GET", options.UpstreamAddress+path, nil)

		req.Header = request.Header
		req.Header.Set("Accept-Encoding", "gzip")

		res, err := proxy.HttpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		if res.Header.Get("Content-Encoding") == "gzip" {
			zr, err := gzip.NewReader(res.Body)
			if err != nil {
				log.Fatal(err)
			}

			res.Body = zr
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		// convert body to string
		pkg = string(body)

		// save to redis
		err = proxy.Database.Set(key, pkg, options.DatabaseExpiration)
		if err != nil {
			return nil, err
		}
	}

	// TODO: avoid calling MustCompile every time
	// find "dist": "https?://.*/ and replace to "dist": "{localurl}/
	pkg = regexp.MustCompile(`(?U)"tarball":"https?://.*/`).ReplaceAllString(pkg, `"dist": "http://localhost:8080/`)

	return []byte(pkg), nil
}

// ListCachedPaths returns list of all cached url paths.
func (proxy Proxy) ListCachedPaths() ([]string, error) {
	options, err := proxy.GetOptions()
	if err != nil {
		return nil, err
	}

	metadata, err := proxy.Database.Keys(options.DatabasePrefix)
	if err != nil {
		return nil, err
	}

	deprefixedMetadata := make([]string, 0)
	for _, record := range metadata {
		deprefixedMetadata = append(deprefixedMetadata, strings.Replace(record, options.DatabasePrefix, "", 1))
	}

	return deprefixedMetadata, nil
}

// PurgeCachedPaths deletes all cached url paths.
func (proxy Proxy) PurgeCachedPaths() error {
	options, err := proxy.GetOptions()
	if err != nil {
		return err
	}

	metadata, err := proxy.Database.Keys(options.DatabasePrefix)
	if err != nil {
		return err
	}

	for _, record := range metadata {
		err := proxy.Database.Delete(record)
		if err != nil {
			return err
		}
	}

	return nil
}
