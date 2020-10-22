package l7

import (
	"log"
	"netsim/api"
	"netsim/devices"
	"netsim/hardware"
	"netsim/protocol"
	"testing"
	"time"
)

func TestRequestParsing(t *testing.T) {
	req := `GET /bin/login?user=Peter+Lee&pw=123456&action=login HTTP/1.1
Accept: image/gif, image/jpeg
Referer: http://127.0.0.1:8000/login.html
Accept-Language: en-us
Accept-Encoding: gzip, deflate
User-Agent: Mozilla/4.0 (compatible; MSIE 6.0; Windows NT 5.1)
Host: 127.0.0.1:8000
Connection: Keep-Alive
`
	httpRequest := NewHttpRequest()
	done := httpRequest.Add([]byte(req))
	if !done {
		t.Fail()
	}
	if httpRequest.Method() != "GET" {
		t.Fail()
	}

	req2 := string(httpRequest.ToBytes())
	print(req2)
}

func TestResponseParsing(t *testing.T) {
	res := `HTTP/1.1 501
Date: Sun, 18 Oct 2009 10:32:05 GMT
Server: Apache/2.2.14 (Win32)
Allow: GET,HEAD,POST,OPTIONS,TRACE
Content-Length: 207
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
`
	httpResponse := NewHttpResponse()
	done := httpResponse.Add([]byte(res))
	if !done {
		t.Fail()
	}
	if httpResponse.Status() != 501 {
		t.Fail()
	}

	res2 := string(httpResponse.ToBytes())
	print(res2)
}

/*
Testcase
*/
func TestSimpleReliableDataTransfer(t *testing.T) {
	node1 := devices.NewComputer([]byte("immac1"), []byte{10, 0, 0, 1})
	node1.AddAddress([]byte{10, 0, 0, 2}, []byte("immac2"))
	node1.AddRoute(protocol.DefaultRouteCidr, []byte{10, 0, 0, 2})

	node2 := devices.NewComputer([]byte("immac2"), []byte{10, 0, 0, 2})
	node2.AddAddress([]byte{10, 0, 0, 1}, []byte("immac1"))
	node2.AddRoute(protocol.DefaultRouteCidr, []byte{10, 0, 0, 1})

	_ = hardware.NewDuplexLink(100, 1e8, 0.00, node1.GetAdapter(), node2.GetAdapter())

	go hardware.Clk.Start()
	node1.TurnOn()
	node2.TurnOn()

	// Send the packet and wait
	node1.Run(server)
	node2.Run(client)

	time.Sleep(10 * time.Second)
}

func server(server *devices.Computer) {
	socket := server.NewSocket(api.AF_INET, api.SOCK_STREAM, 0)

	//Bind to port 80
	socket.Bind([]byte{0, 0, 0, 0}, 80)

	//List on the port
	socket.Listen(10)

	//Accept a connection
	sock := socket.Accept()

	//Create a HttpRequest object
	request := NewHttpRequest()

	//Receive data
	for {
		data := sock.Recv(100)
		if len(data) > 0 {
			done := request.Add(data)
			if done {
				break
			}
		}
	}

	//Process the HTTP request
	if request.Method() == "GET" && request.Path() == "/health" {
		log.Printf("Got request for /health")
		response := NewHttpResponseWithParams(200, map[string]string{"Content-Type": "text"}, []byte("Service is UP"))
		sock.Send(response.ToBytes())
	} else {
		log.Printf("Got %s request for %s", request.Method(), request.Path())
		response := NewHttpResponseWithParams(404, map[string]string{"Content-Type": "text"}, []byte("No such page"))
		sock.Send(response.ToBytes())
	}
}

func client(client *devices.Computer) {
	socket := client.NewSocket(api.AF_INET, api.SOCK_STREAM, 0)

	//Connect to the server
	socket.Connect([]byte{10, 0, 0, 1}, 80)

	//Create a HTTP request
	request := NewHttpRequestWithParams("GET", "/health", map[string]string{}, nil)

	//Send some data
	socket.Send(request.ToBytes())

	//Create a HttpResponse object
	response := NewHttpResponse()

	//Read the data
	for {
		data := socket.Recv(100)
		if len(data) > 0 {
			done := response.Add(data)
			if done {
				break
			}
		}
	}

	//Print the response
	log.Printf("Response: Status=%d, Body=%s", response.Status(), response.Body())

	//Close the connection
	time.Sleep(10 * time.Second)
	socket.Close()
}
