package vulcand

import (
	"fmt"

	. "gopkg.in/check.v1"
)

type FrontendSuite struct{}

var _ = Suite(&FrontendSuite{})

func (s *FrontendSuite) TestSpecAndHash(c *C) {
	for i, tc := range []struct {
		fes  *frontendSpec
		spec string
		hash string
	}{{
		fes:  newFrontendSpec("ghost", "example.com", "/v2/<domain>/events", []string{"GET"}, nil),
		spec: `{"Type":"http","BackendId":"ghost","Route":"Host(\"example.com\") && Method(\"GET\") && Path(\"/v2/<domain>/events\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`,
		hash: "cbbb3953754a6a1276e343754cf8eb288ac959dd",
	}, {
		fes:  newFrontendSpec("ghost", "exAMPle.com", "/v2/<domain>/events", []string{"GET"}, nil),
		spec: `{"Type":"http","BackendId":"ghost","Route":"Host(\"example.com\") && Method(\"GET\") && Path(\"/v2/<domain>/events\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`,
		hash: "cbbb3953754a6a1276e343754cf8eb288ac959dd",
	}, {
		fes:  newFrontendSpec("ghost", "example.com", "/v2/<domain>/events", []string{"get"}, nil),
		spec: `{"Type":"http","BackendId":"ghost","Route":"Host(\"example.com\") && Method(\"GET\") && Path(\"/v2/<domain>/events\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`,
		hash: "cbbb3953754a6a1276e343754cf8eb288ac959dd",
	}, {
		fes:  newFrontendSpec("ghostt", "example.com", "/v2/<domain>/events", []string{"GET"}, nil),
		spec: `{"Type":"http","BackendId":"ghostt","Route":"Host(\"example.com\") && Method(\"GET\") && Path(\"/v2/<domain>/events\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`,
		hash: "6671883acc37c3ea60df613d2a109a6d4ad5751a",
	}, {
		fes:  newFrontendSpec("ghost", "examplee.com", "/v2/<domain>/events", []string{"GET"}, nil),
		spec: `{"Type":"http","BackendId":"ghost","Route":"Host(\"examplee.com\") && Method(\"GET\") && Path(\"/v2/<domain>/events\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`,
		hash: "b5c6c2f3f22a6214267f3dbea4582fa9baa79765",
	}, {
		fes:  newFrontendSpec("ghost", "example.com", "/v2/<domai>/events", []string{"GET"}, nil),
		spec: `{"Type":"http","BackendId":"ghost","Route":"Host(\"example.com\") && Method(\"GET\") && Path(\"/v2/<domai>/events\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`,
		hash: "431965613ccb3debce7565b82b138e7dc6361d30",
	}, {
		fes:  newFrontendSpec("ghost", "example.com", "/v2/<domai>/events", []string{"POST"}, nil),
		spec: `{"Type":"http","BackendId":"ghost","Route":"Host(\"example.com\") && Method(\"POST\") && Path(\"/v2/<domai>/events\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`,
		hash: "d1f569bfba5e8004a76c3001029ee2fadf4ca3ec",
	}, {
		fes: newFrontendSpec("ghost", "example.com", "/v2/<domain>/events", []string{"GET"},
			[]Middleware{{Type: "T1", ID: "Id1", Priority: 7}}),
		spec: `{"Type":"http","BackendId":"ghost","Route":"Host(\"example.com\") && Method(\"GET\") && Path(\"/v2/<domain>/events\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`,
		hash: "71baccfabc77c0bdaf63df4d9c8aa2408bf6f3f6",
	}} {
		fmt.Printf("Test case #%d\n", i)

		// When
		spec := tc.fes.spec()
		hash, err := tc.fes.hash()

		// Then
		c.Assert(err, IsNil)
		c.Assert(spec, Equals, tc.spec)
		c.Assert(hash, Equals, tc.hash)
	}
}
