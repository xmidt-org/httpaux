// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package retry

import (
	"fmt"
	"net/http"
	"time"
)

func ExampleClient_defaults() {
	client := New(Config{}, nil) // uses http.DefaultClient
	fmt.Println(client.Retries())

	// Output:
	// 0
}

func ExampleClient_simple() {
	client := New(Config{
		Retries:  2,
		Interval: 3 * time.Minute,

		// custom Check, if desired
		Check: func(r *http.Response, err error) bool {
			return err == nil && r.StatusCode != 200
		},
	}, new(http.Client))
	fmt.Println(client.Retries())

	// Output:
	// 2
}

func ExampleClient_exponentialBackoff() {
	client := New(Config{
		Retries:    5,
		Interval:   3 * time.Minute,
		Multiplier: 1.5,
		// uses DefaultCheck
	}, new(http.Client))
	fmt.Println(client.Retries())

	// Output:
	// 5
}

func ExampleClient_exponentialBackoffWithJitter() {
	client := New(Config{
		Retries:    5,
		Interval:   3 * time.Minute,
		Multiplier: 3.0,

		// a random window that increases with each retry
		// the window is [1-Jitter,1+Jitter], and a random
		// value in that range is multiplied by Interval*Multipler^n
		// where n is the 0-based attempt.
		Jitter: 0.2,
	}, new(http.Client))
	fmt.Println(client.Retries())

	// Output:
	// 5
}

func ExampleClient_middleware() {
	retryClient := New(Config{
		Retries:  3,
		Interval: 1 * time.Minute,
	}, nil) // will use http.DefaultClient

	// this is a decorated client using the above retry semantics
	_ = retryClient.Then(new(http.Client))

	fmt.Println(retryClient.Retries())

	// Output:
	// 3
}
