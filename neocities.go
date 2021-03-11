// Package neocities is a wrapper around the NeoCities API to perform
// various functions related to managing a NeoCities site, as well as
// obtaining information about NeoCities sites.
package neocities

import (
	"bytes"
	"fmt"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"io"
	"io/fs"
	"log"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Verbose toggles the verbosity of the library when it is performing
// actions. Toggling it will automatically print any relevant information
// to a logger.
var Verbose bool
var logger *log.Logger

func v(i interface{}) {
	if Verbose {
		logger.Println(i)
	}
}

// SetLogger allows you to set a logger. If this isn't used,
// the default logger provided by log will be used instead.
func SetLogger(l *log.Logger) {
	logger = l
}

func init() {
	logger = log.Default()
}

var (
	NoAPI = errors.New("no API supplied to APIClient")
	NoKey = errors.New("no key supplied, required for this operation")
	SiteError = errors.New("an error occurred during API call")
	MissingReq = errors.New("a required variable is missing")
)

type NeoTime time.Time

func (t *NeoTime) UnmarshalJSON(b []byte) error {
	c, err := time.Parse(time.RFC1123Z, string(b[1:len(b)-1]))
	if err != nil {
		return err
	}

	*t = NeoTime(c)
	return nil
}

// API represents a NeoCities API endpoint
type API string

const (
	Upload API = "upload"
	Delete API = "delete"
	List API = "list"
	Info API = "info"
)

// APIClient represents an API client.
type APIClient struct {
	API    API
	client *http.Client
	method string
	url    *url.URL
	vars   url.Values
	head   http.Header
}

// APIError represents an error message returned by the NeoCities API.
type APIError struct {
	Result    string `json:"result"`
	ErrorType string `json:"error_type"`
	Message   string `json:"message"`
}

// NewAPIError takes a reader containing JSON related to an error,
// and returns a struct containing the error type and message.
func NewAPIError(r io.Reader) APIError {
	var e APIError
	b, _ := io.ReadAll(r)

	json.Unmarshal(b, &e)

	return e
}

// Site represents a NeoCities site - both in authentication,
// and in information grabbing. Info is only used to be filled
// in by the info API endpoint.
//
// Some API calls require the Key field to be filled - others
// do not. If an API call requires an authentication key, and
// there is no key in the Key field, the NoKey error will be
// passed from NewAPIClient to the calling function.
type Site struct {
	SiteName string
	Key      string
	Info struct {
		Hits    int      `json:"hits"`
		Updated NeoTime  `json:"last_updated"`
		Domain  string   `json:"domain"`
		Tags    []string `json:"tags"`
	} `json:"info"`
}

// NewAPIClient returns a struct containing everything you need
// to perform RESTful API actions with NeoCities.
//
// If a key is not supplied, a NoKey error will be returned,
// but the API client will still be ready to be used. This is
// meant for API calls that require authentication.
func (s *Site) NewAPIClient(api API) (*APIClient, error) {
	a := new(APIClient)
	a.API = api
	a.client = new(http.Client)
	a.vars = make(url.Values)
	a.head = make(http.Header)
	a.url, _ = url.Parse("https://neocities.org")

	a.url.Path = path.Join("api", string(api))

	if s.Key != "" {
		a.head.Add("Authorization", "Bearer " + s.Key)
		return a, nil
	}

	return a, NoKey
}

// ChangeAPI changes the API being used by an APIClient.
func (a *APIClient) ChangeAPI(api API) *APIClient {
	a.API = api
	a.url.Path = path.Join("api", string(api))

	return a
}

// NewAPIRequest takes the potential body of a request.
func (a *APIClient) NewAPIRequest(r io.Reader) (*http.Request, error) {
	var m string
	switch a.API {
	case Upload, Delete:
		m = "POST"
	case Info, List:
		m = "GET"
	default:
		return nil, NoAPI
	}

	req, err := http.NewRequest(m, a.url.String(), r)
	if err != nil {
		return nil, err
	}

	req.Header = a.head

	return req, nil
}

// ReadFile reads an entire file.
//
// Its intended use is when creating a Site struct,
// and a key is stored in a file.
func ReadFile(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}

	s, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(s), nil
}

func MakeMIMEMultipartFile(f io.Reader, name string) (*bytes.Buffer, string, error) {
	b := new(bytes.Buffer)
	w := multipart.NewWriter(b)
	fb, err := io.ReadAll(f)
	if err != nil {
		return nil, "", err
	}

	w.SetBoundary("NEOCITIES-GO-CLIENT")

	/*
	h := make(textproto.MIMEHeader)
	h.Add(
		"Content-Disposition",
		fmt.Sprintf(
			"form-data; name=\"%s\"; filename=\"%s\"", name, name,
		),
	)
	h.Add("Content-Type", strings.Split(http.DetectContentType(fb), ";")[0])

	d, err := w.CreatePart(h)
	if err != nil {
		return nil, "", err
	}
	*/
	d, err := w.CreateFormFile(name, name)
	wr := bytes.NewReader(fb)
	wr.WriteTo(d)

	w.Close()

	return b, w.FormDataContentType(), nil
}

