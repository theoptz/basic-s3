package tests

import (
	"bytes"
	"io"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

const (
	fileSize        = 5 * 1024 * 1024
	defaultEndpoint = "http://localhost:8080/bucket/test"
	contentType     = "application/x-binary"
)

func TestAPI(t *testing.T) {
	suite.Run(t, &APISuite{})
}

type APISuite struct {
	suite.Suite

	endpoint string
	data     []byte
}

func (s *APISuite) SetupSuite() {
	s.endpoint = os.Getenv("API_ENDPOINT")
	if s.endpoint == "" {
		s.endpoint = defaultEndpoint
	}

	s.generateData()
}

func (s *APISuite) TestAPI() {
	s.upload()
	s.download()
}

func (s *APISuite) upload() {
	startTime := time.Now()
	s.T().Logf("uploading file: %d bytes", len(s.data))

	req, err := http.NewRequest(http.MethodPut, s.endpoint, bytes.NewBuffer(s.data))
	s.Require().NoError(err)

	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)

	s.T().Logf("upload complete for %s", time.Since(startTime))

	s.Require().Equal(http.StatusOK, resp.StatusCode)
}

func (s *APISuite) download() {
	startTime := time.Now()
	s.T().Logf("downloading file: %d bytes", len(s.data))

	req, err := http.NewRequest(http.MethodGet, s.endpoint, http.NoBody)
	s.Require().NoError(err)

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer func() {
		s.Assert().NoError(resp.Body.Close())
	}()
	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	s.Assert().Equal(contentType, resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	s.T().Logf("download complete for %s", time.Since(startTime))

	s.Require().NoError(err)
	s.Assert().Equal(s.data, body)
}

func (s *APISuite) generateData() {
	s.data = make([]byte, fileSize)
	for i := 0; i < fileSize; i++ {
		s.data[i] = byte('a' + rand.Intn(26))
	}
}
