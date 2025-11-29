package acceptance

import (
	"net/http"
)

func (s *Suite) TestHealthEndpoint() {
	resp, err := http.Get(s.BaseURL + "/health")
	s.Require().NoError(err, "Failed to make request")
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode, "Expected status 200")
}
