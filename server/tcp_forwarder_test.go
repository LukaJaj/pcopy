package server

import (
	"bytes"
	"heckel.io/pcopy/clipboard/clipboardtest"
	"heckel.io/pcopy/config"
	"heckel.io/pcopy/config/configtest"
	"heckel.io/pcopy/test"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"regexp"
	"testing"
)

func TestTCPForwarder_Help(t *testing.T) {
	_, conf := configtest.NewTestConfig(t)
	server := newTestServer(t, conf)
	forwarder := newTCPForwarder(":12386", config.ExpandServerAddr(conf.ServerAddr), server.Handle)
	defer forwarder.shutdown()

	go forwarder.listenAndServe()

	var stdout bytes.Buffer
	cmd := exec.Command("sh", "-c", "echo help | nc -N localhost 12386")
	cmd.Stdout = &stdout
	cmd.Run()

	test.StrContains(t, stdout.String(), `This is is the netcat-endpoint for pcopy`)
}

func TestTCPForwarder_Basic(t *testing.T) {
	_, conf := configtest.NewTestConfig(t)
	server := newTestServer(t, conf)
	forwarder := newTCPForwarder(":12386", config.ExpandServerAddr(conf.ServerAddr), server.Handle)
	defer forwarder.shutdown()

	go forwarder.listenAndServe()

	var stdout bytes.Buffer
	cmd := exec.Command("sh", "-c", "echo hi there | nc -N localhost 12386")
	cmd.Stdout = &stdout
	cmd.Run()

	matches := regexp.MustCompile(`https://localhost:12345/(\S+)`).FindStringSubmatch(stdout.String())
	req, _ := http.NewRequest(http.MethodGet, matches[0], nil)

	test.StrContains(t, stdout.String(), `https://localhost:12345/`)
	clipboardtest.Content(t, conf, matches[1], "hi there\n")

	rr := httptest.NewRecorder()
	server.Handle(rr, req)
	test.StrEquals(t, rr.Body.String(), "hi there\n")
}

func TestTCPForwarder_WithOptions(t *testing.T) {
	_, conf := configtest.NewTestConfig(t)
	server := newTestServer(t, conf)
	forwarder := newTCPForwarder(":12386", config.ExpandServerAddr(conf.ServerAddr), server.Handle)
	defer forwarder.shutdown()

	go forwarder.listenAndServe()

	var stdout bytes.Buffer
	cmd := exec.Command("sh", "-c", "(echo \"pcopy:my-id?t=10m\"; echo hi there) | nc -N localhost 12386")
	cmd.Stdout = &stdout
	cmd.Run()

	test.StrContains(t, stdout.String(), "https://localhost:12345/my-id")
	test.StrContains(t, stdout.String(), "valid for 10m")
	clipboardtest.Content(t, conf, "my-id", "hi there\n")
}

func TestTCPForwarder_WithInvalidOptions(t *testing.T) {
	_, conf := configtest.NewTestConfig(t)
	server := newTestServer(t, conf)
	forwarder := newTCPForwarder(":12386", config.ExpandServerAddr(conf.ServerAddr), server.Handle)
	defer forwarder.shutdown()

	go forwarder.listenAndServe()

	var stdout bytes.Buffer
	cmd := exec.Command("sh", "-c", "(echo \"pcopy://my-id?t=10m\"; echo hi there) | nc -N localhost 12386")
	cmd.Stdout = &stdout
	cmd.Run()

	test.StrEquals(t, stdout.String(), "400 Bad Request\n")
	clipboardtest.NotExist(t, conf, "my-id")
}

func TestTCPForwarder_WithLimitFailure(t *testing.T) {
	_, conf := configtest.NewTestConfig(t)
	conf.FileSizeLimit = 5
	server := newTestServer(t, conf)
	forwarder := newTCPForwarder(":12386", config.ExpandServerAddr(conf.ServerAddr), server.Handle)
	defer forwarder.shutdown()

	go forwarder.listenAndServe()

	var stdout bytes.Buffer
	cmd := exec.Command("sh", "-c", "echo 123456 | nc -N localhost 12386")
	cmd.Stdout = &stdout
	cmd.Run()

	test.StrEquals(t, stdout.String(), "413 Request Entity Too Large\n")
}
