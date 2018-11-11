package main

import (
	"flag"
	"github.com/Kledal/go-js-websocket-protobuf/messages/protos"
	"github.com/golang/protobuf/proto"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var MessageType int32

const (
	MessageTypeChatRequest  = 1
	MessageTypeChatResponse = 2
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options

func echo(w http.ResponseWriter, r *http.Request) {

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}

		rootMessage := &root.Root{}
		err = proto.Unmarshal(message, rootMessage)

		if err != nil {
			log.Println("proto read error:", err)
		}

		switch rootMessage.GetType() {
		case MessageTypeChatRequest:
			chatRequest := &root.ChatRequest{}
			err = proto.Unmarshal(rootMessage.GetContent().Value, chatRequest)
			if err != nil {
				log.Println("read error:", err)
				break
			}

			log.Printf("RECV CHAT REQUEST!!: %s", chatRequest.GetMessage())

			chatResponse := root.ChatResponse{
				Message: chatRequest.GetMessage(),
			}
			chatResponseBytes, _ := proto.Marshal(&chatResponse)

			outgoingRoot := root.Root{
				Type:    MessageTypeChatResponse,
				Content: &root.Any{Value: chatResponseBytes},
			}

			outgoingBytes, err := proto.Marshal(&outgoingRoot)
			if err != nil {
				log.Println("write error:", err)
				break
			}

			err = c.WriteMessage(websocket.BinaryMessage, outgoingBytes)
			break
		}

		if err != nil {
			break
		}

		/*
			log.Printf("recv: %s", message)
			err = c.WriteMessage(mt, message)
			if err != nil {
				log.Println("write:", err)
				break
			}
		*/
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, "ws://"+r.Host+"/echo")
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	fs := http.FileServer(http.Dir("protos"))
	http.Handle("/protos/", http.StripPrefix("/protos/", fs))

	http.HandleFunc("/echo", echo)
	http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script src="https://cdn.jsdelivr.net/gh/dcodeIO/protobuf.js@6.8.6/dist/protobuf.min.js"></script>

<script>

var root;

var Root;
var ChatRequest;

protobuf.load("protos/root.proto", function(err, _root) {
root = _root;

Root = root.lookupType("root.Root")
ChatRequest = root.lookupType("root.ChatRequest")
ChatResponse = root.lookupType("root.ChatResponse")
Any = root.lookupType("root.Any")
})


window.addEventListener("load", function(evt) {
    var output = document.getElementById("output");
    var input = document.getElementById("input");
    var ws;
    var print = function(message) {
        var d = document.createElement("div");
        d.innerHTML = message;
        output.appendChild(d);
    };
    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }
        ws = new WebSocket("{{.}}");
		ws.binaryType = "arraybuffer";
        ws.onopen = function(evt) {
            print("OPEN");
        }
        ws.onclose = function(evt) {
            print("CLOSE");
            ws = null;
        }
        ws.onmessage = function(evt) {
			var root = Root.decode(new Uint8Array(evt.data));

			switch(root.type) {
				case 2:
					var chatResponse = ChatResponse.decode(root.content.value);
					print("RESPONSE: " + chatResponse.message);
				break;
			}

            
        }
        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
        return false;
    };
    document.getElementById("send").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        print("SEND: " + input.value);
		var chatRequest = ChatRequest.create({message: input.value});
		var any = Any.create({value: ChatRequest.encode(chatRequest).finish()})
		var root = Root.create({type: 1, content: any })
		var bytes = Root.encode(root).finish();

		console.log(root);

        ws.send(bytes);
        return false;
    };
    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.close();
        return false;
    };
});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<p>Click "Open" to create a connection to the server, 
"Send" to send a message to the server and "Close" to close the connection. 
You can change the message and send multiple times.
<p>
<form>
<button id="open">Open</button>
<button id="close">Close</button>
<p><input id="input" type="text" value="Hello world!">
<button id="send">Send</button>
</form>
</td><td valign="top" width="50%">
<div id="output"></div>
</td></tr></table>
</body>
</html>
`))
