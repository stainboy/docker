package file

import (
    "testing"
    "fmt"
)

func Test_SetFileStat_1(t *testing.T) {

    f, _ := New()
    filename := "/tmp/ExmanProcessMutex"
    s, err := f.GetFileStat(filename)
    if (err != nil) {
        t.Error(err)
    }
    fmt.Println(s)
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
    f, _ := New()
    filename := "/tmp/ExmanProcessMutex"
    s, err := f.GetFileStat(filename)
    if err != nil {
        t.Error(err)
    }
    fmt.Println(s)
}
