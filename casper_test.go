package casper

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"golang.org/x/net/http2"
)

func TestGenerateCookie(t *testing.T) {
	cases := []struct {
		assets      []string
		P           int
		cookieValue string
	}{
		{
			[]string{
				"/static/example.js",
			},
			1 << 6,
			"JA",
		},

		{
			[]string{
				"/js/jquery-1.9.1.min.js",
				"/assets/style.css",
			},
			1 << 6,
			"gU4",
		},

		{
			[]string{
				"/js/jquery-1.9.1.min.js",
				"/assets/style.css",
				"/static/logo.jpg",
				"/static/cover.jpg",
			},
			1 << 6,
			"gU54MA",
		},

		{
			[]string{
				"/js/jquery-1.9.1.min.js",
				"/assets/style.css",
				"/static/logo.jpg",
				"/static/cover.jpg",
			},
			1 << 10,
			"MMOJEkWo",
		},

		// See how long cookie is when push many files.
		// Minimum number of bits is N*log(P) = 20 * log(1<<6) = 120 bits = 15bytes
		{
			[]string{
				"/static/example1.jpg",
				"/static/example2.jpg",
				"/static/example3.jpg",
				"/static/example4.jpg",
				"/static/example5.jpg",
				"/static/example6.jpg",
				"/static/example7.jpg",
				"/static/example8.jpg",
				"/static/example9.jpg",
				"/static/example10.jpg",
				"/static/example11.jpg",
				"/static/example12.jpg",
				"/static/example13.jpg",
				"/static/example14.jpg",
				"/static/example15.jpg",
				"/static/example16.jpg",
				"/static/example17.jpg",
				"/static/example18.jpg",
				"/static/example19.jpg",
				"/static/example20.jpg",
			},
			1 << 6,
			"FmDhUxQHeuwQYINoQrxmr1g_iw", // 26bytes
		},
	}

	for _, tc := range cases {
		casper := New(tc.P, len(tc.assets))

		hashValues := make([]uint, 0, len(tc.assets))
		for _, content := range tc.assets {
			hashValues = append(hashValues, casper.hash([]byte(content)))
		}

		cookie, err := casper.generateCookie(hashValues)
		if err != nil {
			t.Fatalf("generateCookie should not fail")
		}

		if got, want := cookie.Value, tc.cookieValue; got != want {
			t.Fatalf("generateCookie=%q, want=%q", got, want)
		}
	}
}