// UploadFile uploads a file using the information provided by Site.
// If c is nil, a new client is created - otherwise, an existing API
// client is used. If name is nil, the base of the path provided by
// file (file can be a singular string) is used.
//
// File cannot be an empty string.
func (s *Site) UploadFile(file string, name string, c *APIClient) error {
	var err error

	if c == nil {
		c, err = s.NewAPIClient(Upload)
		if errors.Is(err, NoKey) {
			return err
		}
	}

	if file == "" {
		return errors.New("no file provided")
	}

	f, err := os.Open(file)
	if err != nil {
		return err
	}

	if name == "" {
		name = filepath.Base(file)
	}

	u, h, err := MakeMIMEMultipartFile(f, name)
	if err != nil {
		return err
	}

	c.head.Add("Content-Type", h)
	req, err := c.NewAPIRequest(u)
	if err != nil {
		return err
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	d := make(url.Values)
	d.Add(name, string(b))

	v(fmt.Sprintf("NeoCities: uploading %s as %s.", file, name))
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		e := NewAPIError(resp.Body)
		return fmt.Errorf("%w, API: %s, Code: %s, Response: %s", SiteError, string(Upload), resp.Status, e.Message)
	}

	return nil
}

// Push pushes an entire directory and its subdirectories to
// the website in the given directory. If c is nil, a new
// client is created.
//
// Be CAREFUL when using this - Push will automatically
// upload all directories recursively.
//
// Attempts to upload all files to the NeoCities site, and
// only halts if the site key was not passed.
func (s *Site) Push(dir string, c *APIClient) error {
	var err error

	if c == nil {
		c, err = s.NewAPIClient(Upload)
		if errors.Is(err, NoKey) {
			return err
		}
	}

	walk := func(p string, d fs.DirEntry, e error) error {
		if !d.IsDir() {
			err = s.UploadFile(p, p, c)
			if err != nil {
				logger.Println(err)
			}
		}

		return nil
	}

	err = fs.WalkDir(os.DirFS("./"), dir, walk)
	if err != nil {
		return err
	}

	return nil
}

// DeleteFiles deletes a set of files from a NeoCities website.
func (s *Site) DeleteFiles(c *APIClient, files ...string) error {
	var err error

	if c == nil {
		c, err = s.NewAPIClient(Delete)
		if errors.Is(err, NoKey) {
			return err
		}
	}

	d := make(url.Values)

	for _, f := range files {
		d.Add("filenames[]", f)
	}

	req, err := c.NewAPIRequest(strings.NewReader(d.Encode()))
	if err != nil {
		return err
	}

	v(fmt.Sprintf("NeoCities: deleting %s.", files))
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		e := NewAPIError(resp.Body)
		return fmt.Errorf("%w, API: %s, Code: %s, Response: %s", SiteError, string(Upload), resp.Status, e.Message)
	}

	return nil
}

// GetInfo encodes the site information into the recieving
// Site struct. If c is nil, an API client will be made with
// the given info.
func (s *Site) GetInfo(c *APIClient) (*Site, error) {
	var err error

	if s.SiteName == "" {
		return nil, fmt.Errorf("%w: SiteName", MissingReq)
	}

	if c == nil {
		c, err = s.NewAPIClient(Info)
	}

	c.vars.Add("sitename", s.SiteName)
	c.url.RawQuery = c.vars.Encode()

	req, err := http.NewRequest("GET", c.url.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	resb, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(resb, &s)

	return s, nil
}

// SiteFile represents a file in a NeoCities website.
// This is typically the output you get from a list API call.
type SiteFile struct {
	Path string      `json:"path"`
	IsDir bool       `json:"is_directory"`
	Size int         `json:"size"`
	Updated NeoTime  `json:"updated_at"`
	SHA1 string      `json:"sha1_hash"`
}

// List grabs the files from path into an array of SiteFiles.
// Requires an API key. If c is nil, a new client is created.
func (s *Site) List(path string, c *APIClient) ([]SiteFile, error) {
	var err error
	l := struct {
		Files []SiteFile `json:"files"`
	}{
		[]SiteFile{},
	}

	if c == nil {
		c, err = s.NewAPIClient(List)
		if errors.Is(err, NoKey) {
			return l.Files, err
		}
	}

	req, err := c.NewAPIRequest(nil)
	if err != nil {
		return l.Files, err
	}

	c.vars.Add("path", path)
	c.url.RawQuery = c.vars.Encode()

	resp, err := c.client.Do(req)

	if err != nil {
		return l.Files, err
	}

	resb, err := io.ReadAll(resp.Body)
	if err != nil {
		return l.Files, err
	}

	json.Unmarshal(resb, &l)

	return l.Files, nil
}
