/* Adapted from: https://github.com/lovoo/nsq_exporter/blob/master/collector/stats.go
Copyright (c) 2015-2016, LOVOO GmbH
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

* Neither the name of LOVOO GmbH nor the names of its
  contributors may be used to endorse or promote products derived from
  this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package main

import (
	"encoding/json"
	"net/http"
)

type statsResponse struct {
	StatusCode int    `json:"status_code"`
	StatusText string `json:"status_text"`
	Data       stats  `json:"data"`
}

type stats struct {
	Version   string   `json:"version"`
	Health    string   `json:"health"`
	StartTime int64    `json:"start_time"`
	Topics    []*topic `json:"topics"`
}

// see https://github.com/nsqio/nsq/blob/master/nsqd/stats.go
type topic struct {
	Name         string     `json:"topic_name"`
	Paused       bool       `json:"paused"`
	Depth        int64      `json:"depth"`
	BackendDepth int64      `json:"backend_depth"`
	MessageCount uint64     `json:"message_count"`
	Channels     []*channel `json:"channels"`
}

type channel struct {
	Name          string    `json:"channel_name"`
	Paused        bool      `json:"paused"`
	Depth         int64     `json:"depth"`
	BackendDepth  int64     `json:"backend_depth"`
	MessageCount  uint64    `json:"message_count"`
	InFlightCount int       `json:"in_flight_count"`
	DeferredCount int       `json:"deferred_count"`
	RequeueCount  uint64    `json:"requeue_count"`
	TimeoutCount  uint64    `json:"timeout_count"`
	Clients       []*client `json:"clients"`
}

type client struct {
	ID            string `json:"client_id"`
	Hostname      string `json:"hostname"`
	Version       string `json:"version"`
	RemoteAddress string `json:"remote_address"`
	State         int32  `json:"state"`
	FinishCount   uint64 `json:"finish_count"`
	MessageCount  uint64 `json:"message_count"`
	ReadyCount    int64  `json:"ready_count"`
	InFlightCount int64  `json:"in_flight_count"`
	RequeueCount  uint64 `json:"requeue_count"`
	ConnectTime   int64  `json:"connect_ts"`
	SampleRate    int32  `json:"sample_rate"`
	Deflate       bool   `json:"deflate"`
	Snappy        bool   `json:"snappy"`
	TLS           bool   `json:"tls"`
}

// getNsqdStats calls nsqd's HTTP API and returns the response.
func getNsqdStats(nsqdURL string) (*stats, error) {
	resp, err := http.Get(nsqdURL + "/stats?format=json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var sr statsResponse
	if err = json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}
	return &sr.Data, nil
}