func TestPush(t *testing.T) {
	cases := []struct {
		p        int
		push     []string
		sameTime bool

		clientCookie *http.Cookie
		serverCookie *http.Cookie

		casperCookie *http.Cookie
		pushed       []string
	}{
		{
			1 << 6,
			[]string{"/static/example.jpg"},
			false,
			nil,
			nil,

			&http.Cookie{
				Name:  defaultCookieName,
				Value: "KA",
				Path:  defaultCookiePath,
			},
			[]string{"/static/example.jpg"},
		},

		{
			1 << 6,
			[]string{"/static/example.jpg"},
			true, // push one by one
			nil,
			nil,

			&http.Cookie{
				Name:  defaultCookieName,
				Value: "KA",
				Path:  defaultCookiePath,
			},
			[]string{"/static/example.jpg"},
		},

		{
			1 << 6,
			[]string{
				"/js/jquery-1.9.1.min.js",
				"/assets/style.css",
				"/static/logo.jpg",
				"/static/cover.jpg",
			},
			false,
			nil,
			nil,

			&http.Cookie{
				Name:  defaultCookieName,
				Value: "gU54MA",
				Path:  defaultCookiePath,
			},
			[]string{
				"/js/jquery-1.9.1.min.js",
				"/assets/style.css",
				"/static/logo.jpg",
				"/static/cover.jpg",
			},
		},

		// With additional server side cookies
		{
			1 << 6,
			[]string{"/static/example.jpg"},
			false,
			nil,
			&http.Cookie{
				Name:  "session",
				Value: "BAh7CiIKZmxhc2hJ",
				Path:  "/",
			},

			&http.Cookie{
				Name:  defaultCookieName,
				Value: "KA",
				Path:  defaultCookiePath,
			},
			[]string{"/static/example.jpg"},
		},

		// With client side cookies
		{
			1 << 6,
			[]string{
				"/js/jquery-1.9.1.min.js",
				"/assets/style.css",
				"/static/logo.jpg",
				"/static/cover.jpg",
			},
			false,

			// This cookie is generated by /js/jquery-1.9.1.min.js and /assets/style.css
			// This means these are already pushed on previous request and should not
			// be pushed this time.
			&http.Cookie{
				Name:  defaultCookieName,
				Value: "gU4",
			},
			nil,

			&http.Cookie{
				Name:  defaultCookieName,
				Value: "gU54MA",
				Path:  defaultCookiePath,
			},
			[]string{
				"/static/logo.jpg",
				"/static/cover.jpg",
			},
		},

		{
			1 << 6,
			[]string{
				"/js/jquery-1.9.1.min.js",
				"/assets/style.css",
				"/static/logo.jpg",
				"/static/cover.jpg",
			},
			true, // push one by one

			// This cookie is generated by /js/jquery-1.9.1.min.js and /assets/style.css
			// This means these are already pushed on previous request and should not
			// be pushed this time.
			&http.Cookie{
				Name:  defaultCookieName,
				Value: "gU4",
			},
			nil,

			&http.Cookie{
				Name:  defaultCookieName,
				Value: "gU54MA",
				Path:  defaultCookiePath,
			},
			[]string{
				"/static/logo.jpg",
				"/static/cover.jpg",
			},
		},

		// With server and client cookies
		{
			1 << 6,
			[]string{
				"/js/jquery-1.9.1.min.js",
				"/assets/style.css",
				"/static/logo.jpg",
				"/static/cover.jpg",
			},
			false,

			// This cookie is generated by /js/jquery-1.9.1.min.js and /assets/style.css
			// This means these are already pushed on previous request and should not
			// be pushed this time.
			&http.Cookie{
				Name:  defaultCookieName,
				Value: "gU4",
			},

			&http.Cookie{
				Name:  "session",
				Value: "BAh7CiIKZmxhc2hJ",
				Path:  "/",
			},

			&http.Cookie{
				Name:  defaultCookieName,
				Value: "gU54MA",
				Path:  defaultCookiePath,
			},
			[]string{
				"/static/logo.jpg",
				"/static/cover.jpg",
			},
		},
	}

	for _, tc := range cases {
		pusher := New(tc.p, len(tc.push))
		pusher.skipPush = true

		testServer := newTestServer(t, pusher, tc.push, tc.sameTime, tc.serverCookie)
		defer testServer.Close()

		req, err := http.NewRequest("GET", testServer.URL, nil)
		if err != nil {
			t.Fatal(err)
		}

		if tc.clientCookie != nil {
			req.AddCookie(tc.clientCookie)
		}

		client := newTestH2Client(t)
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()

		// Inspect pushed contents
		// TODO(tcnksm): Separate test push one by one and push same time.
		n := len(tc.pushed)
		if !tc.sameTime {
			n = 1
		}
		if got, want := len(pusher.Pushed()), n; got != want {
			t.Fatalf("number of pushed contents %d, want %d", got, want)
		}

		if tc.sameTime {
			if got, want := pusher.Pushed(), tc.pushed; !reflect.DeepEqual(got, want) {
				t.Fatalf("number of pushed contents %v, want %v", got, want)
			}
		}

		// Inspect cookies to be returned from server.
		wantCookie := 1
		if tc.serverCookie != nil {
			wantCookie = 2
		}

		cookies := res.Cookies()
		if got, want := len(cookies), wantCookie; got != want {
			t.Fatalf("Number of cookie %d, want %d", got, want)
		}

		tc.casperCookie.Raw = tc.casperCookie.String() // Need to set Raw to compare
		if got, want := cookies[wantCookie-1], tc.casperCookie; !reflect.DeepEqual(got, want) {
			t.Fatalf("Get cookie name %#v, want %#v", got, want)
		}
	}
}

func TestPush_ServerPushNotSupported(t *testing.T) {
	var err error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cspr := New(1<<6, 10)
		_, err = cspr.Push(w, r, []string{"/static/example.jpg"}, nil)
	}))
	defer ts.Close()

	http.Get(ts.URL)

	if err == nil {
		t.Fatal("expect to be failed") // TODO(tcnksm): define error
	}
}

func newTestServer(t *testing.T, casper *Casper, contents []string, sameTime bool, cookie *http.Cookie) *httptest.Server {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Set additinal cookie if provided.
		if cookie != nil {
			http.SetCookie(w, cookie)
		}

		if sameTime {
			// Push all contents at same time
			if _, err := casper.Push(w, r, contents, nil); err != nil {
				t.Fatalf("Push failed: %s", err)
			}
		} else {
			// Push contents one by one. Test for context.
			for _, content := range contents {
				var err error
				r, err = casper.Push(w, r, []string{content}, nil)
				if err != nil {
					t.Fatalf("Push failed: %s", err)
				}
			}
		}

		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte(""))
		w.WriteHeader(http.StatusOK)
	}))

	if err := http2.ConfigureServer(ts.Config, nil); err != nil {
		t.Fatalf("Failed to configure h2 server: %s", err)
	}
	ts.TLS = ts.Config.TLSConfig
	ts.StartTLS()

	return ts
}

func newTestH2Client(t *testing.T) *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	if err := http2.ConfigureTransport(tr); err != nil {
		t.Fatalf("Failed to configure h2 transport: %s", err)
	}

	return &http.Client{
		Transport: tr,
	}
}
