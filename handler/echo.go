package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
)

// GetEcho is a simple handler that returns the request headers and body as the response.  This is useful for testing and debugging.
func GetEcho(ctx echo.Context) error {

	r := ctx.Request()
	var b bytes.Buffer

	fmt.Fprintf(&b, "HTTP REQUEST ECHO\n\n")

	fmt.Fprintf(&b, "  %-12s %s\n", "Method:", r.Method)
	fmt.Fprintf(&b, "  %-12s %s\n", "URL:", r.URL.String())
	fmt.Fprintf(&b, "  %-12s %s\n", "Proto:", r.Proto)
	fmt.Fprintf(&b, "  %-12s %s\n", "Remote:", ctx.RealIP())
	fmt.Fprintf(&b, "  %-12s %s\n", "Host:", r.Host)

	// ── Query Parameters ─────────────────────────────────────────
	queryParams := ctx.QueryParams()
	if len(queryParams) > 0 {
		fmt.Fprintf(&b, "\nQuery Parameters\n")
		keys := make([]string, 0, len(queryParams))
		for k := range queryParams {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			for _, v := range queryParams[k] {
				fmt.Fprintf(&b, "  %-20s %s\n", k+":", v)
			}
		}
	}

	// ── Headers ───────────────────────────────────────────────────
	fmt.Fprintf(&b, "\nHeaders\n")
	headerKeys := make([]string, 0, len(r.Header))
	for k := range r.Header {
		headerKeys = append(headerKeys, k)
	}
	sort.Strings(headerKeys)
	for _, k := range headerKeys {
		for _, v := range r.Header[k] {
			fmt.Fprintf(&b, "  %-30s %s\n", k+":", v)
		}
	}

	// ── Cookies ───────────────────────────────────────────────────
	if cookies := r.Cookies(); len(cookies) > 0 {
		fmt.Fprintf(&b, "\nCookies\n")
		for _, cookie := range cookies {
			fmt.Fprintf(&b, "  %-20s %s\n", cookie.Name+":", cookie.Value)
		}
	}

	// ── Path Parameters ───────────────────────────────────────────
	if paramNames := ctx.ParamNames(); len(paramNames) > 0 {
		fmt.Fprintf(&b, "\nPath Parameters\n")
		for _, k := range paramNames {
			fmt.Fprintf(&b, "  %-20s %s\n", k+":", ctx.Param(k))
		}
	}

	// ── Body ──────────────────────────────────────────────────────
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "failed to read body")
	}
	defer r.Body.Close()

	fmt.Fprintf(&b, "\nBody (%d bytes)\n", len(body))
	if len(body) == 0 {
		fmt.Fprintf(&b, "  (empty)\n")
	} else {
		ct := r.Header.Get("Content-Type")
		bodyStr := string(body)
		switch {
		case strings.Contains(ct, "application/x-www-form-urlencoded"):
			// Restore body so ParseForm can read it
			r.Body = io.NopCloser(bytes.NewBuffer(body))
			if err := r.ParseForm(); err == nil {
				formKeys := make([]string, 0, len(r.Form))
				for k := range r.Form {
					formKeys = append(formKeys, k)
				}
				sort.Strings(formKeys)
				for _, k := range formKeys {
					for _, v := range r.Form[k] {
						fmt.Fprintf(&b, "  %-20s %s\n", k+":", v)
					}
				}
			} else {
				fmt.Fprintf(&b, "  %s\n", bodyStr)
			}
		default:
			for _, line := range strings.Split(bodyStr, "\n") {
				fmt.Fprintf(&b, "  %s\n", line)
			}
		}
	}

	return ctx.String(http.StatusOK, b.String())
}
