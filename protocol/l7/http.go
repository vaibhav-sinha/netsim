package l7

import (
	"fmt"
	"strconv"
	"strings"
)

/*
GET /bin/login?user=Peter+Lee&pw=123456&action=login HTTP/1.1
Accept: image/gif, image/jpeg
Referer: http://127.0.0.1:8000/login.html
Accept-Language: en-us
Accept-Encoding: gzip, deflate
User-Agent: Mozilla/4.0 (compatible; MSIE 6.0; Windows NT 5.1)
Host: 127.0.0.1:8000
Connection: Keep-Alive

--------------------------------------------------------

HTTP/1.1 501
Date: Sun, 18 Oct 2009 10:32:05 GMT
Server: Apache/2.2.14 (Win32)
Allow: GET,HEAD,POST,OPTIONS,TRACE
Content-Length: 215
Connection: close
Content-Type: text/html; charset=iso-8859-1

<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>501 Method Not Implemented</title>
</head><body>
<h1>Method Not Implemented</h1>
<p>get to /index.html not supported.<br />
</p>
</body></html>
*/
type HttpRequest struct {
	buffer  []byte
	method  string
	path    string
	headers map[string]string
	body    []byte
}

func NewHttpRequest() *HttpRequest {
	return &HttpRequest{
		headers: map[string]string{},
	}
}

func NewHttpRequestWithParams(method string, path string, headers map[string]string, body []byte) *HttpRequest {
	return &HttpRequest{
		method:  method,
		path:    path,
		headers: headers,
		body:    body,
	}
}

func (h *HttpRequest) Add(d []byte) bool {
	h.buffer = append(h.buffer, d...)
	return h.parse()
}

func (h *HttpRequest) ToBytes() []byte {
	str := fmt.Sprintf("%s %s HTTP/1.1\n", h.method, h.path)
	if len(h.body) > 0 {
		h.headers["Content-Length"] = strconv.Itoa(len(h.body))
	}
	for k, v := range h.headers {
		str += fmt.Sprintf("%s: %s\n", k, v)
	}
	if len(h.body) > 0 {
		str += string(h.body)
		str += "\n"
	}

	return []byte(str)
}

func (h *HttpRequest) Method() string {
	return h.method
}

func (h *HttpRequest) Path() string {
	return h.path
}

func (h *HttpRequest) Headers() map[string]string {
	return h.headers
}

func (h *HttpRequest) Body() []byte {
	return h.body
}

func (h *HttpRequest) parse() bool {
	req := string(h.buffer)
	lines := strings.Split(req, "\n")
	if len(lines) == 1 {
		//The first line is not arrived yet, nothing to do
		return false
	}

	//The first line has come
	lineParts := strings.Split(lines[0], " ")

	//Parse a GET request
	if lineParts[0] == "GET" {
		//If the last line is empty, then the request is complete
		if lines[len(lines)-1] == "" {
			h.method = "GET"
			h.path = lineParts[1]

			//Parse the headers
			for _, line := range lines[1 : len(lines)-1] {
				lineParts = strings.Split(line, ": ")
				h.headers[lineParts[0]] = lineParts[1]
			}
			return true
		} else {
			return false
		}
	}

	//Parse a POST request
	if lineParts[0] == "POST" {
		h.method = "POST"
		h.path = lineParts[1]

		//If an empty line has not come then body could not have started
		found := false
		emptyLineIdx := -1
		for i, line := range lines[1:] {
			if line == "" {
				found = true
				emptyLineIdx = i + 1
				break
			}
		}
		if !found {
			return false
		}

		//Parse the headers
		headers := map[string]string{}
		for _, line := range lines[1:emptyLineIdx] {
			lineParts = strings.Split(line, ": ")
			headers[lineParts[0]] = lineParts[1]
		}

		//Check the Content-Length header
		contentLength, _ := strconv.Atoi(headers["Content-Length"])

		//If the length of body matches the value in header, then request is complete
		actualLength := 0
		for _, line := range lines[emptyLineIdx+1:] {
			actualLength += len(line)
		}

		if contentLength == actualLength {
			h.headers = headers
			for _, line := range lines[emptyLineIdx+1:] {
				h.body = append(h.body, []byte(line)...)
			}
			return true
		} else {
			return false
		}
	}

	return false
}

type HttpResponse struct {
	buffer  []byte
	status  int
	headers map[string]string
	body    []byte
}

func NewHttpResponse() *HttpResponse {
	return &HttpResponse{
		headers: map[string]string{},
	}
}

func NewHttpResponseWithParams(status int, headers map[string]string, body []byte) *HttpResponse {
	return &HttpResponse{
		status:  status,
		headers: headers,
		body:    body,
	}
}

func (h *HttpResponse) Add(d []byte) bool {
	h.buffer = append(h.buffer, d...)
	return h.parse()
}

func (h *HttpResponse) ToBytes() []byte {
	str := fmt.Sprintf("HTTP/1.1 %d\n", h.status)
	if len(h.body) > 0 {
		h.headers["Content-Length"] = strconv.Itoa(len(h.body))
	}
	for k, v := range h.headers {
		str += fmt.Sprintf("%s: %s\n", k, v)
	}
	if len(h.body) > 0 {
		str += "\n"
		str += string(h.body)
		str += "\n"
	}

	return []byte(str)
}

func (h *HttpResponse) Status() int {
	return h.status
}

func (h *HttpResponse) Headers() map[string]string {
	return h.headers
}

func (h *HttpResponse) Body() []byte {
	return h.body
}

func (h *HttpResponse) parse() bool {
	req := string(h.buffer)
	lines := strings.Split(req, "\n")

	//If an empty line has not come then body could not have started
	found := false
	emptyLineIdx := -1
	for i, line := range lines[1:] {
		if line == "" {
			found = true
			emptyLineIdx = i + 1
			break
		}
	}
	if !found {
		return false
	}

	lineParts := strings.Split(lines[0], " ")
	h.status, _ = strconv.Atoi(lineParts[1])

	//Parse the headers
	headers := map[string]string{}
	for _, line := range lines[1:emptyLineIdx] {
		lineParts = strings.Split(line, ": ")
		headers[lineParts[0]] = lineParts[1]
	}

	//Check the Content-Length header
	contentLength, _ := strconv.Atoi(headers["Content-Length"])

	//If the length of body matches the value in header, then request is complete
	actualLength := 0
	for _, line := range lines[emptyLineIdx+1:] {
		actualLength += len(line)
	}

	if contentLength == actualLength {
		h.headers = headers
		for _, line := range lines[emptyLineIdx+1:] {
			h.body = append(h.body, []byte(line)...)
		}
		return true
	} else {
		return false
	}
}
