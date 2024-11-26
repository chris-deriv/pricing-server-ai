package main

import (
    "log"
    "net/http"

    "pricingserver/internal/server"

    "github.com/gorilla/websocket"
    "pricingserver/internal/common/logging"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

func serveWs(hub *server.Hub, w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        logging.DebugLog("Upgrade error:", err)
        return
    }
    clientID := server.GenerateUniqueID()
    client := &server.Client{
        ID:        clientID,
        Conn:      conn,
        Send:      make(chan []byte, 256),
        Contracts: make(map[string]string),
        Hub:       hub,
    }
    hub.Register <- client
    go client.WritePump()
    go client.ReadPump()
}

func main() {
    hub := server.NewHub()
    go hub.Run()
    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        serveWs(hub, w, r)
    })

    addr := ":8080"
    logging.DebugLog("Server started on", addr)
    if err := http.ListenAndServe(addr, nil); err != nil {
        log.Fatal("ListenAndServe:", err)
    }
}