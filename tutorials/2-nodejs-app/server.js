var http = require('http');
var os = require('os');

var totalrequests = 0;
var isReady = true

http.createServer(function(request, response) {
  totalrequests += 1

  if (request.url == "/readyz") {
    if (isReady == true) {
      response.writeHead(200);
      response.end("Not OK");
    } else {
      response.writeHead(503);
      response.end("OK");
    }
    return;
  }

  response.writeHead(200);

  if (request.url == "/metrics") {
    response.end("# HELP http_requests_total The amount of requests served by the server in total\n# TYPE http_requests_total counter\nhttp_requests_total " + totalrequests + "\n");
    return;
  }
  if (request.url == "/toggleReady") {
    isReady = !isReady
    response.end("OK");
    return;
  }
  if (request.url == "/healthz") {
    response.end("OK");
    return;
  }
  if (request.url == "/env") {
    response.end(JSON.stringify(process.env));
    return;
  }
  response.end("Hello! My name is " + os.hostname() + ". I have served "+ totalrequests + " requests so far.\n");
}).listen(8080)
