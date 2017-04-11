package redis

import (
    "testing"
    "time"
    "github.com/docker/docker/daemon/logger"

    "os"
    "syscall"
)

const URL = "127.0.0.1"
const Port = "6379"

func Test_NewConnection_1(t *testing.T) {
    client, err := New(URL, Port, 0)
    if err != nil {
        t.Error(err)
    }
    res, err := client.HealthCheck()
    if err != nil {
        t.Error(err)
    }
    if res == "PONG" {
        t.Log(res)
    } else {
        t.Fatalf("the result is `%v`, we need `PONG`", res)
    }
    err = client.Close()
    if err != nil {
        t.Error(err)
    }
}

func Test_SetOffset_1(t *testing.T) {
    client, err := New(URL, Port, 0)
    if err != nil {
        t.Error(err)
    }
    defer client.Close()

    client.SetOffset("sss", 3)
}
func Test_SetOffset_2(t *testing.T) {
    client, err := New(URL, Port, 0)
    if err != nil {
        t.Error(err)
    }
    defer client.Close()

    msg := logger.Message{
        Line:      []byte("safdfa"),
        Source:    "filename",
        Timestamp: time.Now(),
    }
    t.Log(9)
    err = client.SetOffset(msg.Source, 9)
    if err != nil {
        t.Error(err)
    }
}
func Test_GetOffset_2(t *testing.T) {
    client, err := New(URL, Port, 0)
    if err != nil {
        t.Error(err)
    }
    defer client.Close()

    msg := logger.Message{
        Line:      []byte("safdfa"),
        Source:    "filename",
        Timestamp: time.Now(),
    }
    t.Log(9)
    offset, err := client.GetOffset(msg.Source)
    if err != nil {
        t.Error(err)
    }
    if offset == 9 {
        t.Log("Good!")
    } else {
        t.Errorf("did not match, we need:\n %v\n but got \n %v", 9, offset)

    }
}
func Test_SetFileStat_1(t *testing.T) {
    client, err := New(URL, Port, 0)
    if err != nil {
        t.Error(err)
    }
    defer client.Close()
    msg := logger.Message{
        Line:      []byte("safdfa"),
        Source:    "/tmp/ExmanProcessMutex",
        Timestamp: time.Now(),
    }

    finfo, err := os.Stat(msg.Source)
    if err != nil {
        t.Error(err)
    }

    s := finfo.Sys().(*syscall.Stat_t)
    err = client.SetFileStat(msg.Source, s)
    if err != nil {
        t.Error(err)
    }

}
func Test_GetFileStat_1(t *testing.T) {
    client, err := New(URL, Port, 0)
    if err != nil {
        t.Error(err)
    }
    defer client.Close()

    msg := logger.Message{
        Line:      []byte("safdfa"),
        Source:    "/tmp/ExmanProcessMutex",
        Timestamp: time.Now(),
    }

    finfo, err := os.Stat(msg.Source)
    if err != nil {
        t.Error(err)
    }

    res, err := client.GetFileStat(msg.Source)
    if err != nil {
        t.Error(err)
    }
    if *res != *finfo.Sys().(*syscall.Stat_t) {
        t.Error("not match")
        t.Error(res)
        t.Error(finfo.Sys().(*syscall.Stat_t))
    }
}
