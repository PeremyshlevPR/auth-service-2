package acceptance

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestHealthEndpoint() {
	resp, err := http.Get(s.App.BaseURL + "/health")
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode, "Expected status 200")
}
