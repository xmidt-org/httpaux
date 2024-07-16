// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package observe

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

func ExampleThen() {
	decorator := Then(
		http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			o := response.(Writer)

			// register a callback to be notified when a status code is written
			o.OnWriteHeader(func(code int) {
				fmt.Println("onWriteHeader callback =>", "code:", code)
			})

			// this will normally be done in another nested handler
			o.WriteHeader(201)

			// now, decorator code can access observed information
			fmt.Println("observed status code:", o.StatusCode())
		}),
	)

	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/example", nil)
	decorator.ServeHTTP(response, request)

	// Output:
	// onWriteHeader callback => code: 201
	// observed status code: 201
}
