// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

func ExampleCopyHeadersOnRedirect() {
	request := httptest.NewRequest("GET", "/", nil)
	previous := httptest.NewRequest("GET", "/", nil)
	previous.Header.Set("Copy-Me", "copied value")

	client := http.Client{
		CheckRedirect: CopyHeadersOnRedirect("copy-me"), // canonicalization will happen
	}

	if err := client.CheckRedirect(request, []*http.Request{previous}); err != nil {
		// shouldn't be output
		fmt.Println(err)
	}

	fmt.Println(request.Header.Get("Copy-Me"))

	// Output:
	// copied value
}

func ExampleMaxRedirects() {
	request := httptest.NewRequest("GET", "/", nil)
	client := http.Client{
		CheckRedirect: MaxRedirects(5),
	}

	if client.CheckRedirect(request, make([]*http.Request, 4)) == nil {
		fmt.Println("fewer than 5 redirects")
	}

	if client.CheckRedirect(request, make([]*http.Request, 6)) != nil {
		fmt.Println("max redirects exceeded")
	}

	// Output:
	// fewer than 5 redirects
	// max redirects exceeded
}

func ExampleNewCheckRedirects() {
	request := httptest.NewRequest("GET", "/", nil)
	previous := httptest.NewRequest("GET", "/", nil)
	previous.Header.Set("Copy-Me", "copied value")

	client := http.Client{
		CheckRedirect: NewCheckRedirects(
			MaxRedirects(10),
			CopyHeadersOnRedirect("Copy-Me"),
			func(*http.Request, []*http.Request) error {
				fmt.Println("custom check redirect")
				return nil
			},
		),
	}

	if err := client.CheckRedirect(request, []*http.Request{previous}); err != nil {
		// shouldn't be output
		fmt.Println(err)
	}

	fmt.Println(request.Header.Get("Copy-Me"))

	// Output:
	// custom check redirect
	// copied value
}
