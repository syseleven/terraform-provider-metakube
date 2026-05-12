package metakube

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type loggingTransport struct {
	next http.RoundTripper
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	orig := req.Header
	req.Header = redactHeaders(orig.Clone())

	if dump, err := httputil.DumpRequestOut(req, true); err == nil {
		tflog.Trace(req.Context(), "MetaKube API Request", map[string]interface{}{"request": string(dump)})
	}

	req.Header = orig

	resp, err := t.nextTransport().RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if dump, err := httputil.DumpResponse(resp, true); err == nil {
		tflog.Trace(req.Context(), "MetaKube API Response", map[string]interface{}{"response": string(dump)})
	}

	return resp, nil
}

func (t *loggingTransport) nextTransport() http.RoundTripper {
	if t.next != nil {
		return t.next
	}
	return http.DefaultTransport
}

func redactHeaders(h http.Header) http.Header {
	for k := range h {
		if strings.EqualFold(k, "Authorization") {
			h.Set(k, "[REDACTED]")
		}
	}
	return h
}
