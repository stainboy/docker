package file

import (
    "testing"
    "os"
    "syscall"
    "time"
    "github.com/docker/docker/daemon/logger"
)

const URL = "127.0.0.1"
const Port = "6379"

func Test_SetFileStat_1(t *testing.T) {
    f := New()
    filename := "/tmp/ExmanProcessMutex"
    s, err := f.GetFileStat(filename)
    if err != nil {
        t.Error(err)
    }
    t.Log(s)
    //client.SetFileStat(filename, s)
    //msg := logger.Message{
    //    Line:      []byte("safdfa"),
    //    Source:    "filename",
    //    Timestamp: time.Now(),
    //    //Partial:   false,
    //    //Stat_t: Stat_t{
    //    //    Dev:  333,
    //    //    Ino:  222,
    //    //    Size: 333,
    //    //},
    //}
    //client.SetFileStat(msg.Source, msg.Stat_t)
}
func Test_GetFileStat_1(t *testing.T) {
    f := New()
    filename := "/tmp/ExmanProcessMutex"
    s, err := f.GetFileStat(filename)
    if err != nil {
        t.Error(err)
    }
    t.Log(s)
}

func Test_Redis_SetOffset_1(t *testing.T) {
    f := New()
    err := f.SetRedis(URL, Port, 0)
    if err != nil {
        t.Error(err)
    }
    filename := "/tmp/test"
    err = f.Redis.SetOffset(filename, 0)
    if err != nil {
        t.Error(err)
    }
}
func Test_Redis_SetOffset_2(t *testing.T) {
    f := New()
    err := f.Redis.SetRedis(URL, Port, 0)
    if err != nil {
        t.Error(err)
    }

    msg := logger.Message{
        Line:      []byte("safdfa"),
        Source:    "filename",
        Timestamp: time.Now(),
    }
    t.Log(9)
    err = f.Redis.SetOffset(msg.Source, 9)
    if err != nil {
        t.Error(err)
    }
}
func Test_Redis_GetOffset_2(t *testing.T) {
    f := New()
    err := f.Redis.SetRedis(URL, Port, 0)
    if err != nil {
        t.Error(err)
    }

    msg := logger.Message{
        Line:      []byte("safdfa"),
        Source:    "filename",
        Timestamp: time.Now(),
    }
    t.Log(9)
    offset, err := f.Redis.GetOffset(msg.Source)
    if err != nil {
        t.Error(err)
    }
    if offset == 9 {
        t.Log("Good!")
    } else {
        t.Errorf("did not match, we need:\n %v\n but got \n %v", 9, offset)
    }
}
func Test_Redis_SetFileStat_1(t *testing.T) {
    f := New()
    err := f.Redis.SetRedis(URL, Port, 0)
    if err != nil {
        t.Error(err)
    }
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
    err = f.Redis.SetFileStat(msg.Source, s)
    if err != nil {
        t.Error(err)
    }

}
func Test_Redis_GetFileStat_1(t *testing.T) {
    f := New()
    err := f.Redis.SetRedis(URL, Port, 0)
    if err != nil {
        t.Error(err)
    }
    msg := logger.Message{
        Line:      []byte("safdfa"),
        Source:    "/tmp/ExmanProcessMutex",
        Timestamp: time.Now(),
    }

    finfo, err := os.Stat(msg.Source)
    if err != nil {
        t.Error(err)
    }

    res, err := f.Redis.GetFileStat(msg.Source)
    if err != nil {
        t.Error(err)
    }
    if *res != *finfo.Sys().(*syscall.Stat_t) {
        t.Error("not match")
        t.Error(res)
        t.Error(finfo.Sys().(*syscall.Stat_t))
    }
}
