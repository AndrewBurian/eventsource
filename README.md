# EventSource
Swiss Army Knife for SSE in Golang

## Semantic Versioning
This library is module-ready and versioned symantically. `master` branch tracks the latest unstable work, and the `vN` branches track stable releases.

# Up and running in 30 seconds
So you want to publish events to client that connect to your server?
```go
func main() {
	stream := eventsource.NewStream()

	go func(s *eventsource.Stream) {
		for {
			time.Sleep(time.Second)
			stream.Broadcast(eventsource.DataEvent("tick"))
		}
	}(stream)
	http.ListenAndServe(":8080", stream)
}
```
The `Stream` object implements an `http.Handler` for you so it can be registered directly to a server. Broadcast events to it and it'll forward them to every client that's connected to it.

`DataEvent` is shorthand for creating a new `*Event` object and assigning `Data()` to it

# More control on the `Stream`
You got it! What do you need?

## Multiplexing / Topics / Rooms / Channels
We call them "topics" but the gist is the same. All Clients always receive `Broadcast` events, but you can `Publish` events to a specific topic, and then only clients that have `Subscribe`d to that topic will receive the event.

```go
stream.Subscribe("weather", myClient)
stream.Publish("weather", weatherUpdateEvent)
stream.Broadcast(tornadoWarningEvent)
```

You can also just create multiple `Stream` objects much to the same effect, then only use `Broadcast`. Streams are cheap and run no background routines, so this is a valid pattern.

```go
weatherStream := eventstream.NewStream()
lotteryStream := eventstream.NewStream()

weatherStream.Register(clientPlanningHikes)
lotteryStream.Register(soonToBePoorClient)

lotteryStream.Broadcast(everyoneLosesEvent)
```

### Auto subscribe certain routes to topics
`Stream` implements an `http.Handler` but by default just registers clients for broadcasts.

Use `TopicHandler` to create another handler for that stream that will subscribe clients to topics as well as broadcasts.
```go
stream := eventsource.NewStream()
catsHandler := stream.TopicHandler([]string{"cat"})

http.ListenAndServe(":8080", catsHandler)
```

## Subscribe/Unsubscribe
Use the stream's `Register`, `Subscribe`, `Remove`, `Unsubscribe`, and `CloseTopic` functions to control which clients are registered where.

## Tell me when clients connect
Register a callback for the `Stream` to invoke every time a new client connects with `Stream.ClientConnectHook`. It'll give you a handle to the resulting Client and the http request that created it, letting you do whatever you please.

```go
stream := eventsource.NewStream()
stream.ClientConnectHook(func(r *http.Requset, c *eventsource.Client){
  fmt.Println("Recieved connection from", r.Host)
  fmt.Println("Hate that guy")
  stream.Remove(c)
  c.Shutdown()
})
```

The callback will be on the same goroutine as the incoming web request that created it, but the Client is live and functioning so it'll start receiving broadcasts and publications immediately before your callback has returned.

## Graceful shutdown
The stream's `Shutdown` command will unsubscribe and disconnect all connected clients. However the `Stream` itself is not running any background routines, and may continue to register new clients if it's still registered as an http handler.

## Get out of my way
Fine! The `Stream` object is entirely convenience. It runs no background routines and does no special handling. It just adds the topics abstraction and calls `NewClient` for you when it's connected to. Feel free not to use it.

# More control of the `Client`
You betcha.

## Create my own clients
Clients have to be created off an `http.ResponseWriter` that supports the `http.Flusher` and `http.CloseNotifier` interfaces. When creating a client, callers can optionally also pass the original `http.Request` being served, which helps determine which headers are appropriate to send in response.

`NewClient` _does_ kick off a background routine to handle sending events, so constructing an object literal will not work. This is done because it's assumed you will likely be calling `NewClient` on an http handler routine, and will likely not be doing any interesting work on that routine.

```go
func ServeHTTP(w http.ResponseWriter, r *http.Request) {
  client := eventsource.NewClient(w, r)
  if client == nil {
    http.Error(...)
    return
  }

  client.Wait()
}
```
Letting the http handler routine that created the Client return may cause the underlying connection to be closed by the server. Since `NewClient` does not block, use `Wait` to block until the client is shutdown.

## Shutdown the client
The client's `Shutdown` function terminates the background routine and marks the client as closed. It does not actually sever the connection. It does unblock any routines waiting on `Wait`, which assuming the main http handler routine was waiting there, will cause the connection to close as it returns.

Attempts to `Send` events to a client after it has been shutdown will result in an error

# More control of Events
Events are the most critical part of the library, and are the most versatile.

## Write my own events from scratch
Events implement the `io.ReadWriter` interface so that data can be written to it from practically any source. However the `Write` interface _does not_ write an entire event in wire format. It writes the provided buffer into `data:` sections in the resulting event.

```go
ev := &eventsource.Event{}
io.WriteString(ev, "This is an event! How exciting.")

fmt.Fprintf(ev, "This is the %d time I've had to update this readme...", 42)
```

If you'd like to write bytes exactly as you'd like them to be written to the wire, use the `WriteRaw` function.

`Read` on the other hand, _does_ return the event as it would have been written to the wire.

## Deep copy an event
You can use `Read` and `WriteRaw` to create a perfect deep copy, _however_ since this writes to the underlying data buffer and not to the "assembly area", any calls to `Data`, `ID`, `Type` etc will clobber the buffer and not give you the expected result.

```go
evData, _ := ioutil.ReadAll(oldEvent) // full wire format
newEvent.Write(evData)                // ... smushed into the `data:` section. Not what you wanted

io.Copy(newEvent, oldEvent)           // still not what you wanted

newEvent.WriteRaw(evData)             // that will work
newEvent.AppendData("Moar")           // ... and you ruined it
```

Use `Clone` to create a perfect deep copy that survives further mutation. Though this is less efficient in memory.

## Create events more easily
Since you will probably be creating more than just a few events, the `EventFactory` interface and a couple helper factories and functions have been provided to speed things up.

```go
// Create events of the same type
typeFact := &eventsource.EventTypeFactory{
  Type: "message",
}

// then with incrementing ID's
idFact := &eventsource.EventIdFactory{
  Next:    0,
  NewFact: typeFact,
}

// then generate as many events as you want with
// type: message, and an ID that increments
ev := idFact.New()
```
