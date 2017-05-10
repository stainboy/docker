package redis

import (
    "testing"
    "time"
    "github.com/docker/docker/daemon/logger"
)

const URL = "127.0.0.1"
const Port = "6379"

func Test_NewConnection_1(t *testing.T) {
    client, err := New(URL, Port, 0)
    if err != nil {
        t.Error(err)
    }
    msg := logger.Message{
        Line:      []byte("safdfa"),
        Source:    "filename",
        Timestamp: time.Now(),
    }
    err = client.SendMessage(&msg, false)
    if err != nil {
        t.Error(err)
    }
    //if res == "PONG" {
    //    t.Log(res)
    //} else {
    //    t.Fatalf("the result is `%v`, we need `PONG`", res)
    //}
    //err = client.Close()
    //if err != nil {
    //    t.Error(err)
    //}
}
